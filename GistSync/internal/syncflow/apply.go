package syncflow

import (
	"context"

	"GistSync/internal/security"
)

func (s *Service) PreviewApplyConflicts(ctx context.Context, req ApplySnapshotRequest) ([]ApplyConflict, error) {
	gistID, data, err := s.loadManifest(ctx)
	if err != nil {
		return nil, err
	}
	profile, snap, err := data.findProfileAndSnapshot(req.ProfileID, req.SnapshotID)
	if err != nil {
		return nil, err
	}
	restoreMode := chooseRestoreMode(req.RestoreMode, profile.RestoreMode)
	if restoreMode == restoreRooted && empty(req.RestoreRoot) {
		return nil, ErrEmptyRestoreRoot
	}

	var conflicts []ApplyConflict
	selectedSet := buildSelectedSet(req.SelectedItemIDs)
	for _, item := range snap.Items {
		if !isSelected(item.ItemID, selectedSet) {
			continue
		}
		targetPath, targetErr := resolveTargetPath(item, restoreMode, req.RestoreRoot)
		if targetErr != nil {
			return nil, targetErr
		}
		if fileExists(targetPath) {
			conflicts = append(conflicts, s.buildConflict(ctx, gistID, item, req, targetPath))
		}
	}
	return conflicts, nil
}

func (s *Service) ApplySnapshot(ctx context.Context, req ApplySnapshotRequest) (ApplySnapshotResult, error) {
	if empty(req.MasterPassword) {
		return ApplySnapshotResult{}, ErrEmptyPassword
	}
	gistID, data, err := s.loadManifest(ctx)
	if err != nil {
		return ApplySnapshotResult{}, err
	}
	profile, snap, err := data.findProfileAndSnapshot(req.ProfileID, req.SnapshotID)
	if err != nil {
		return ApplySnapshotResult{}, err
	}
	restoreMode := chooseRestoreMode(req.RestoreMode, profile.RestoreMode)
	if restoreMode == restoreRooted && empty(req.RestoreRoot) {
		return ApplySnapshotResult{}, ErrEmptyRestoreRoot
	}

	overwriteSet := buildSet(req.OverwriteItemIDs)
	selectedSet := buildSelectedSet(req.SelectedItemIDs)
	result := ApplySnapshotResult{}
	for _, item := range snap.Items {
		if !isSelected(item.ItemID, selectedSet) {
			continue
		}
		result = s.applySnapshotItem(ctx, result, gistID, item, req, restoreMode, overwriteSet)
	}
	return result, nil
}

func (s *Service) applySnapshotItem(
	ctx context.Context,
	result ApplySnapshotResult,
	gistID string,
	item manifestSnapshotItem,
	req ApplySnapshotRequest,
	restoreMode string,
	overwriteSet map[string]bool,
) ApplySnapshotResult {
	itemResult := ApplyItemResult{ItemID: item.ItemID}
	targetPath, targetErr := resolveTargetPath(item, restoreMode, req.RestoreRoot)
	if targetErr != nil {
		return appendSkipped(result, itemResult, "error", targetErr.Error())
	}
	itemResult.TargetPath = targetPath
	if shouldSkip(targetPath, overwriteSet[item.ItemID]) {
		return appendSkipped(result, itemResult, "skipped", "target exists and overwrite not granted")
	}
	encrypted, readErr := s.cloud.GetFileContent(ctx, FileRequest{GistID: gistID, FileName: item.BlobFile})
	if readErr != nil {
		return appendSkipped(result, itemResult, "error", readErr.Error())
	}
	decrypted, decErr := security.DecryptString(encrypted, req.MasterPassword)
	if decErr != nil {
		return appendSkipped(result, itemResult, "error", decErr.Error())
	}
	if writeErr := writeFile(targetPath, decrypted); writeErr != nil {
		return appendSkipped(result, itemResult, "error", writeErr.Error())
	}
	itemResult.Status = "applied"
	result.Items = append(result.Items, itemResult)
	result.Applied++
	return result
}

func appendSkipped(result ApplySnapshotResult, item ApplyItemResult, status string, reason string) ApplySnapshotResult {
	item.Status = status
	item.Reason = reason
	result.Items = append(result.Items, item)
	result.Skipped++
	return result
}
