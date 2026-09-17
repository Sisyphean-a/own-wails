package appsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"GistSync/internal/settings"
	"GistSync/internal/syncflow"
)

type fakeStore struct {
	data settings.Data
}

func (s *fakeStore) Load() (settings.Data, error) {
	return s.data, nil
}

func (s *fakeStore) Save(data settings.Data) error {
	s.data = data
	return nil
}

type fakeSync struct {
	preview        []syncflow.ApplyConflict
	apply          syncflow.ApplySnapshotResult
	upload         syncflow.UploadProfileResult
	uploadReq      syncflow.UploadProfileRequest
	applyReq       syncflow.ApplySnapshotRequest
	previewReq     syncflow.ApplySnapshotRequest
	uploadCalls    int
	applyCalls     int
	previewCalls   int
	cloudProfiles  []settings.Profile
	listSnapResult []syncflow.SnapshotMeta
}

func (f *fakeSync) ListProfilesFromCloud(context.Context) ([]settings.Profile, error) {
	return f.cloudProfiles, nil
}

func (f *fakeSync) UploadProfile(_ context.Context, req syncflow.UploadProfileRequest) (syncflow.UploadProfileResult, error) {
	f.uploadCalls++
	f.uploadReq = req
	return f.upload, nil
}

func (f *fakeSync) ListSnapshots(context.Context, string) ([]syncflow.SnapshotMeta, error) {
	return f.listSnapResult, nil
}

func (f *fakeSync) ValidateMasterPassword(context.Context, string) (syncflow.MasterPasswordValidation, error) {
	return syncflow.MasterPasswordValidation{}, nil
}

func (f *fakeSync) PreviewApplyConflicts(_ context.Context, req syncflow.ApplySnapshotRequest) ([]syncflow.ApplyConflict, error) {
	f.previewCalls++
	f.previewReq = req
	return f.preview, nil
}

func (f *fakeSync) ApplySnapshot(_ context.Context, req syncflow.ApplySnapshotRequest) (syncflow.ApplySnapshotResult, error) {
	f.applyCalls++
	f.applyReq = req
	return f.apply, nil
}

