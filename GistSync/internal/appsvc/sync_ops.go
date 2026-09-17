package appsvc

import (
	"context"

	"GistSync/internal/syncflow"
)

func (s *Service) UploadProfile(ctx context.Context, profileID string, selectedItemIDs []string) (syncflow.UploadProfileResult, error) {
	return s.syncer.UploadProfile(ctx, profileID, selectedItemIDs)
}

func (s *Service) ListSnapshots(ctx context.Context, profileID string) ([]syncflow.SnapshotMeta, error) {
	return s.syncer.ListSnapshots(ctx, profileID)
}

func (s *Service) ValidateMasterPassword(ctx context.Context, token string, password string) (syncflow.MasterPasswordValidation, error) {
	return s.syncer.ValidateMasterPassword(ctx, token, password)
}

func (s *Service) PreviewApplyConflicts(ctx context.Context, req syncflow.ApplySnapshotRequest) ([]syncflow.ApplyConflict, error) {
	return s.syncer.PreviewApplyConflicts(ctx, req)
}

func (s *Service) ApplySnapshot(ctx context.Context, req syncflow.ApplySnapshotRequest) (syncflow.ApplySnapshotResult, error) {
	return s.syncer.ApplySnapshot(ctx, req)
}

func (s *Service) UploadSync(ctx context.Context) (string, error) {
	return s.syncer.UploadSync(ctx)
}

func (s *Service) DownloadSync(ctx context.Context, overwrite bool) (string, error) {
	return s.syncer.DownloadSync(ctx, overwrite)
}

func (s *Service) QuickUpload(ctx context.Context, req QuickUploadRequest) (QuickOperationResult, error) {
	return s.syncer.QuickUpload(ctx, req)
}

func (s *Service) QuickDownload(ctx context.Context, req QuickDownloadRequest) (QuickOperationResult, error) {
	return s.syncer.QuickDownload(ctx, req)
}
