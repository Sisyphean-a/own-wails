package app

import (
	"errors"
	"fmt"
	"strings"

	"RepoMirror/internal/diff"
	"RepoMirror/internal/model"
	"RepoMirror/internal/syncer"
)

func (s *Service) currentSourceSelection() model.SourceSelection {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sourceSelection
}

func (s *Service) sourceRoot() (string, error) {
	cfg := s.currentConfig()
	return s.resolveRepository(cfg.PathFor(cfg.Direction.SourceSlot()), "源仓库")
}

func (s *Service) ListSourceBranches() ([]model.BranchSummary, error) {
	root, err := s.sourceRoot()
	if err != nil {
		return nil, err
	}
	return s.inspector.ListBranches(root)
}

func (s *Service) ListSourceCommits(revision string, offset int) (model.CommitPage, error) {
	root, err := s.sourceRoot()
	if err != nil {
		return model.CommitPage{}, err
	}
	return s.inspector.ListCommits(root, revision, offset)
}

func (s *Service) SelectSource(mode string, revision string) (model.DashboardState, error) {
	next := model.SourceSelection{Mode: mode}
	switch mode {
	case "working":
	case "branch", "commit":
		root, err := s.sourceRoot()
		if err != nil {
			return model.DashboardState{}, err
		}
		if mode == "branch" {
			branches, err := s.inspector.ListBranches(root)
			if err != nil {
				return model.DashboardState{}, err
			}
			found := false
			for _, branch := range branches {
				if branch.Ref == revision {
					found = true
					break
				}
			}
			if !found {
				return model.DashboardState{}, fmt.Errorf("分支不存在，请重新选择")
			}
		}
		hash, err := s.inspector.ResolveRevision(root, strings.TrimSpace(revision))
		if err != nil {
			return model.DashboardState{}, err
		}
		next.Ref = revision
		next.Commit = hash
	default:
		return model.DashboardState{}, fmt.Errorf("无效的差异来源: %s", mode)
	}
	previous := s.currentSourceSelection()
	s.mu.Lock()
	s.sourceSelection = next
	s.mu.Unlock()
	state, err := s.buildState(s.currentConfig())
	if err != nil {
		s.mu.Lock()
		s.sourceSelection = previous
		s.mu.Unlock()
	}
	return state, err
}

func (s *Service) prepareSourceRequest(request diff.Request) (diff.Request, func() error, error) {
	selection := s.currentSourceSelection()
	if selection.Mode == "working" {
		return request, func() error { return nil }, nil
	}
	snapshot, err := s.inspector.ExportRevision(request.SourceRoot, selection)
	if err != nil {
		return request, nil, err
	}
	request.SourceRoot = snapshot.Root
	request.SourceFiles = snapshot.Files
	request.ScopePaths = snapshot.Scope
	return request, snapshot.Close, nil
}

func (s *Service) calculateSourceDiff(request diff.Request) (result diff.Result, err error) {
	request, release, err := s.prepareSourceRequest(request)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, release()) }()
	return s.differ.Calculate(request)
}

func (s *Service) syncSource(sourceRoot string, targetRoot string) (err error) {
	request, release, err := s.prepareSourceRequest(diff.Request{SourceRoot: sourceRoot, TargetRoot: targetRoot})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, release()) }()
	return s.synchronizer.Sync(syncer.Request(request))
}
