package history

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestApproveDeletePlanBlocksUnapprovedExecution(t *testing.T) {
	service := newFixtureService(t)

	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testThreadID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	if _, err := service.ExecuteDeletePlan(ExecuteRequest{PlanPath: plan.PlanPath, Confirmed: true}); err == nil {
		t.Fatalf("ExecuteDeletePlan() should reject unapproved plan")
	}
	approved, err := service.ApproveDeletePlan(ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	if !fileExists(approved.ApprovedPlanPath) {
		t.Fatalf("approved plan missing: %s", approved.ApprovedPlanPath)
	}
	assertJSONContains(t, approved.ApprovedPlanPath, "\"approved\": true")
}

func TestBackupOnlyWritesArtifactsWithoutDeleting(t *testing.T) {
	service := newFixtureService(t)

	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testThreadID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	approved, err := service.ApproveDeletePlan(ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	result, err := service.ExecuteDeletePlan(ExecuteRequest{
		PlanPath:   approved.ApprovedPlanPath,
		Confirmed:  true,
		BackupOnly: true,
	})
	if err != nil {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
	if result.Mode != "backup-only" {
		t.Fatalf("Mode = %q", result.Mode)
	}
	if result.ManifestAfterPath != "" {
		t.Fatalf("ManifestAfterPath = %q", result.ManifestAfterPath)
	}
	if result.Verification.Status != "skipped" {
		t.Fatalf("Verification.Status = %q", result.Verification.Status)
	}
	assertThreadRestored(t, service, testThreadID)
}

func TestDeleteWithoutBackupSkipsRollbackArtifacts(t *testing.T) {
	service := newFixtureService(t)

	plan, err := service.BuildDeletePlan(BuildPlanRequest{ThreadIDs: []string{testThreadID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	approved, err := service.ApproveDeletePlan(ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	result, err := service.ExecuteDeletePlan(ExecuteRequest{
		PlanPath:   approved.ApprovedPlanPath,
		Confirmed:  true,
		SkipBackup: true,
	})
	if err != nil {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
	if result.RollbackJournalPath != "" {
		t.Fatalf("RollbackJournalPath = %q", result.RollbackJournalPath)
	}
	if len(result.Backups) != 0 {
		t.Fatalf("backup count = %d", len(result.Backups))
	}
	assertThreadDeleted(t, service, testThreadID)
}

func TestExportEvidencePackWritesIndex(t *testing.T) {
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
		PlanPath:   approved.ApprovedPlanPath,
		Confirmed:  true,
		BackupOnly: true,
	})
	if err != nil {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
	pack, err := service.ExportEvidencePack(EvidencePackRequest{
		RunID:               execResult.RunID,
		DeletePlanPath:      plan.PlanPath,
		ApprovedPlanPath:    execResult.ApprovedPlanPath,
		RollbackJournalPath: execResult.RollbackJournalPath,
		ExecResultPath:      execResult.ExecResultPath,
	})
	if err != nil {
		t.Fatalf("ExportEvidencePack() error = %v", err)
	}
	if !fileExists(pack.EvidencePackPath) {
		t.Fatalf("evidence pack missing: %s", pack.EvidencePackPath)
	}
	if len(pack.Artifacts) < 4 {
		t.Fatalf("artifact count = %d", len(pack.Artifacts))
	}
	assertJSONContains(t, pack.EvidencePackPath, "\"delete_plan\"")
}

func assertJSONContains(t *testing.T, path string, needle string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !strings.Contains(string(data), needle) {
		t.Fatalf("file does not contain %s", needle)
	}
}
