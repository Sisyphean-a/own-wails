package history

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildAndExecuteDeduplicatePhysicalRolloutPath(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	thread := findThread(t, mustListThreads(t, service), testThreadID)
	realDir := filepath.Dir(thread.RolloutPath)
	aliasDir := filepath.Join(paths.sessionsDir, "junction-alias")
	createDirectoryAlias(t, aliasDir, realDir)
	aliasPath := filepath.Join(aliasDir, filepath.Base(thread.RolloutPath))
	if _, err := os.Stat(aliasPath); err != nil {
		t.Fatalf("Stat(alias) error = %v", err)
	}
	if resolved, err := resolvePhysicalPath(aliasPath); err != nil {
		t.Fatalf("resolvePhysicalPath(alias) error = %v", err)
	} else if !sameFilesystemPath(resolved, thread.RolloutPath) {
		t.Fatalf("junction resolved to unexpected path: %s", resolved)
	}
	db, err := openDatabase(paths.stateDB)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}
	mustExec(t, db, `update threads set rollout_path = ? where id = ?`, aliasPath, testThreadID)
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testThreadID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	if countRolloutStores(plan.Targets[0]) != 1 {
		t.Fatalf("rollout stores = %d", countRolloutStores(plan.Targets[0]))
	}
	approved, err := service.ApproveDeletePlan(ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	if _, err := service.ExecuteDeletePlan(ExecuteRequest{
		PlanPath: approved.ApprovedPlanPath, Confirmed: true, SkipBackup: true,
	}); err != nil {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
	assertPathState(t, thread.RolloutPath, false)
}

func TestDeleteRegisteredThreadWithMissingRollout(t *testing.T) {
	service := newFixtureService(t)
	thread := findThread(t, mustListThreads(t, service), testThreadID)
	if err := os.Remove(thread.RolloutPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	approvedPath := approvedClonePlan(t, service, testThreadID)
	if _, err := service.ExecuteDeletePlan(ExecuteRequest{
		PlanPath: approvedPath, Confirmed: true,
	}); err != nil {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
	assertThreadDeleted(t, service, testThreadID)
}

func createDirectoryAlias(t *testing.T, alias string, target string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		if err := os.Symlink(target, alias); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}
		return
	}
	output, err := exec.Command("cmd", "/c", "mklink", "/J", alias, target).CombinedOutput()
	if err != nil {
		t.Fatalf("create junction: %v: %s", err, output)
	}
}
