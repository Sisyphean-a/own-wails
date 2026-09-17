package main

import (
	"os"
	"path/filepath"

	"codex-history-manager/internal/history"
)

func (a *App) ListHistoryThreads(request history.ListRequest) (history.ListResult, error) {
	return a.history.ListThreads(request)
}

func (a *App) BuildHistoryDeletePlan(request history.BuildPlanRequest) (history.PlanResult, error) {
	return a.history.BuildDeletePlan(request)
}

func (a *App) ApproveHistoryDeletePlan(request history.ApproveRequest) (history.ApproveResult, error) {
	return a.history.ApproveDeletePlan(request)
}

func (a *App) ExecuteHistoryDeletePlan(request history.ExecuteRequest) (history.ExecuteResult, error) {
	return a.history.ExecuteDeletePlan(request)
}

func (a *App) RollbackHistoryDelete(request history.RollbackRequest) (history.RollbackResult, error) {
	return a.history.RollbackExecution(request)
}

func (a *App) ExportHistoryEvidencePack(request history.EvidencePackRequest) (history.EvidencePackResult, error) {
	request = a.fillEvidenceArtifacts(request)
	return a.history.ExportEvidencePack(request)
}

func (a *App) fillEvidenceArtifacts(request history.EvidencePackRequest) history.EvidencePackRequest {
	if request.DiscoveryPath == "" || request.ManifestBeforePath == "" || request.DuplicateGroupsPath == "" {
		if scan, plan, err := a.rebuildReadOnlyArtifacts(); err == nil {
			request.DiscoveryPath = scan.DiscoveryPath
			request.ManifestBeforePath = scan.ManifestPath
			request.DuplicateGroupsPath = plan.DuplicateGroupsPath
		}
	}
	request.ArchitecturePath = repoDocPath(".codestable", "architecture", "INDEX.md")
	request.ContextPath = repoDocPath(".codestable", "requirements", "contexts", "history-governance.md")
	request.HistoryPath = repoDocPath(".codestable", "history", "2026-09.md")
	return request
}

func (a *App) rebuildReadOnlyArtifacts() (ScanResult, DeletePlanResult, error) {
	scan, err := a.RunReadOnlyScan()
	if err != nil {
		return ScanResult{}, DeletePlanResult{}, err
	}
	plan, err := a.BuildDeletePlan(scan.ManifestPath)
	if err != nil {
		return ScanResult{}, DeletePlanResult{}, err
	}
	return scan, plan, nil
}

func repoDocPath(parts ...string) string {
	current, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Wails 开发模式和打包后的进程可能从不同目录启动，沿父目录查找单仓库记忆入口。
	for {
		candidate := filepath.Join(append([]string{current}, parts...)...)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}
