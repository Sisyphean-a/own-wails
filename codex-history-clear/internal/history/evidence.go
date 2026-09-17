package history

import (
	"fmt"
	"os"
	"path/filepath"
)

type evidencePackDocument struct {
	RunID     string                 `json:"run_id"`
	Artifacts []EvidencePackArtifact `json:"artifacts"`
}

func exportEvidencePack(request EvidencePackRequest) (EvidencePackResult, error) {
	artifacts, err := collectEvidenceArtifacts(request)
	if err != nil {
		return EvidencePackResult{}, err
	}
	outputDir, runID, err := evidencePackOutput(request, artifacts)
	if err != nil {
		return EvidencePackResult{}, err
	}
	packPath := filepath.Join(outputDir, "evidence-pack.json")
	document := evidencePackDocument{RunID: runID, Artifacts: artifacts}
	if err := writeJSON(packPath, document); err != nil {
		return EvidencePackResult{}, err
	}
	return EvidencePackResult{
		RunID:            runID,
		EvidencePackPath: packPath,
		Artifacts:        artifacts,
	}, nil
}

func collectEvidenceArtifacts(request EvidencePackRequest) ([]EvidencePackArtifact, error) {
	artifacts := []EvidencePackArtifact{}
	for _, item := range []EvidencePackArtifact{
		{Label: "discovery", Path: request.DiscoveryPath},
		{Label: "manifest_before", Path: request.ManifestBeforePath},
		{Label: "duplicate_groups", Path: request.DuplicateGroupsPath},
		{Label: "delete_plan", Path: request.DeletePlanPath},
		{Label: "approved_plan", Path: request.ApprovedPlanPath},
		{Label: "rollback_journal", Path: request.RollbackJournalPath},
		{Label: "exec_result", Path: request.ExecResultPath},
		{Label: "manifest_after", Path: request.ManifestAfterPath},
		{Label: "architecture", Path: request.ArchitecturePath},
		{Label: "context", Path: request.ContextPath},
		{Label: "history", Path: request.HistoryPath},
	} {
		if item.Path == "" {
			continue
		}
		if !fileExists(item.Path) {
			return nil, fmt.Errorf("证据文件不存在: %s", item.Path)
		}
		artifacts = append(artifacts, item)
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("缺少可导出的证据文件")
	}
	return artifacts, nil
}

func evidencePackOutput(request EvidencePackRequest, artifacts []EvidencePackArtifact) (string, string, error) {
	runID := request.RunID
	if runID == "" {
		runID = inferRunID(artifacts)
	}
	outputDir := outputDirFor(runID)
	if len(artifacts) > 0 {
		outputDir = filepath.Dir(artifacts[0].Path)
	}
	return outputDir, runID, ensureDir(outputDir)
}

func inferRunID(artifacts []EvidencePackArtifact) string {
	parent := filepath.Base(filepath.Dir(artifacts[0].Path))
	if parent == "history-runs" || parent == "." {
		return "external-run"
	}
	return parent
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
