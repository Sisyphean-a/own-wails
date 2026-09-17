package history

import "testing"

func TestListThreadsMergesSessionIndexTitle(t *testing.T) {
	service := newFixtureService(t)

	result, err := service.ListThreads(ListRequest{Limit: 10})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("ListThreads() count = %d", len(result.Items))
	}
	if result.Items[0].Title != "展示标题" {
		t.Fatalf("ListThreads() title = %q", result.Items[0].Title)
	}
	if result.Items[0].Source != "vscode" || result.Items[0].ModelProvider != "hi_code" {
		t.Fatalf("ListThreads() source/provider = %q/%q", result.Items[0].Source, result.Items[0].ModelProvider)
	}
}

func TestListThreadsWithoutLimitReturnsAll(t *testing.T) {
	service := newFixtureService(t)

	result, err := service.ListThreads(ListRequest{Limit: -1})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("ListThreads() count = %d", len(result.Items))
	}
	if result.Summary.HasMore {
		t.Fatalf("ListThreads() hasMore = true")
	}
	if result.Summary.Limit != len(result.Items) {
		t.Fatalf("ListThreads() limit = %d", result.Summary.Limit)
	}
}

func TestListThreadsAllowsNullThreadSource(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	db, err := openDatabase(paths.stateDB)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}
	defer db.Close()
	mustExec(t, db, `update threads set thread_source = null where id = ?`, testThreadID)

	result, err := service.ListThreads(ListRequest{Limit: 10})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if result.Items[0].ThreadSource != "" {
		t.Fatalf("ListThreads() threadSource = %q", result.Items[0].ThreadSource)
	}
}

func TestBuildDeletePlanCreatesArtifact(t *testing.T) {
	service := newFixtureService(t)

	result, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testThreadID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	if result.Summary.TargetCount != 1 {
		t.Fatalf("TargetCount = %d", result.Summary.TargetCount)
	}
	if !fileExists(result.PlanPath) {
		t.Fatalf("plan file missing: %s", result.PlanPath)
	}
	if len(result.Targets[0].Stores) < 6 {
		t.Fatalf("store count = %d", len(result.Targets[0].Stores))
	}
}

func TestExecuteDeletePlanAndRollback(t *testing.T) {
	service := newFixtureService(t)

	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testThreadID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	approved, err := service.ApproveDeletePlan(ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	execResult, err := service.ExecuteDeletePlan(ExecuteRequest{
		PlanPath:  approved.ApprovedPlanPath,
		Confirmed: true,
	})
	if err != nil {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
	if !execResult.Verification.Success {
		t.Fatalf("verification = %#v", execResult.Verification)
	}
	assertThreadDeleted(t, service, testThreadID)

	rollbackResult, err := service.RollbackExecution(RollbackRequest{JournalPath: execResult.RollbackJournalPath})
	if err != nil {
		t.Fatalf("RollbackExecution() error = %v", err)
	}
	if rollbackResult.RestoredCount == 0 {
		t.Fatalf("RestoredCount = %d", rollbackResult.RestoredCount)
	}
	assertThreadRestored(t, service, testThreadID)
}
