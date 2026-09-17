package history

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codex-history-manager/internal/discovery"
)

type Service struct {
	now               func() time.Time
	userHomeDir       func() (string, error)
	newDiscovery      func() *discovery.Service
	codexHomeOverride string
	scanMetrics       *historyScanMetrics
}

func NewService() *Service {
	return &Service{
		now:               time.Now,
		userHomeDir:       os.UserHomeDir,
		newDiscovery:      discovery.NewService,
		codexHomeOverride: "",
	}
}

func (s *Service) SetCodexHomeOverride(root string) {
	s.codexHomeOverride = root
}

func (s *Service) CodexHomeOverride() string {
	return s.codexHomeOverride
}

func (s *Service) ListThreads(request ListRequest) (ListResult, error) {
	paths, err := s.resolvePaths()
	if err != nil {
		return ListResult{}, err
	}
	if err := validateDataModel(paths); err != nil {
		return ListResult{}, err
	}
	threads, total, warnings, err := listThreads(paths, request)
	if err != nil {
		return ListResult{}, err
	}
	limit := reportLimit(request.Limit, len(threads))
	return ListResult{
		CodexHome: paths.codexHome,
		Summary: ListSummary{
			Count:        len(threads),
			Limit:        limit,
			HasMore:      total > len(threads),
			WarningCount: len(warnings),
		},
		Items:    threads,
		Warnings: warnings,
	}, nil
}

func reportLimit(requested int, count int) int {
	if requested < 0 {
		return count
	}
	return effectiveLimit(requested)
}

func (s *Service) BuildDeletePlan(request BuildPlanRequest) (PlanResult, error) {
	paths, err := s.resolvePaths()
	if err != nil {
		return PlanResult{}, err
	}
	if err := validateDataModel(paths); err != nil {
		return PlanResult{}, err
	}
	targets, err := resolveTargets(paths, request.ThreadIDs)
	if err != nil {
		return PlanResult{}, err
	}
	if err := validateTargetTranscripts(paths, targets); err != nil {
		return PlanResult{}, err
	}
	runID := buildRunID(s.now().UTC())
	return buildDeletePlan(paths, targets, runID)
}

func (s *Service) ApproveDeletePlan(request ApproveRequest) (ApproveResult, error) {
	if request.PlanPath == "" {
		return ApproveResult{}, fmt.Errorf("缺少删除计划路径")
	}
	return approveDeletePlan(request.PlanPath)
}

func (s *Service) ExecuteDeletePlan(request ExecuteRequest) (ExecuteResult, error) {
	paths, err := s.resolvePaths()
	if err != nil {
		return ExecuteResult{}, err
	}
	if err := validateDataModel(paths); err != nil {
		return ExecuteResult{}, err
	}
	document, err := loadPlanDocument(request.PlanPath)
	if err != nil {
		return ExecuteResult{}, err
	}
	if err := validateApprovedPlan(paths, request.PlanPath, document); err != nil {
		return ExecuteResult{}, err
	}
	if err := validatePlanDeletes(paths, document.Targets); err != nil {
		return ExecuteResult{}, err
	}
	return executeDeletePlan(paths, request, document, s.newDiscovery)
}

func (s *Service) RollbackExecution(request RollbackRequest) (RollbackResult, error) {
	if request.JournalPath == "" {
		return RollbackResult{}, fmt.Errorf("缺少 rollback journal 路径")
	}
	return rollbackExecution(request.JournalPath)
}

func (s *Service) ExportEvidencePack(request EvidencePackRequest) (EvidencePackResult, error) {
	return exportEvidencePack(request)
}

func buildRunID(now time.Time) string {
	return fmt.Sprintf("%s-%09d", now.Format("20060102-150405"), now.Nanosecond())
}

func outputDirFor(runID string) string {
	return filepath.Join(os.TempDir(), "codex-history-manager", "history-runs", runID)
}
