package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateTargetTranscripts(paths codexPaths, targets []ThreadSummary) error {
	for _, target := range targets {
		for _, path := range targetRolloutPaths(target) {
			if target.Registered && !fileExists(path) {
				continue
			}
			if err := validateTranscriptPath(paths, path, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePlanDeletes(paths codexPaths, targets []PlanTarget) error {
	for _, target := range targets {
		for _, store := range target.Stores {
			if store.Action != "delete_file" || !store.Exists {
				continue
			}
			if err := validateDeleteStore(paths, target.Thread, store); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDeleteStore(paths codexPaths, target ThreadSummary, store PlanStore) error {
	if store.Path == "" {
		return fmt.Errorf("删除文件路径不能为空: %s", store.Store)
	}
	switch store.Store {
	case "rollout_jsonl":
		return validateTranscriptPath(paths, store.Path, target)
	case "shell_snapshot":
		return validateShellSnapshot(paths, store.Path, target.ID)
	default:
		return fmt.Errorf("删除计划包含不允许的文件类型: %s", store.Store)
	}
}

func validateTranscriptPath(paths codexPaths, path string, target ThreadSummary) error {
	root, err := transcriptRootForPath(paths, path)
	if err != nil {
		return err
	}
	if err := validateRegularFile(root, path, "会话转录"); err != nil {
		return err
	}
	item, err := readTranscriptIdentity(path)
	if err != nil {
		return err
	}
	if item.id != target.ID {
		return fmt.Errorf("会话转录 ID 不匹配: 期望 %s，实际 %s，路径 %s", target.ID, item.id, path)
	}
	if target.IsClone && item.clonedFrom != target.ClonedFrom {
		return fmt.Errorf("会话转录克隆来源不匹配: 期望 %s，实际 %s，路径 %s", target.ClonedFrom, item.clonedFrom, path)
	}
	if target.IsClone && item.originalProvider != target.OriginalProvider {
		return fmt.Errorf("会话转录原始提供方不匹配: 期望 %s，实际 %s，路径 %s", target.OriginalProvider, item.originalProvider, path)
	}
	return nil
}

func validateShellSnapshot(paths codexPaths, path string, expectedID string) error {
	if !pathWithin(paths.shellSnapshotsDir, path) {
		return fmt.Errorf("shell 快照路径越出允许根目录: %s", path)
	}
	name := filepath.Base(path)
	if !strings.HasPrefix(name, expectedID+".") || !strings.HasSuffix(name, ".sh") {
		return fmt.Errorf("shell 快照与会话 ID 不匹配: %s", path)
	}
	return validateRegularFile(paths.shellSnapshotsDir, path, "shell 快照")
}

func validateRegularFile(root string, path string, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("读取%s失败 %s: %w", label, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%s不是普通文件: %s", label, path)
	}
	realRoot, err := resolvePhysicalPath(root)
	if err != nil {
		return fmt.Errorf("解析%s根目录失败 %s: %w", label, root, err)
	}
	realPath, err := resolvePhysicalPath(path)
	if err != nil {
		return fmt.Errorf("解析%s失败 %s: %w", label, path, err)
	}
	if !pathWithin(realRoot, realPath) {
		return fmt.Errorf("%s真实路径越出允许根目录: %s", label, path)
	}
	return nil
}

func transcriptRootForPath(paths codexPaths, path string) (string, error) {
	for _, root := range []string{paths.sessionsDir, paths.archivedSessionsDir} {
		if pathWithin(root, path) {
			return root, nil
		}
	}
	return "", fmt.Errorf("会话转录路径越出允许根目录: %s", path)
}

func pathWithin(root string, path string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil || relative == "." {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func readTranscriptIdentity(path string) (transcriptMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return transcriptMetadata{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return transcriptMetadata{}, fmt.Errorf("读取会话转录失败 %s: %w", path, err)
		}
		return transcriptMetadata{}, fmt.Errorf("会话转录缺少 session_meta: %s", path)
	}
	var record map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
		return transcriptMetadata{}, fmt.Errorf("会话转录首条记录无效 %s: %w", path, err)
	}
	item, ok := transcriptFromFirstRecord(record, path, false)
	if !ok {
		return transcriptMetadata{}, fmt.Errorf("会话转录首条记录不是有效 session_meta: %s", path)
	}
	return item, nil
}

func targetRolloutPaths(target ThreadSummary) []string {
	if len(target.RolloutPaths) > 0 {
		return target.RolloutPaths
	}
	return nonEmptyStrings(target.RolloutPath)
}

func rolloutPathsFromPlanTarget(target PlanTarget) []string {
	paths := []string{}
	for _, store := range target.Stores {
		if store.Store == "rollout_jsonl" && store.Action == "delete_file" && store.Path != "" {
			paths = appendUnique(paths, store.Path)
		}
	}
	return paths
}
