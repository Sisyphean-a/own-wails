package appsvc

import (
	"context"
	"time"

	"GistSync/internal/settings"
	"GistSync/internal/syncflow"
)

func (o *DefaultSyncOrchestrator) QuickUpload(ctx context.Context, req QuickUploadRequest) (QuickOperationResult, error) {
	data, profile, syncService, err := o.loadSyncContext(req.ProfileID)
	if err != nil {
		return QuickOperationResult{}, err
	}
	selected := enabledItemIDs(profile.Items)
	uploadResult, err := syncService.UploadProfile(ctx, syncflow.UploadProfileRequest{
		Profile: profile, MasterPassword: data.MasterPassword, SelectedItemIDs: selected,
	})
	if err != nil {
		return QuickOperationResult{}, err
	}
	return QuickOperationResult{
		OperationID: buildQuickOperationID("upload"),
		Action:      "upload",
		ProfileID:   profile.ID,
		SnapshotID:  uploadResult.SnapshotID,
		Summary:     QuickOperationSummary{Uploaded: uploadResult.Uploaded},
		Items:       []QuickOperationItem{},
		Conflicts:   []QuickOperationItem{},
	}, nil
}

func (o *DefaultSyncOrchestrator) QuickDownload(ctx context.Context, req QuickDownloadRequest) (QuickOperationResult, error) {
	data, profile, syncService, err := o.loadSyncContext(req.ProfileID)
	if err != nil {
		return QuickOperationResult{}, err
	}
	selected := enabledItemIDs(profile.Items)
	previewReq := syncflow.ApplySnapshotRequest{ProfileID: profile.ID, RestoreMode: profile.RestoreMode, RestoreRoot: profile.RestoreRoot, SelectedItemIDs: selected}
	conflicts, err := syncService.PreviewApplyConflicts(ctx, previewReq)
	if err != nil {
		return QuickOperationResult{}, err
	}
	snapshotID, err := latestSnapshotID(ctx, syncService, profile.ID)
	if err != nil {
		return QuickOperationResult{}, err
	}
	policy := normalizeConflictPolicy(req.ConflictPolicy)
	if policy == QuickConflictManual && len(conflicts) > 0 && len(req.OverwriteItemIDs) == 0 {
		return QuickOperationResult{
			OperationID:                buildQuickOperationID("download"),
			Action:                     "download",
			ProfileID:                  profile.ID,
			SnapshotID:                 snapshotID,
			RequiresConflictResolution: true,
			Summary:                    QuickOperationSummary{Conflicts: len(conflicts)},
			Conflicts:                  mapConflicts(conflicts),
			Items:                      []QuickOperationItem{},
		}, nil
	}
	overwriteIDs := resolveOverwriteIDs(policy, conflicts, req.OverwriteItemIDs)
	applyReq := syncflow.ApplySnapshotRequest{
		ProfileID: profile.ID, SnapshotID: snapshotID, MasterPassword: data.MasterPassword,
		RestoreMode: profile.RestoreMode, RestoreRoot: profile.RestoreRoot,
		SelectedItemIDs: selected, OverwriteItemIDs: overwriteIDs,
	}
	applied, err := syncService.ApplySnapshot(ctx, applyReq)
	if err != nil {
		return QuickOperationResult{}, err
	}
	return QuickOperationResult{
		OperationID: buildQuickOperationID("download"),
		Action:      "download",
		ProfileID:   profile.ID,
		SnapshotID:  snapshotID,
		Summary: QuickOperationSummary{
			Applied: applied.Applied, Skipped: applied.Skipped, Conflicts: len(conflicts),
			Errors: countApplyErrors(applied.Items),
		},
		Conflicts: mapConflicts(conflicts),
		Items:     mapApplyItems(applied.Items),
	}, nil
}

func enabledItemIDs(items []settings.ProfileItem) []string {
	selected := make([]string, 0, len(items))
	for _, item := range items {
		if item.Enabled {
			selected = append(selected, item.ID)
		}
	}
	return selected
}

func normalizeConflictPolicy(policy QuickConflictPolicy) QuickConflictPolicy {
	if policy == QuickConflictManual {
		return policy
	}
	return QuickConflictOverwriteAll
}

func resolveOverwriteIDs(policy QuickConflictPolicy, conflicts []syncflow.ApplyConflict, selected []string) []string {
	if policy == QuickConflictManual {
		return selected
	}
	ids := make([]string, 0, len(conflicts))
	for _, conflict := range conflicts {
		ids = append(ids, conflict.ItemID)
	}
	return ids
}

func latestSnapshotID(ctx context.Context, syncService SyncService, profileID string) (string, error) {
	snapshots, err := syncService.ListSnapshots(ctx, profileID)
	if err != nil {
		return "", err
	}
	if len(snapshots) == 0 {
		return "", nil
	}
	return snapshots[0].ID, nil
}

func mapConflicts(conflicts []syncflow.ApplyConflict) []QuickOperationItem {
	items := make([]QuickOperationItem, 0, len(conflicts))
	for _, conflict := range conflicts {
		items = append(items, QuickOperationItem{
			ItemID: conflict.ItemID, TargetPath: conflict.TargetPath, Status: "conflict",
			DiffPreview: conflict.DiffPreview, DiffStatus: conflict.DiffStatus,
			DiffLines: conflict.DiffLines, AddedLines: conflict.AddedLines, RemovedLines: conflict.RemovedLines,
		})
	}
	return items
}

func mapApplyItems(items []syncflow.ApplyItemResult) []QuickOperationItem {
	result := make([]QuickOperationItem, 0, len(items))
	for _, item := range items {
		result = append(result, QuickOperationItem{
			ItemID: item.ItemID, TargetPath: item.TargetPath, Status: item.Status, Reason: item.Reason,
		})
	}
	return result
}

func countApplyErrors(items []syncflow.ApplyItemResult) int {
	count := 0
	for _, item := range items {
		if item.Status == "error" {
			count++
		}
	}
	return count
}

func buildQuickOperationID(action string) string {
	return action + "-" + time.Now().UTC().Format("20060102T150405.000000000")
}
