package appsvc

import (
	"context"
	"errors"
	"fmt"

	"GistSync/internal/settings"
	"GistSync/internal/syncflow"
)

type DefaultSyncOrchestrator struct {
	store     SettingsStore
	buildSync SyncFactory
}

func NewDefaultSyncOrchestrator(store SettingsStore, buildSync SyncFactory) *DefaultSyncOrchestrator {
	return &DefaultSyncOrchestrator{store: store, buildSync: buildSync}
}

func (o *DefaultSyncOrchestrator) UploadProfile(ctx context.Context, profileID string, selectedItemIDs []string) (syncflow.UploadProfileResult, error) {
	data, profile, syncService, err := o.loadSyncContext(profileID)
	if err != nil {
		return syncflow.UploadProfileResult{}, err
	}
	return syncService.UploadProfile(ctx, syncflow.UploadProfileRequest{
		Profile: profile, MasterPassword: data.MasterPassword, SelectedItemIDs: selectedItemIDs,
	})
}

func (o *DefaultSyncOrchestrator) ListSnapshots(ctx context.Context, profileID string) ([]syncflow.SnapshotMeta, error) {
	data, err := o.store.Load()
	if err != nil {
		return nil, err
	}
	resolvedID := resolveProfileID(data, profileID)
	if empty(resolvedID) {
		return []syncflow.SnapshotMeta{}, nil
	}
	syncService, err := o.buildSync(data.Token)
	if err != nil {
		return nil, err
	}
	snapshots, err := syncService.ListSnapshots(ctx, resolvedID)
	if errors.Is(err, syncflow.ErrProfileNotFound) {
		return []syncflow.SnapshotMeta{}, nil
	}
	return snapshots, err
}

func (o *DefaultSyncOrchestrator) ValidateMasterPassword(ctx context.Context, token string, password string) (syncflow.MasterPasswordValidation, error) {
	syncService, err := o.buildSync(token)
	if err != nil {
		return syncflow.MasterPasswordValidation{}, err
	}
	return syncService.ValidateMasterPassword(ctx, password)
}

func (o *DefaultSyncOrchestrator) PreviewApplyConflicts(ctx context.Context, req syncflow.ApplySnapshotRequest) ([]syncflow.ApplyConflict, error) {
	data, _, syncService, err := o.loadSyncContext(req.ProfileID)
	if err != nil {
		return nil, err
	}
	if empty(req.MasterPassword) {
		req.MasterPassword = data.MasterPassword
	}
	return syncService.PreviewApplyConflicts(ctx, req)
}

func (o *DefaultSyncOrchestrator) ApplySnapshot(ctx context.Context, req syncflow.ApplySnapshotRequest) (syncflow.ApplySnapshotResult, error) {
	data, _, syncService, err := o.loadSyncContext(req.ProfileID)
	if err != nil {
		return syncflow.ApplySnapshotResult{}, err
	}
	if empty(req.MasterPassword) {
		req.MasterPassword = data.MasterPassword
	}
	return syncService.ApplySnapshot(ctx, req)
}

func (o *DefaultSyncOrchestrator) UploadSync(ctx context.Context) (string, error) {
	data, err := o.store.Load()
	if err != nil {
		return "", err
	}
	if empty(data.ActiveProfileID) {
		return "", syncflow.ErrProfileNotFound
	}
	result, err := o.UploadProfile(ctx, data.ActiveProfileID, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("上传完成，快照 %s，文件 %d 个", result.SnapshotID, result.Uploaded), nil
}

func (o *DefaultSyncOrchestrator) DownloadSync(ctx context.Context, overwrite bool) (string, error) {
	data, err := o.store.Load()
	if err != nil {
		return "", err
	}
	if empty(data.ActiveProfileID) {
		return "", syncflow.ErrProfileNotFound
	}
	conflicts, err := o.PreviewApplyConflicts(ctx, syncflow.ApplySnapshotRequest{ProfileID: data.ActiveProfileID})
	if err != nil {
		return "", err
	}
	overwriteIDs, overwriteRequired := buildOverwriteList(conflicts, overwrite)
	if overwriteRequired {
		return "", ErrOverwriteRequired
	}
	result, err := o.ApplySnapshot(ctx, syncflow.ApplySnapshotRequest{
		ProfileID:        data.ActiveProfileID,
		OverwriteItemIDs: overwriteIDs,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("下载完成，应用 %d，跳过 %d", result.Applied, result.Skipped), nil
}

func buildOverwriteList(conflicts []syncflow.ApplyConflict, overwrite bool) ([]string, bool) {
	if overwrite {
		ids := make([]string, 0, len(conflicts))
		for _, c := range conflicts {
			ids = append(ids, c.ItemID)
		}
		return ids, false
	}
	if len(conflicts) > 0 {
		return nil, true
	}
	return []string{}, false
}

func (o *DefaultSyncOrchestrator) loadSyncContext(profileID string) (settings.Data, settings.Profile, SyncService, error) {
	data, err := o.store.Load()
	if err != nil {
		return settings.Data{}, settings.Profile{}, nil, err
	}
	resolvedID := resolveProfileID(data, profileID)
	profile, ok := findProfile(data, resolvedID)
	if !ok {
		return settings.Data{}, settings.Profile{}, nil, syncflow.ErrProfileNotFound
	}
	syncService, err := o.buildSync(data.Token)
	if err != nil {
		return settings.Data{}, settings.Profile{}, nil, err
	}
	return data, *profile, syncService, nil
}
