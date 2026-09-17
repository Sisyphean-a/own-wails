package gitops

import (
	"fmt"
	"path/filepath"
	"strings"

	"RepoMirror/internal/model"
)

func (s *Service) ListWorktrees(root string) ([]model.WorktreeSummary, error) {
	output, err := s.runner.Run(root, nil, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository worktrees: %w", err)
	}
	return parseWorktrees(output, root), nil
}

func parseWorktrees(output []byte, currentRoot string) []model.WorktreeSummary {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil
	}

	records := strings.Split(trimmed, "\n\n")
	worktrees := make([]model.WorktreeSummary, 0, len(records))
	currentClean := filepath.Clean(currentRoot)
	for _, record := range records {
		worktree, ok := parseWorktreeRecord(record, currentClean)
		if !ok {
			continue
		}
		worktrees = append(worktrees, worktree)
	}
	return worktrees
}

func parseWorktreeRecord(record string, currentRoot string) (model.WorktreeSummary, bool) {
	lines := strings.Split(record, "\n")
	if len(lines) == 0 {
		return model.WorktreeSummary{}, false
	}

	path, ok := strings.CutPrefix(strings.TrimSpace(lines[0]), "worktree ")
	if !ok {
		return model.WorktreeSummary{}, false
	}

	summary := model.WorktreeSummary{
		Path:      filepath.Clean(path),
		Name:      model.RepositoryName(path),
		Branch:    "HEAD",
		IsCurrent: samePath(path, currentRoot),
	}
	for _, line := range lines[1:] {
		branch, ok := parseWorktreeBranch(line)
		if ok {
			summary.Branch = branch
			break
		}
	}
	return summary, true
}

func parseWorktreeBranch(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if branch, ok := strings.CutPrefix(trimmed, "branch refs/heads/"); ok {
		return branch, true
	}
	if trimmed == "detached" {
		return "HEAD", true
	}
	return "", false
}

func samePath(left string, right string) bool {
	cleanLeft := filepath.Clean(left)
	cleanRight := filepath.Clean(right)
	if cleanLeft == "" || cleanRight == "" {
		return false
	}
	if filepath.Separator == '\\' {
		return strings.EqualFold(cleanLeft, cleanRight)
	}
	return cleanLeft == cleanRight
}
