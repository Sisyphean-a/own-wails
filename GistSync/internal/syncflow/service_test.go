package syncflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"GistSync/internal/security"
	"GistSync/internal/settings"
)

type fakeCloud struct {
	gistID string
	files  map[string]string
}

var fakeCloudCounter int64

func newFakeCloud() *fakeCloud {
	id := atomic.AddInt64(&fakeCloudCounter, 1)
	return &fakeCloud{
		gistID: "gist-" + strconv.FormatInt(id, 10),
		files: map[string]string{
			manifestFileName: `{"version":2,"profiles":[],"snapshots":[]}`,
		},
	}
}

func (f *fakeCloud) EnsureManifestGist(context.Context) (string, error) {
	return f.gistID, nil
}

func (f *fakeCloud) FindManifestGist(context.Context) (string, error) {
	return f.gistID, nil
}

func (f *fakeCloud) UpsertFile(_ context.Context, req UpsertFileRequest) error {
	f.files[req.FileName] = req.Content
	return nil
}

func (f *fakeCloud) GetFileContent(_ context.Context, req FileRequest) (string, error) {
	return f.files[req.FileName], nil
}

func TestService_UploadAndApplySnapshot(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	sourceFile := filepath.Join(t.TempDir(), "config.txt")
	if err := os.WriteFile(sourceFile, []byte("PM_Secret_Data"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	profile := settings.Profile{
		ID:          "profile-1",
		Name:        "Work",
		RestoreMode: restoreOriginal,
		Enabled:     true,
		Items: []settings.ProfileItem{
			{
				ID:                 "item-1",
				SourcePathTemplate: sourceFile,
				RelativePath:       "config.txt",
				Enabled:            true,
			},
		},
	}

	uploadResult, err := service.UploadProfile(context.Background(), UploadProfileRequest{
		Profile: profile, MasterPassword: "123",
	})
	if err != nil {
		t.Fatalf("UploadProfile returned error: %v", err)
	}
	if uploadResult.Uploaded != 1 {
		t.Fatalf("expected one uploaded file, got %d", uploadResult.Uploaded)
	}

	if err = os.WriteFile(sourceFile, []byte("Fake Data"), 0o600); err != nil {
		t.Fatalf("write fake data: %v", err)
	}
	conflicts, err := service.PreviewApplyConflicts(context.Background(), ApplySnapshotRequest{
		ProfileID: profile.ID, SnapshotID: uploadResult.SnapshotID,
	})
	if err != nil {
		t.Fatalf("PreviewApplyConflicts returned error: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected one conflict, got %d", len(conflicts))
	}

	applyResult, err := service.ApplySnapshot(context.Background(), ApplySnapshotRequest{
		ProfileID: profile.ID, SnapshotID: uploadResult.SnapshotID, MasterPassword: "123",
		OverwriteItemIDs: []string{"item-1"},
	})
	if err != nil {
		t.Fatalf("ApplySnapshot returned error: %v", err)
	}
	if applyResult.Applied != 1 {
		t.Fatalf("expected one applied item, got %d", applyResult.Applied)
	}

	raw, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("read target file: %v", err)
	}
	if string(raw) != "PM_Secret_Data" {
		t.Fatalf("content mismatch: %s", string(raw))
	}
}

func TestService_ValidateMasterPassword(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	sourceFile := filepath.Join(t.TempDir(), "config.txt")
	if err := os.WriteFile(sourceFile, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	profile := settings.Profile{
		ID: "profile-1", RestoreMode: restoreOriginal, Enabled: true,
		Items: []settings.ProfileItem{{
			ID: "item-1", SourcePathTemplate: sourceFile, RelativePath: "config.txt", Enabled: true,
		}},
	}
	if _, err := service.UploadProfile(context.Background(), UploadProfileRequest{Profile: profile, MasterPassword: "correct"}); err != nil {
		t.Fatalf("UploadProfile returned error: %v", err)
	}

	valid, err := service.ValidateMasterPassword(context.Background(), "correct")
	if err != nil {
		t.Fatalf("ValidateMasterPassword returned error: %v", err)
	}
	if !valid.HasSnapshot || !valid.Valid {
		t.Fatalf("expected valid snapshot password, got %+v", valid)
	}

	invalid, err := service.ValidateMasterPassword(context.Background(), "wrong")
	if err != nil {
		t.Fatalf("ValidateMasterPassword returned error: %v", err)
	}
	if !invalid.HasSnapshot || invalid.Valid {
		t.Fatalf("expected invalid snapshot password, got %+v", invalid)
	}
}

func TestService_ValidateMasterPasswordWithoutSnapshot(t *testing.T) {
	result, err := NewService(newFakeCloud()).ValidateMasterPassword(context.Background(), "password")
	if err != nil {
		t.Fatalf("ValidateMasterPassword returned error: %v", err)
	}
	if result.HasSnapshot || result.Valid {
		t.Fatalf("expected no validation result without snapshots, got %+v", result)
	}
}

func TestService_UploadProfile_WithSelectedItemIDs(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "a.txt")
	fileB := filepath.Join(tmpDir, "b.txt")
	if err := os.WriteFile(fileA, []byte("A"), 0o600); err != nil {
		t.Fatalf("write fileA: %v", err)
	}
	if err := os.WriteFile(fileB, []byte("B"), 0o600); err != nil {
		t.Fatalf("write fileB: %v", err)
	}
	profile := settings.Profile{
		ID:          "profile-selective-upload",
		Name:        "Work",
		RestoreMode: restoreOriginal,
		Enabled:     true,
		Items: []settings.ProfileItem{
			{ID: "item-a", SourcePathTemplate: fileA, RelativePath: "a.txt", Enabled: true},
			{ID: "item-b", SourcePathTemplate: fileB, RelativePath: "b.txt", Enabled: true},
		},
	}

	result, err := service.UploadProfile(context.Background(), UploadProfileRequest{
		Profile: profile, MasterPassword: "pwd", SelectedItemIDs: []string{"item-b"},
	})
	if err != nil {
		t.Fatalf("UploadProfile returned error: %v", err)
	}
	if result.Uploaded != 1 {
		t.Fatalf("expected one uploaded file, got %d", result.Uploaded)
	}

	var manifestData manifest
	if err = json.Unmarshal([]byte(cloud.files[manifestFileName]), &manifestData); err != nil {
		t.Fatalf("decode manifest failed: %v", err)
	}
	if len(manifestData.Snapshots) != 1 || len(manifestData.Snapshots[0].Items) != 1 {
		t.Fatalf("expected one snapshot item, got %#v", manifestData.Snapshots)
	}
	if manifestData.Snapshots[0].Items[0].ItemID != "item-b" {
		t.Fatalf("expected item-b uploaded, got %s", manifestData.Snapshots[0].Items[0].ItemID)
	}
}

func TestService_ListSnapshots(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	manifestRaw := `{"version":2,"profiles":[{"id":"profile-1","name":"Work","restoreMode":"original","restoreRoot":"","items":[]}],"snapshots":[{"id":"s1","profileId":"profile-1","createdAt":"2026-03-24T10:00:00Z","items":[]},{"id":"s2","profileId":"profile-1","createdAt":"2026-03-24T11:00:00Z","items":[]}]}`
	cloud.files[manifestFileName] = manifestRaw

	snaps, err := service.ListSnapshots(context.Background(), "profile-1")
	if err != nil {
		t.Fatalf("ListSnapshots returned error: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("snapshot count mismatch: %d", len(snaps))
	}
	if snaps[0].ID != "s2" {
		t.Fatalf("expected latest snapshot first, got %s", snaps[0].ID)
	}
}

func TestService_ListProfilesFromCloud(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	manifestRaw := `{"version":2,"profiles":[{"id":"profile-1","name":"Work","restoreMode":"original","restoreRoot":"","items":[{"id":"item-1","sourcePathTemplate":"{{HOME}}/.gitconfig","relativePath":".gitconfig","enabled":true}]}],"snapshots":[]}`
	cloud.files[manifestFileName] = manifestRaw

	profiles, err := service.ListProfilesFromCloud(context.Background())
	if err != nil {
		t.Fatalf("ListProfilesFromCloud returned error: %v", err)
	}
	if len(profiles) != 1 || len(profiles[0].Items) != 1 {
		t.Fatalf("unexpected cloud profiles: %#v", profiles)
	}
	if profiles[0].ID != "profile-1" {
		t.Fatalf("profile id mismatch: %s", profiles[0].ID)
	}
}

func TestService_ListProfilesFromCloud_RefreshesManifestOnEveryCall(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	cloud.files[manifestFileName] = `{"version":2,"profiles":[{"id":"profile-1","name":"Work","restoreMode":"original","restoreRoot":"","items":[{"id":"item-1","sourcePathTemplate":"{{HOME}}/.gitconfig","relativePath":".gitconfig","enabled":true}]}],"snapshots":[]}`

	profiles, err := service.ListProfilesFromCloud(context.Background())
	if err != nil {
		t.Fatalf("first ListProfilesFromCloud returned error: %v", err)
	}
	if len(profiles) != 1 || len(profiles[0].Items) != 1 {
		t.Fatalf("unexpected first cloud profiles: %#v", profiles)
	}

	cloud.files[manifestFileName] = `{"version":2,"profiles":[{"id":"profile-1","name":"Work","restoreMode":"original","restoreRoot":"","items":[{"id":"item-1","sourcePathTemplate":"{{HOME}}/.gitconfig","relativePath":".gitconfig","enabled":true},{"id":"item-2","sourcePathTemplate":"{{HOME}}/.ssh/config","relativePath":".ssh/config","enabled":true}]}],"snapshots":[]}`
	profiles, err = service.ListProfilesFromCloud(context.Background())
	if err != nil {
		t.Fatalf("second ListProfilesFromCloud returned error: %v", err)
	}
	if len(profiles) != 1 || len(profiles[0].Items) != 2 {
		t.Fatalf("expected the newly added cloud item, got %#v", profiles)
	}
}

func TestService_ApplySnapshotRootedMode(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	rootDir := t.TempDir()
	enc, err := securityEncrypt("hello", "pwd")
	if err != nil {
		t.Fatalf("encrypt helper failed: %v", err)
	}
	manifestData := manifest{
		Version: 2,
		Profiles: []manifestProfile{
			{ID: "p1", Name: "P1", RestoreMode: restoreRooted, Items: []manifestProfileItem{}},
		},
		Snapshots: []manifestSnapshot{
			{
				ID: "s1", ProfileID: "p1", CreatedAt: "2026-03-24T11:00:00Z",
				Items: []manifestSnapshotItem{
					{ItemID: "i1", SourcePathTemplate: "{{HOME}}/.gitconfig", RelativePath: ".gitconfig", BlobFile: "blob1.enc"},
				},
			},
		},
	}
	raw, _ := json.Marshal(manifestData)
	cloud.files[manifestFileName] = string(raw)
	cloud.files["blob1.enc"] = enc

	result, err := service.ApplySnapshot(context.Background(), ApplySnapshotRequest{
		ProfileID: "p1", SnapshotID: "s1", MasterPassword: "pwd",
		RestoreMode: restoreRooted, RestoreRoot: rootDir,
	})
	if err != nil {
		t.Fatalf("ApplySnapshot returned error: %v", err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected apply count 1, got %d", result.Applied)
	}
	targetPath := filepath.Join(rootDir, ".gitconfig")
	out, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read rooted file failed: %v", readErr)
	}
	if string(out) != "hello" {
		t.Fatalf("unexpected rooted content: %q", string(out))
	}
}

func TestService_ApplySnapshotOriginalMode_MapsForeignHomePathToCurrentHome(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	if filepath.Dir(homeDir) == homeDir {
		t.Skip("home dir does not have a parent user directory")
	}

	cloud := newFakeCloud()
	service := NewService(cloud)
	enc, err := securityEncrypt("hello", "pwd")
	if err != nil {
		t.Fatalf("encrypt helper failed: %v", err)
	}
	targetPath := filepath.Join(homeDir, ".ssh", "config")
	if err = os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("prepare target dir failed: %v", err)
	}
	if err = os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("cleanup target path failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(targetPath) })

	foreignPath := filepath.Join(filepath.Dir(homeDir), "Fusi", ".ssh", "config")
	manifestData := manifest{
		Version: 2,
		Profiles: []manifestProfile{
			{ID: "p1", Name: "P1", RestoreMode: restoreOriginal, Items: []manifestProfileItem{}},
		},
		Snapshots: []manifestSnapshot{
			{
				ID: "s1", ProfileID: "p1", CreatedAt: "2026-03-24T11:00:00Z",
				Items: []manifestSnapshotItem{
					{ItemID: "i1", SourcePathTemplate: foreignPath, RelativePath: "Users/Fusi/.ssh/config", BlobFile: "blob1.enc"},
				},
			},
		},
	}
	raw, _ := json.Marshal(manifestData)
	cloud.files[manifestFileName] = string(raw)
	cloud.files["blob1.enc"] = enc

	result, err := service.ApplySnapshot(context.Background(), ApplySnapshotRequest{
		ProfileID: "p1", SnapshotID: "s1", MasterPassword: "pwd",
	})
	if err != nil {
		t.Fatalf("ApplySnapshot returned error: %v", err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected apply count 1, got %d", result.Applied)
	}
	out, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read target file failed: %v", readErr)
	}
	if string(out) != "hello" {
		t.Fatalf("unexpected content: %q", string(out))
	}
}

func TestService_ApplySnapshot_WithSelectedItemIDs(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	rootDir := t.TempDir()
	fileA := filepath.Join(rootDir, "a.txt")
	fileB := filepath.Join(rootDir, "b.txt")
	if err := os.WriteFile(fileA, []byte("local-a"), 0o600); err != nil {
		t.Fatalf("write local fileA failed: %v", err)
	}
	encA, err := securityEncrypt("remote-a", "pwd")
	if err != nil {
		t.Fatalf("encrypt A failed: %v", err)
	}
	encB, err := securityEncrypt("remote-b", "pwd")
	if err != nil {
		t.Fatalf("encrypt B failed: %v", err)
	}
	manifestData := manifest{
		Version: 2,
		Profiles: []manifestProfile{
			{ID: "p1", Name: "P1", RestoreMode: restoreRooted, Items: []manifestProfileItem{}},
		},
		Snapshots: []manifestSnapshot{
			{
				ID: "s1", ProfileID: "p1", CreatedAt: "2026-03-24T11:00:00Z",
				Items: []manifestSnapshotItem{
					{ItemID: "i1", SourcePathTemplate: "{{HOME}}/a.txt", RelativePath: "a.txt", BlobFile: "blob1.enc"},
					{ItemID: "i2", SourcePathTemplate: "{{HOME}}/b.txt", RelativePath: "b.txt", BlobFile: "blob2.enc"},
				},
			},
		},
	}
	raw, _ := json.Marshal(manifestData)
	cloud.files[manifestFileName] = string(raw)
	cloud.files["blob1.enc"] = encA
	cloud.files["blob2.enc"] = encB

	conflicts, err := service.PreviewApplyConflicts(context.Background(), ApplySnapshotRequest{
		ProfileID: "p1", SnapshotID: "s1", RestoreMode: restoreRooted, RestoreRoot: rootDir, SelectedItemIDs: []string{"i1"},
	})
	if err != nil {
		t.Fatalf("PreviewApplyConflicts returned error: %v", err)
	}
	if len(conflicts) != 1 || conflicts[0].ItemID != "i1" {
		t.Fatalf("expected only i1 conflict, got %#v", conflicts)
	}

	result, err := service.ApplySnapshot(context.Background(), ApplySnapshotRequest{
		ProfileID: "p1", SnapshotID: "s1", MasterPassword: "pwd", RestoreMode: restoreRooted,
		RestoreRoot: rootDir, SelectedItemIDs: []string{"i2"},
	})
	if err != nil {
		t.Fatalf("ApplySnapshot returned error: %v", err)
	}
	if result.Applied != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}

	rawB, err := os.ReadFile(fileB)
	if err != nil {
		t.Fatalf("read fileB failed: %v", err)
	}
	if string(rawB) != "remote-b" {
		t.Fatalf("expected fileB to be updated, got %q", string(rawB))
	}
	rawA, err := os.ReadFile(fileA)
	if err != nil {
		t.Fatalf("read fileA failed: %v", err)
	}
	if string(rawA) != "local-a" {
		t.Fatalf("expected fileA unchanged, got %q", string(rawA))
	}
}

func TestService_PreviewApplyConflicts_IncludesTextDiffPreview(t *testing.T) {
	cloud := newFakeCloud()
	service := NewService(cloud)
	rootDir := t.TempDir()
	targetPath := filepath.Join(rootDir, "config.txt")
	if err := os.WriteFile(targetPath, []byte("line-1\nlocal-line\n"), 0o600); err != nil {
		t.Fatalf("write local file failed: %v", err)
	}
	encrypted, err := securityEncrypt("line-1\nremote-line\n", "pwd")
	if err != nil {
		t.Fatalf("encrypt remote content failed: %v", err)
	}
	manifestData := manifest{
		Version: 2,
		Profiles: []manifestProfile{
			{ID: "p1", Name: "P1", RestoreMode: restoreRooted, Items: []manifestProfileItem{}},
		},
		Snapshots: []manifestSnapshot{
			{
				ID: "s1", ProfileID: "p1", CreatedAt: "2026-03-24T11:00:00Z",
				Items: []manifestSnapshotItem{
					{ItemID: "i1", SourcePathTemplate: "{{HOME}}/config.txt", RelativePath: "config.txt", BlobFile: "blob1.enc"},
				},
			},
		},
	}
	raw, _ := json.Marshal(manifestData)
	cloud.files[manifestFileName] = string(raw)
	cloud.files["blob1.enc"] = encrypted

	conflicts, err := service.PreviewApplyConflicts(context.Background(), ApplySnapshotRequest{
		ProfileID: "p1", SnapshotID: "s1", MasterPassword: "pwd",
		RestoreMode: restoreRooted, RestoreRoot: rootDir,
	})
	if err != nil {
		t.Fatalf("PreviewApplyConflicts returned error: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected one conflict, got %d", len(conflicts))
	}
	if conflicts[0].DiffStatus != "ready" {
		t.Fatalf("expected diff status ready, got %q", conflicts[0].DiffStatus)
	}
	if conflicts[0].DiffPreview == "" {
		t.Fatalf("expected non-empty diff preview")
	}
	if !containsAll(conflicts[0].DiffPreview, []string{"--- local", "+++ remote", "-local-line", "+remote-line"}) {
		t.Fatalf("unexpected diff preview: %s", conflicts[0].DiffPreview)
	}
}

func containsAll(raw string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(raw, part) {
			return false
		}
	}
	return true
}

func securityEncrypt(data string, password string) (string, error) {
	return security.EncryptString(data, password)
}
