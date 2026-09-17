package history

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildDeletePlanHandlesOneThousandFileClones(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	ids := make([]string, 0, 1000)
	for index := 0; index < 1000; index++ {
		id := fmt.Sprintf("20000000-0000-7000-8000-%012x", index)
		ids = append(ids, id)
		path := filepath.Join(paths.sessionsDir, "bulk", fmt.Sprintf("rollout-%s.jsonl", id))
		writeCloneRollout(t, path, id, testThreadID)
	}
	if err := os.MkdirAll(paths.archivedSessionsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	metrics := &historyScanMetrics{}
	service.scanMetrics = metrics

	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: ids})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	if plan.Summary.TargetCount != 1000 {
		t.Fatalf("TargetCount = %d", plan.Summary.TargetCount)
	}
	for _, target := range plan.Targets {
		if countRolloutStores(target) != 1 {
			t.Fatalf("target %s rollout store count = %d", target.Thread.ID, countRolloutStores(target))
		}
	}
	want := historyScanMetrics{
		transcriptWalks: 2, catalogDatabaseOpen: 1, planDatabaseOpens: 3,
		planJSONLScans: 2, planJSONScans: 2,
	}
	if *metrics != want {
		t.Fatalf("scan metrics = %#v", metrics)
	}
}

func TestFileClonePlanDeletesAndRestoresAllPathsWithoutTouchingSource(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	livePath := filepath.Join(paths.sessionsDir, "2026", "07", "01", "rollout-live-"+testCloneID+".jsonl")
	archivedPath := filepath.Join(paths.archivedSessionsDir, "rollout-archived-"+testCloneID+".jsonl")
	writeCloneRollout(t, livePath, testCloneID, testThreadID)
	writeCloneRollout(t, archivedPath, testCloneID, testThreadID)
	sourcePath := findThread(t, mustListThreads(t, service), testThreadID).RolloutPath

	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testCloneID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	if got := countRolloutStores(plan.Targets[0]); got != 2 {
		t.Fatalf("rollout store count = %d", got)
	}
	approved, err := service.ApproveDeletePlan(ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	executed, err := service.ExecuteDeletePlan(ExecuteRequest{PlanPath: approved.ApprovedPlanPath, Confirmed: true})
	if err != nil {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
	assertPathState(t, livePath, false)
	assertPathState(t, archivedPath, false)
	assertPathState(t, sourcePath, true)

	if _, err := service.RollbackExecution(RollbackRequest{JournalPath: executed.RollbackJournalPath}); err != nil {
		t.Fatalf("RollbackExecution() error = %v", err)
	}
	assertPathState(t, livePath, true)
	assertPathState(t, archivedPath, true)
	assertPathState(t, sourcePath, true)
}

func TestExecuteDeletePlanRejectsReplacedCloneTranscript(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	clonePath := filepath.Join(paths.sessionsDir, "rollout-clone-"+testCloneID+".jsonl")
	writeCloneRollout(t, clonePath, testCloneID, testThreadID)
	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testCloneID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	approved, err := service.ApproveDeletePlan(ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	writeCloneRollout(t, clonePath, testOtherID, testThreadID)

	if _, err := service.ExecuteDeletePlan(ExecuteRequest{PlanPath: approved.ApprovedPlanPath, Confirmed: true}); err == nil {
		t.Fatalf("ExecuteDeletePlan() should reject replaced transcript")
	}
	assertPathState(t, clonePath, true)
	assertPathState(t, findThread(t, mustListThreads(t, service), testThreadID).RolloutPath, true)
}

func TestExecuteDeletePlanRejectsChangedCloneIdentity(t *testing.T) {
	for _, clonedFrom := range []string{"", testOtherID} {
		t.Run(clonedFrom, func(t *testing.T) {
			service := newFixtureService(t)
			paths, err := service.resolvePaths()
			if err != nil {
				t.Fatalf("resolvePaths() error = %v", err)
			}
			clonePath := filepath.Join(paths.sessionsDir, "rollout-clone-"+testCloneID+".jsonl")
			writeCloneRollout(t, clonePath, testCloneID, testThreadID)
			approvedPath := approvedClonePlan(t, service, testCloneID)
			writeCloneRollout(t, clonePath, testCloneID, clonedFrom)

			if _, err := service.ExecuteDeletePlan(ExecuteRequest{PlanPath: approvedPath, Confirmed: true}); err == nil {
				t.Fatalf("ExecuteDeletePlan() should reject changed clone identity")
			}
			assertPathState(t, clonePath, true)
		})
	}
}

func TestExecuteDeletePlanRejectsUnknownDeleteFile(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	clonePath := filepath.Join(paths.sessionsDir, "rollout-clone-"+testCloneID+".jsonl")
	writeCloneRollout(t, clonePath, testCloneID, testThreadID)
	approvedPath := approvedClonePlan(t, service, testCloneID)
	document, err := loadPlanDocument(approvedPath)
	if err != nil {
		t.Fatalf("loadPlanDocument() error = %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.txt")
	writeFixtureFile(t, outsidePath, "keep")
	document.Targets[0].Stores = append(document.Targets[0].Stores, PlanStore{
		Store: "unexpected", Path: outsidePath, Action: "delete_file", Exists: true,
	})
	if err := writeJSON(approvedPath, document); err != nil {
		t.Fatalf("writeJSON() error = %v", err)
	}

	if _, err := service.ExecuteDeletePlan(ExecuteRequest{PlanPath: approvedPath, Confirmed: true}); err == nil {
		t.Fatalf("ExecuteDeletePlan() should reject unknown delete file")
	}
	assertPathState(t, outsidePath, true)
	assertPathState(t, clonePath, true)
}

func TestExecuteDeletePlanRejectsTamperedApprovedIntent(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*planDocument)
	}{
		{"target-id", func(document *planDocument) {
			document.Targets[0].Thread.ID = testOtherID
			document.Targets[0].Stores = nil
		}},
		{"stores", func(document *planDocument) { document.Targets[0].Stores = nil }},
		{"codex-home", func(document *planDocument) { document.CodexHome = filepath.Join(t.TempDir(), ".codex") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newFixtureService(t)
			paths, err := service.resolvePaths()
			if err != nil {
				t.Fatalf("resolvePaths() error = %v", err)
			}
			clonePath := filepath.Join(paths.sessionsDir, "rollout-clone-"+testCloneID+".jsonl")
			writeCloneRollout(t, clonePath, testCloneID, testThreadID)
			approvedPath := approvedClonePlan(t, service, testCloneID)
			document, err := loadPlanDocument(approvedPath)
			if err != nil {
				t.Fatalf("loadPlanDocument() error = %v", err)
			}
			test.mutate(&document)
			if err := writeJSON(approvedPath, document); err != nil {
				t.Fatalf("writeJSON() error = %v", err)
			}

			if _, err := service.ExecuteDeletePlan(ExecuteRequest{PlanPath: approvedPath, Confirmed: true}); err == nil {
				t.Fatalf("ExecuteDeletePlan() should reject tampered %s", test.name)
			}
			assertPathState(t, clonePath, true)
		})
	}
}

func TestBuildDeletePlanRejectsSymlinkCloneTranscript(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	targetPath := filepath.Join(t.TempDir(), "target.jsonl")
	writeCloneRollout(t, targetPath, testCloneID, testThreadID)
	linkPath := filepath.Join(paths.sessionsDir, "rollout-link-"+testCloneID+".jsonl")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if _, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testCloneID}}); err == nil {
		t.Fatalf("BuildDeletePlan() should reject symlink transcript")
	}
	assertPathState(t, targetPath, true)
}

func mustListThreads(t *testing.T, service *Service) []ThreadSummary {
	t.Helper()
	result, err := service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	return result.Items
}

func approvedClonePlan(t *testing.T, service *Service, id string) string {
	t.Helper()
	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{id}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	approved, err := service.ApproveDeletePlan(ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	return approved.ApprovedPlanPath
}

func countRolloutStores(target PlanTarget) int {
	count := 0
	for _, store := range target.Stores {
		if store.Store == "rollout_jsonl" {
			count++
		}
	}
	return count
}

func assertPathState(t *testing.T, path string, wantExists bool) {
	t.Helper()
	_, err := os.Stat(path)
	if wantExists && err != nil {
		t.Fatalf("expected path to exist %s: %v", path, err)
	}
	if !wantExists && !os.IsNotExist(err) {
		t.Fatalf("expected path to be absent %s: %v", path, err)
	}
}
