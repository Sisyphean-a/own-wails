package history

import (
	"fmt"
	"path/filepath"

	"codex-history-manager/internal/discovery"
)

type executionPreparation struct {
	document         planDocument
	outputDir        string
	backups          []BackupArtifact
	journalPath      string
	approvedPlanPath string
}

func executeDeletePlan(paths codexPaths, request ExecuteRequest, document planDocument, newDiscovery func() *discovery.Service) (ExecuteResult, error) {
	preparation, err := prepareExecution(paths, request, document)
	if err != nil {
		return ExecuteResult{}, err
	}
	events := []JobEvent{}
	if request.BackupOnly {
		emit(&events, "backup", 0, len(preparation.document.Targets), "info", "已生成备份，未执行 destructive 改写", preparation.journalPath)
		verification := VerificationResult{
			Status:              "skipped",
			Summary:             "仅生成备份，未执行 destructive 改写",
			Success:             false,
			RemainingReferences: []VerificationFinding{},
		}
		return writeExecutionResult(request.PlanPath, "backup-only", preparation, nil, events, verification)
	}
	mutations, verification, runErr := runExecutionPipeline(paths, preparation.document, preparation.outputDir, &events, newDiscovery)
	if runErr != nil {
		if preparation.journalPath == "" {
			return ExecuteResult{}, runErr
		}
		rollbackEvents, rollbackErr := restoreBackups(preparation.journalPath)
		events = append(events, rollbackEvents...)
		if rollbackErr != nil {
			return ExecuteResult{}, fmt.Errorf("%w；自动回滚失败: %v", runErr, rollbackErr)
		}
		return ExecuteResult{}, runErr
	}
	return writeExecutionResult(request.PlanPath, "delete", preparation, mutations, events, verification)
}

func prepareExecution(paths codexPaths, request ExecuteRequest, document planDocument) (executionPreparation, error) {
	if !document.Approved {
		return executionPreparation{}, fmt.Errorf("删除计划尚未批准，请先生成 approved-plan.json")
	}
	if !request.Confirmed {
		return executionPreparation{}, fmt.Errorf("删除计划尚未确认")
	}
	if err := assertInactiveTargets(document.Targets); err != nil {
		return executionPreparation{}, err
	}
	if err := validatePlanDeletes(paths, document.Targets); err != nil {
		return executionPreparation{}, err
	}
	return writeExecutionPreparation(request.PlanPath, document, false, request.BackupOnly || !request.SkipBackup)
}

func assertInactiveTargets(targets []PlanTarget) error {
	for _, target := range targets {
		if err := assertTargetNotActive(target.Thread); err != nil {
			return err
		}
	}
	return nil
}

func writeExecutionPreparation(planPath string, document planDocument, rewritePlan bool, createBackup bool) (executionPreparation, error) {
	outputDir := filepath.Dir(planPath)
	backups := []BackupArtifact{}
	journalPath := ""
	var err error
	if createBackup {
		var journal rollbackJournal
		backups, journal, err = createBackups(outputDir, document)
		if err != nil {
			return executionPreparation{}, err
		}
		journalPath = filepath.Join(outputDir, "rollback-journal.json")
		if err := writeJSON(journalPath, journal); err != nil {
			return executionPreparation{}, err
		}
	}
	approvedPlanPath := planPath
	if rewritePlan {
		approvedPlanPath, err = writeApprovedPlan(outputDir, document)
		if err != nil {
			return executionPreparation{}, err
		}
	}
	return executionPreparation{
		document:         document,
		outputDir:        outputDir,
		backups:          backups,
		journalPath:      journalPath,
		approvedPlanPath: approvedPlanPath,
	}, nil
}

func writeApprovedPlan(outputDir string, document planDocument) (string, error) {
	approvedPlanPath := filepath.Join(outputDir, "approved-plan.json")
	document.Approved = true
	return approvedPlanPath, writeJSON(approvedPlanPath, document)
}

func runExecutionPipeline(paths codexPaths, document planDocument, outputDir string, events *[]JobEvent, newDiscovery func() *discovery.Service) ([]MutationResult, VerificationResult, error) {
	mutations, err := performExecution(paths, document.Targets, events)
	if err != nil {
		return nil, VerificationResult{Status: "fail", Summary: "执行中断", Success: false}, err
	}
	if err := checkpointMutatedDatabases(paths); err != nil {
		return nil, VerificationResult{Status: "fail", Summary: "数据库 checkpoint 失败", Success: false}, err
	}
	if err := writeAfterManifest(outputDir, newDiscovery); err != nil {
		return nil, VerificationResult{Status: "fail", Summary: "执行后复扫失败", Success: false}, err
	}
	verification, err := verifyDeletion(paths, document.Targets)
	if err != nil {
		return nil, VerificationResult{Status: "fail", Summary: "一致性校验失败", Success: false}, err
	}
	if !verification.Success {
		return nil, verification, fmt.Errorf("执行后仍残留 %d 处引用", len(verification.RemainingReferences))
	}
	return mutations, verification, nil
}

func writeExecutionResult(planPath string, mode string, preparation executionPreparation, mutations []MutationResult, events []JobEvent, verification VerificationResult) (ExecuteResult, error) {
	manifestAfterPath := filepath.Join(preparation.outputDir, "manifest-after.json")
	if mode == "backup-only" {
		manifestAfterPath = ""
	}
	if mutations == nil {
		mutations = []MutationResult{}
	}
	execDoc := execResultDocument{
		RunID:               preparation.document.RunID,
		PlanPath:            planPath,
		ApprovedPlanPath:    preparation.approvedPlanPath,
		RollbackJournalPath: preparation.journalPath,
		ManifestAfterPath:   manifestAfterPath,
		Backups:             preparation.backups,
		Mutations:           mutations,
		Events:              events,
		Verification:        verification,
	}
	execResultPath := filepath.Join(preparation.outputDir, "exec-result.json")
	if err := writeJSON(execResultPath, execDoc); err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{
		RunID:               preparation.document.RunID,
		Mode:                mode,
		PlanPath:            planPath,
		ApprovedPlanPath:    preparation.approvedPlanPath,
		RollbackJournalPath: preparation.journalPath,
		ExecResultPath:      execResultPath,
		ManifestAfterPath:   execDoc.ManifestAfterPath,
		Backups:             preparation.backups,
		Mutations:           mutations,
		Events:              events,
		Verification:        verification,
	}, nil
}
