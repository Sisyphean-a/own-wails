package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func buildDeletePlan(paths codexPaths, targets []ThreadSummary, runID string) (PlanResult, error) {
	outputDir := outputDirFor(runID)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return PlanResult{}, err
	}
	planTargets := make([]PlanTarget, 0, len(targets))
	planTargets, warnings, err := assemblePlanTargets(paths, targets)
	if err != nil {
		return PlanResult{}, err
	}
	sort.Slice(planTargets, func(i, j int) bool {
		return timestampAfter(planTargets[i].Thread.UpdatedAt, planTargets[j].Thread.UpdatedAt)
	})
	document := planDocument{
		RunID: runID, CodexHome: paths.codexHome, Approved: false,
		Summary: planSummary(planTargets, warnings), Targets: planTargets, Warnings: warnings,
	}
	planPath := filepath.Join(outputDir, "delete-plan.json")
	if err := writeJSON(planPath, document); err != nil {
		return PlanResult{}, err
	}
	return PlanResult{
		RunID: document.RunID, CodexHome: document.CodexHome, PlanPath: planPath,
		Summary: document.Summary, Targets: document.Targets, Warnings: document.Warnings,
	}, nil
}

func assemblePlanTargets(paths codexPaths, targets []ThreadSummary) ([]PlanTarget, []string, error) {
	planTargets := make([]PlanTarget, 0, len(targets))
	warnings := []string{}
	snapshots := indexShellSnapshots(paths.shellSnapshotsDir)
	storeGroups, err := planStoresBatch(paths, targets)
	if err != nil {
		return nil, nil, err
	}
	for _, target := range targets {
		rolloutStores, targetWarnings := planRolloutStores(target, snapshots)
		planTarget := PlanTarget{Thread: target, Stores: append(storeGroups[target.ID], rolloutStores...), Warnings: targetWarnings}
		planTargets = append(planTargets, planTarget)
		warnings = append(warnings, planTarget.Warnings...)
	}
	return planTargets, warnings, nil
}

func planRolloutStores(target ThreadSummary, snapshots snapshotIndex) ([]PlanStore, []string) {
	rolloutPaths := target.RolloutPaths
	if len(rolloutPaths) == 0 && target.RolloutPath != "" {
		rolloutPaths = []string{target.RolloutPath}
	}
	stores := make([]PlanStore, 0, len(rolloutPaths))
	warnings := []string{}
	if len(rolloutPaths) == 0 {
		warnings = append(warnings, fmt.Sprintf("会话 %s 缺少 rollout_path", target.ID))
	}
	for _, rolloutPath := range rolloutPaths {
		stores = append(stores, planDeleteFile(rolloutPath, "rollout_jsonl"))
	}
	for _, snapshot := range snapshots[target.ID] {
		stores = append(stores, planDeleteFile(snapshot, "shell_snapshot"))
	}
	return stores, warnings
}

func planDeleteFile(path string, store string) PlanStore {
	info, err := os.Stat(path)
	if err != nil {
		return PlanStore{Store: store, Path: path, Action: "delete_file", Detail: "delete file", Exists: false}
	}
	return PlanStore{
		Store:  store,
		Path:   path,
		Action: "delete_file",
		Detail: "delete file",
		Count:  info.Size(),
		Exists: true,
	}
}

func planSummary(targets []PlanTarget, warnings []string) PlanSummary {
	storeCount := 0
	for _, target := range targets {
		storeCount += len(target.Stores)
	}
	return PlanSummary{
		TargetCount:  len(targets),
		StoreCount:   storeCount,
		WarningCount: len(warnings),
	}
}
