package app

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"RepoMirror/internal/config"
	"RepoMirror/internal/diff"
	"RepoMirror/internal/gitops"
	"RepoMirror/internal/model"
	"RepoMirror/internal/syncer"
)

type RepositoryInspector interface {
	ListBranches(root string) ([]model.BranchSummary, error)
	ListCommits(root string, revision string, offset int) (model.CommitPage, error)
	ResolveRevision(root string, revision string) (string, error)
	ExportRevision(root string, selection model.SourceSelection) (gitops.Snapshot, error)
	ResolveRepositoryRoot(path string) (string, error)
	ReadTargetStatus(repoPath string) (model.TargetRepositoryStatus, error)
	ReadTargetStatusFromRoot(root string) (model.TargetRepositoryStatus, error)
	ListWorktrees(root string) ([]model.WorktreeSummary, error)
	DescribeWorkingTree(repoPath string) (string, error)
	Commit(repoPath string, message string) error
	Push(repoPath string) error
}

type DifferenceCalculator interface {
	Calculate(request diff.Request) (diff.Result, error)
}

type RepositorySynchronizer interface {
	Sync(request syncer.Request) error
}

type Service struct {
	mu              sync.Mutex
	ctx             context.Context
	store           *config.Store
	selector        DirectorySelector
	inspector       RepositoryInspector
	commitGenerator CommitMessageGenerator
	differ          DifferenceCalculator
	synchronizer    RepositorySynchronizer
	config          model.AppConfig
	sourceSelection model.SourceSelection
}

func NewService(
	store *config.Store,
	selector DirectorySelector,
	inspector RepositoryInspector,
	differ DifferenceCalculator,
	synchronizer RepositorySynchronizer,
	initialConfig model.AppConfig,
) *Service {
	return &Service{
		store:           store,
		selector:        selector,
		inspector:       inspector,
		differ:          differ,
		synchronizer:    synchronizer,
		config:          initialConfig.WithDefaults(),
		sourceSelection: model.SourceSelection{Mode: "working"},
	}
}

func (s *Service) Startup(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

func (s *Service) Shutdown(ctx context.Context) {
	cfg := s.currentConfig()
	width, height := runtime.WindowGetSize(ctx)
	cfg.WindowWidth = width
	cfg.WindowHeight = height
	if err := s.saveConfig(cfg); err != nil {
		runtime.LogErrorf(ctx, "persist config on shutdown: %v", err)
	}
}

func (s *Service) LoadState() (model.DashboardState, error) {
	return s.buildState(s.currentConfig())
}

func (s *Service) Refresh() (model.DashboardState, error) {
	return s.buildState(s.currentConfig())
}

func (s *Service) SaveConfig() (model.DashboardState, error) {
	cfg := s.currentConfig()
	if err := s.saveConfig(cfg); err != nil {
		return model.DashboardState{}, err
	}
	return s.buildState(cfg)
}

func (s *Service) currentConfig() model.AppConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config
}

func (s *Service) updateConfig(mutator func(*model.AppConfig)) (model.AppConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.config
	mutator(&next)
	next = next.WithDefaults()
	if err := s.saveConfig(next); err != nil {
		return model.AppConfig{}, err
	}
	if next.ProjectA != s.config.ProjectA || next.ProjectB != s.config.ProjectB || next.Direction != s.config.Direction {
		s.sourceSelection = model.SourceSelection{Mode: "working"}
	}
	s.config = next
	return s.config, nil
}

func (s *Service) saveConfig(cfg model.AppConfig) error {
	return s.store.Save(cfg)
}
