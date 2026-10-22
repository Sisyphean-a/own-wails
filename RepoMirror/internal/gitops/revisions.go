package gitops

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"RepoMirror/internal/model"
)

const commitPageSize = 40

func (s *Service) ResolveRevision(root string, revision string) (string, error) {
	if strings.TrimSpace(revision) == "" {
		return "", fmt.Errorf("请选择分支或输入提交号")
	}
	output, err := s.runner.Run(root, nil, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("无法读取版本 %q: %w", revision, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Service) ListBranches(root string) ([]model.BranchSummary, error) {
	output, err := s.runner.Run(root, nil, "for-each-ref", "--sort=refname", "--format=%(refname)%00%(refname:short)%00%(symref)", "refs/heads/", "refs/remotes/")
	if err != nil {
		return nil, fmt.Errorf("无法读取分支: %w", err)
	}
	branches := make([]model.BranchSummary, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Split(strings.TrimSuffix(line, "\r"), "\x00")
		if len(fields) == 3 && fields[0] != "" && fields[2] == "" {
			branches = append(branches, model.BranchSummary{Ref: fields[0], Name: fields[1]})
		}
	}
	return branches, nil
}

func (s *Service) ListCommits(root string, revision string, offset int) (model.CommitPage, error) {
	page := model.CommitPage{Commits: make([]model.CommitSummary, 0)}
	if offset < 0 {
		return page, fmt.Errorf("提交列表偏移量不能为负数")
	}
	hash, err := s.ResolveRevision(root, revision)
	if err != nil {
		return page, err
	}
	output, err := s.runner.Run(root, nil, "log", "--format=%H%x00%s%x00%an%x00%aI", "-z", "--max-count="+strconv.Itoa(commitPageSize+1), "--skip="+strconv.Itoa(offset), hash, "--")
	if err != nil {
		return page, fmt.Errorf("无法读取提交列表: %w", err)
	}
	fields := bytes.Split(bytes.TrimSuffix(output, []byte{0}), []byte{0})
	if len(output) == 0 {
		return page, nil
	}
	if len(fields)%4 != 0 {
		return page, fmt.Errorf("Git 提交列表格式无效")
	}
	for index := 0; index < len(fields); index += 4 {
		if len(page.Commits) == commitPageSize {
			page.HasMore = true
			break
		}
		page.Commits = append(page.Commits, model.CommitSummary{Hash: string(fields[index]), Subject: string(fields[index+1]), Author: string(fields[index+2]), Date: string(fields[index+3])})
	}
	return page, nil
}

func (s *Service) changedPaths(root string, hash string) ([]string, error) {
	parents, err := s.runner.Run(root, nil, "rev-list", "--parents", "-n", "1", hash)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(parents))
	args := []string{"diff-tree", "--no-commit-id", "--name-only", "-r", "--no-renames", "-z"}
	// Rule: 合并提交相对第一父提交；首次提交相对空树。
	if len(fields) > 1 {
		args = append(args, fields[1], hash, "--")
	} else {
		args = append(args, "--root", hash, "--")
	}
	output, err := s.runner.Run(root, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("无法读取提交变更: %w", err)
	}
	paths := make([]string, 0)
	for _, path := range strings.Split(string(output), "\x00") {
		if path != "" && !isProtectedPath(path) {
			if err := validateSnapshotPath(path); err != nil {
				return nil, err
			}
			paths = append(paths, path)
		}
	}
	return paths, nil
}