func TestService_PullProfilesFromCloud_ReplacesExistingProfileWithNewCloudItems(t *testing.T) {
	store := &fakeStore{data: settings.Data{
		Token:           "t1",
		ActiveProfileID: "p1",
		Profiles: []settings.Profile{{
			ID:    "p1",
			Items: []settings.ProfileItem{{ID: "i1", Enabled: true}},
		}},
	}}
	remote := settings.Profile{
		ID: "p1",
		Items: []settings.ProfileItem{
			{ID: "i1", Enabled: true},
			{ID: "i2", Enabled: true},
		},
	}
	syncSvc := &fakeSync{cloudProfiles: []settings.Profile{remote}}
	svc := NewServiceWithDeps(store, func(token string) (SyncService, error) {
		if token != "t1" {
			t.Fatalf("unexpected token %q", token)
		}
		return syncSvc, nil
	}, func(prefix string) string { return prefix + "-id" })

	count, err := svc.PullProfilesFromCloud(context.Background())
	if err != nil {
		t.Fatalf("PullProfilesFromCloud returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("profile count mismatch: got %d want 1", count)
	}
	if got := store.data.Profiles[0].Items; len(got) != 2 || got[1].ID != "i2" {
		t.Fatalf("expected the newly added cloud item to replace local metadata, got %#v", got)
	}
	if !store.data.CloudBootstrapDone {
		t.Fatal("expected cloud profile sync to be persisted")
	}
}

func TestService_AddFilesToProfile_NormalizesRelativePath(t *testing.T) {
	store := &fakeStore{data: settings.Data{Profiles: []settings.Profile{{ID: "p1", Items: []settings.ProfileItem{}}}}}
	svc := NewServiceWithDeps(store, defaultSyncFactory, func(prefix string) string { return prefix + "-id" })

	err := svc.AddFilesToProfile(context.Background(), "p1", []string{`C:\Users\me\.gitconfig`})
	if err != nil {
		t.Fatalf("AddFilesToProfile returned error: %v", err)
	}
	item := store.data.Profiles[0].Items[0]
	if item.SourcePathTemplate == "{{HOME}}/.gitconfig" && item.RelativePath == ".gitconfig" {
		return
	}
	if item.RelativePath != "Users/me/.gitconfig" {
		t.Fatalf("relative path mismatch: %q", item.RelativePath)
	}
}

func TestService_AddFilesToProfile_UsesHomePlaceholder(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	if filepath.Dir(homeDir) == homeDir {
		t.Skip("home dir does not have a parent user directory")
	}

	store := &fakeStore{data: settings.Data{Profiles: []settings.Profile{{ID: "p1", Items: []settings.ProfileItem{}}}}}
	svc := NewServiceWithDeps(store, defaultSyncFactory, func(prefix string) string { return prefix + "-id" })
	sourcePath := filepath.Join(filepath.Dir(homeDir), "Fusi", ".gitconfig")

	err = svc.AddFilesToProfile(context.Background(), "p1", []string{sourcePath})
	if err != nil {
		t.Fatalf("AddFilesToProfile returned error: %v", err)
	}

	item := store.data.Profiles[0].Items[0]
	if item.SourcePathTemplate != "{{HOME}}/.gitconfig" {
		t.Fatalf("source path template mismatch: %q", item.SourcePathTemplate)
	}
	if item.RelativePath != ".gitconfig" {
		t.Fatalf("relative path mismatch: %q", item.RelativePath)
	}
}

func TestService_SetProfileItemsEnabled(t *testing.T) {
	store := &fakeStore{data: settings.Data{Profiles: []settings.Profile{
		{ID: "p1", Items: []settings.ProfileItem{
			{ID: "i1", Enabled: true},
			{ID: "i2", Enabled: true},
		}},
	}}}
	svc := NewServiceWithDeps(store, defaultSyncFactory, func(prefix string) string { return prefix + "-id" })

	if err := svc.SetProfileItemsEnabled(context.Background(), "p1", []string{"i2"}, false); err != nil {
		t.Fatalf("SetProfileItemsEnabled returned error: %v", err)
	}
	items := store.data.Profiles[0].Items
	if !items[0].Enabled {
		t.Fatalf("expected i1 to stay enabled")
	}
	if items[1].Enabled {
		t.Fatalf("expected i2 to be disabled")
	}
}

func TestService_DownloadSync_RequiresOverwrite(t *testing.T) {
	store := &fakeStore{data: settings.Data{Token: "t1", ActiveProfileID: "p1", Profiles: []settings.Profile{{ID: "p1"}}}}
	syncSvc := &fakeSync{preview: []syncflow.ApplyConflict{{ItemID: "i1", TargetPath: "/tmp/x"}}}
	svc := NewServiceWithDeps(store, func(token string) (SyncService, error) {
		if token != "t1" {
			t.Fatalf("unexpected token %q", token)
		}
		return syncSvc, nil
	}, func(prefix string) string { return prefix + "-id" })

	_, err := svc.DownloadSync(context.Background(), false)
	if !errors.Is(err, ErrOverwriteRequired) {
		t.Fatalf("expected ErrOverwriteRequired, got %v", err)
	}
}

func TestService_QuickUpload_UsesEnabledItemsOnly(t *testing.T) {
	store := &fakeStore{
		data: settings.Data{
			Token:           "t1",
			MasterPassword:  "pwd",
			ActiveProfileID: "p1",
			Profiles: []settings.Profile{{
				ID: "p1",
				Items: []settings.ProfileItem{
					{ID: "i1", Enabled: true},
					{ID: "i2", Enabled: false},
					{ID: "i3", Enabled: true},
				},
			}},
		},
	}
	syncSvc := &fakeSync{
		upload: syncflow.UploadProfileResult{SnapshotID: "s1", Uploaded: 2},
	}
	svc := NewServiceWithDeps(store, func(string) (SyncService, error) { return syncSvc, nil }, func(prefix string) string { return prefix + "-id" })

	result, err := svc.QuickUpload(context.Background(), QuickUploadRequest{ProfileID: "p1"})
	if err != nil {
		t.Fatalf("QuickUpload returned error: %v", err)
	}
	if syncSvc.uploadCalls != 1 {
		t.Fatalf("expected one upload call, got %d", syncSvc.uploadCalls)
	}
	if len(syncSvc.uploadReq.SelectedItemIDs) != 2 || syncSvc.uploadReq.SelectedItemIDs[0] != "i1" || syncSvc.uploadReq.SelectedItemIDs[1] != "i3" {
		t.Fatalf("selected item ids mismatch: %#v", syncSvc.uploadReq.SelectedItemIDs)
	}
	if result.SnapshotID != "s1" || result.Summary.Uploaded != 2 {
		t.Fatalf("quick upload result mismatch: %+v", result)
	}
}

func TestService_QuickDownload_UsesEnabledItemsOnly(t *testing.T) {
	store := &fakeStore{
		data: settings.Data{
			Token:           "t1",
			MasterPassword:  "pwd",
			ActiveProfileID: "p1",
			Profiles: []settings.Profile{{
				ID: "p1",
				Items: []settings.ProfileItem{
					{ID: "i1", Enabled: true},
					{ID: "i2", Enabled: false},
					{ID: "i3", Enabled: true},
				},
			}},
		},
	}
	syncSvc := &fakeSync{
		apply:          syncflow.ApplySnapshotResult{Applied: 2},
		listSnapResult: []syncflow.SnapshotMeta{{ID: "snap-1"}},
	}
	svc := NewServiceWithDeps(store, func(string) (SyncService, error) { return syncSvc, nil }, func(prefix string) string { return prefix + "-id" })

	_, err := svc.QuickDownload(context.Background(), QuickDownloadRequest{
		ProfileID:      "p1",
		ConflictPolicy: QuickConflictOverwriteAll,
	})
	if err != nil {
		t.Fatalf("QuickDownload returned error: %v", err)
	}
	if got := syncSvc.previewReq.SelectedItemIDs; len(got) != 2 || got[0] != "i1" || got[1] != "i3" {
		t.Fatalf("preview selected item ids mismatch: %#v", got)
	}
	if got := syncSvc.applyReq.SelectedItemIDs; len(got) != 2 || got[0] != "i1" || got[1] != "i3" {
		t.Fatalf("apply selected item ids mismatch: %#v", got)
	}
}

func TestService_QuickDownload_ManualReturnsConflictPrompt(t *testing.T) {
	store := &fakeStore{
		data: settings.Data{
			Token:           "t1",
			MasterPassword:  "pwd",
			ActiveProfileID: "p1",
			Profiles: []settings.Profile{{
				ID: "p1", RestoreMode: "original",
			}},
		},
	}
	syncSvc := &fakeSync{
		preview: []syncflow.ApplyConflict{{
			ItemID: "i1", TargetPath: "/tmp/a", DiffPreview: "--- local\n+++ remote\n", DiffStatus: "ready",
		}},
		listSnapResult: []syncflow.SnapshotMeta{{ID: "s-latest", CreatedAt: "2026-03-30T00:00:00Z"}},
	}
	svc := NewServiceWithDeps(store, func(string) (SyncService, error) { return syncSvc, nil }, func(prefix string) string { return prefix + "-id" })

	result, err := svc.QuickDownload(context.Background(), QuickDownloadRequest{
		ProfileID:        "p1",
		ConflictPolicy:   QuickConflictManual,
		OverwriteItemIDs: nil,
	})
	if err != nil {
		t.Fatalf("QuickDownload returned error: %v", err)
	}
	if !result.RequiresConflictResolution || len(result.Conflicts) != 1 {
		t.Fatalf("expected conflict resolution required, got %+v", result)
	}
	if result.Conflicts[0].DiffStatus != "ready" || result.Conflicts[0].DiffPreview == "" {
		t.Fatalf("expected diff fields mapped, got %+v", result.Conflicts[0])
	}
	if syncSvc.applyCalls != 0 {
		t.Fatalf("expected no apply call before manual confirmation, got %d", syncSvc.applyCalls)
	}
}

func TestService_QuickDownload_OverwriteAllApplies(t *testing.T) {
	store := &fakeStore{
		data: settings.Data{
			Token:           "t1",
			MasterPassword:  "pwd",
			ActiveProfileID: "p1",
			Profiles: []settings.Profile{{
				ID: "p1", RestoreMode: "original",
			}},
		},
	}
	syncSvc := &fakeSync{
		preview:        []syncflow.ApplyConflict{{ItemID: "i1", TargetPath: "/tmp/a"}, {ItemID: "i2", TargetPath: "/tmp/b"}},
		apply:          syncflow.ApplySnapshotResult{Applied: 2, Skipped: 0, Items: []syncflow.ApplyItemResult{{ItemID: "i1"}, {ItemID: "i2"}}},
		listSnapResult: []syncflow.SnapshotMeta{{ID: "s-latest", CreatedAt: "2026-03-30T00:00:00Z"}},
	}
	svc := NewServiceWithDeps(store, func(string) (SyncService, error) { return syncSvc, nil }, func(prefix string) string { return prefix + "-id" })

	result, err := svc.QuickDownload(context.Background(), QuickDownloadRequest{
		ProfileID:      "p1",
		ConflictPolicy: QuickConflictOverwriteAll,
	})
	if err != nil {
		t.Fatalf("QuickDownload returned error: %v", err)
	}
	if result.RequiresConflictResolution {
		t.Fatalf("did not expect manual conflict resolution: %+v", result)
	}
	if syncSvc.applyCalls != 1 {
		t.Fatalf("expected one apply call, got %d", syncSvc.applyCalls)
	}
	if len(syncSvc.applyReq.OverwriteItemIDs) != 2 {
		t.Fatalf("expected overwrite ids populated, got %#v", syncSvc.applyReq.OverwriteItemIDs)
	}
	if result.Summary.Applied != 2 || result.Summary.Conflicts != 2 {
		t.Fatalf("summary mismatch: %+v", result.Summary)
	}
}

func TestService_QuickDownload_ManualApplyWithSelectedOverwrite(t *testing.T) {
	store := &fakeStore{
		data: settings.Data{
			Token:           "t1",
			MasterPassword:  "pwd",
			ActiveProfileID: "p1",
			Profiles: []settings.Profile{{
				ID: "p1", RestoreMode: "original",
			}},
		},
	}
	syncSvc := &fakeSync{
		preview:        []syncflow.ApplyConflict{{ItemID: "i1", TargetPath: "/tmp/a"}, {ItemID: "i2", TargetPath: "/tmp/b"}},
		apply:          syncflow.ApplySnapshotResult{Applied: 1, Skipped: 1, Items: []syncflow.ApplyItemResult{{ItemID: "i1", Status: "applied"}, {ItemID: "i2", Status: "skipped"}}},
		listSnapResult: []syncflow.SnapshotMeta{{ID: "s-latest", CreatedAt: "2026-03-30T00:00:00Z"}},
	}
	svc := NewServiceWithDeps(store, func(string) (SyncService, error) { return syncSvc, nil }, func(prefix string) string { return prefix + "-id" })

	result, err := svc.QuickDownload(context.Background(), QuickDownloadRequest{
		ProfileID:        "p1",
		ConflictPolicy:   QuickConflictManual,
		OverwriteItemIDs: []string{"i1"},
	})
	if err != nil {
		t.Fatalf("QuickDownload returned error: %v", err)
	}
	if syncSvc.applyCalls != 1 {
		t.Fatalf("expected one apply call, got %d", syncSvc.applyCalls)
	}
	if len(syncSvc.applyReq.OverwriteItemIDs) != 1 || syncSvc.applyReq.OverwriteItemIDs[0] != "i1" {
		t.Fatalf("overwrite ids mismatch: %#v", syncSvc.applyReq.OverwriteItemIDs)
	}
	if result.Summary.Applied != 1 || result.Summary.Skipped != 1 {
		t.Fatalf("summary mismatch: %+v", result.Summary)
	}
}
