package syncflow

import (
	"context"
	"fmt"
	"os"
	"time"

	"GistSync/internal/pathmap"
	"GistSync/internal/security"
	"GistSync/internal/settings"
)

func (s *Service) UploadProfile(ctx context.Context, req UploadProfileRequest) (UploadProfileResult, error) {
	if empty(req.MasterPassword) {
		return UploadProfileResult{}, ErrEmptyPassword
	}
	gistID, data, err := s.loadManifest(ctx)
	if err != nil {
		return UploadProfileResult{}, err
	}

	snapshot := manifestSnapshot{
		ID:        buildID("snapshot", req.Profile.ID),
		ProfileID: req.Profile.ID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	selectedSet := buildSelectedSet(req.SelectedItemIDs)
	for _, item := range req.Profile.Items {
		snapshotItem, uploadErr := s.uploadItem(ctx, gistID, req, item, selectedSet)
		if uploadErr != nil {
			return UploadProfileResult{}, uploadErr
		}
		if snapshotItem == nil {
			continue
		}
		snapshot.Items = append(snapshot.Items, *snapshotItem)
	}

	data.upsertProfile(req.Profile)
	data.Snapshots = append(data.Snapshots, snapshot)
	if err = s.saveManifest(ctx, gistID, data); err != nil {
		return UploadProfileResult{}, err
	}
	return UploadProfileResult{SnapshotID: snapshot.ID, Uploaded: len(snapshot.Items)}, nil
}

func (s *Service) uploadItem(
	ctx context.Context,
	gistID string,
	req UploadProfileRequest,
	item settings.ProfileItem,
	selectedSet map[string]bool,
) (*manifestSnapshotItem, error) {
	if !isSelected(item.ID, selectedSet) || !item.Enabled {
		return nil, nil
	}
	templatePath := pathmap.CompactHomePath(item.SourcePathTemplate)
	absolutePath, resolveErr := pathmap.ExpandHomePath(templatePath)
	if resolveErr != nil {
		return nil, resolveErr
	}
	raw, readErr := os.ReadFile(absolutePath)
	if readErr != nil {
		return nil, fmt.Errorf("read local file: %w", readErr)
	}
	encrypted, encErr := security.EncryptString(string(raw), req.MasterPassword)
	if encErr != nil {
		return nil, encErr
	}
	blob := buildBlobFileName(req.Profile.ID, item.ID, time.Now().UnixNano())
	if err := s.cloud.UpsertFile(ctx, UpsertFileRequest{GistID: gistID, FileName: blob, Content: encrypted}); err != nil {
		return nil, err
	}
	return &manifestSnapshotItem{
		ItemID:             item.ID,
		SourcePathTemplate: templatePath,
		RelativePath:       resolveRelativePath(templatePath, item.RelativePath),
		BlobFile:           blob,
	}, nil
}
