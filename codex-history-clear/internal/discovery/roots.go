package discovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"codex-history-manager/internal/codexhome"
)

func (s *Service) resolveRoots() ([]string, error) {
	root, err := codexhome.Resolve(s.codexHomeOverride, s.userHomeDir, "扫描目录")
	if err != nil {
		return nil, err
	}
	return []string{root}, nil
}

func validateRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("扫描目录不存在: %s", root)
		}
		return fmt.Errorf("扫描目录不可用: %s: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("扫描目录不是文件夹: %s", root)
	}
	return nil
}

func (s *Service) collectItems(roots []string) ([]DiscoveryItem, []UnknownItem, error) {
	items := make([]DiscoveryItem, 0, len(roots))
	unknownItems := []UnknownItem{}
	for _, root := range roots {
		rootItems, rootUnknownItems, err := scanRoot(root)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, rootItems...)
		unknownItems = append(unknownItems, rootUnknownItems...)
	}
	sort.Slice(items, func(i, j int) bool {
		if strings.ToLower(items[i].SourceRoot) == strings.ToLower(items[j].SourceRoot) {
			return items[i].Path < items[j].Path
		}
		return strings.ToLower(items[i].SourceRoot) < strings.ToLower(items[j].SourceRoot)
	})
	sort.Slice(unknownItems, func(i, j int) bool {
		if strings.ToLower(unknownItems[i].SourceRoot) == strings.ToLower(unknownItems[j].SourceRoot) {
			return unknownItems[i].Path < unknownItems[j].Path
		}
		return strings.ToLower(unknownItems[i].SourceRoot) < strings.ToLower(unknownItems[j].SourceRoot)
	})
	return items, unknownItems, nil
}

func scanRoot(root string) ([]DiscoveryItem, []UnknownItem, error) {
	items := []DiscoveryItem{}
	unknownItems := []UnknownItem{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDirectory(root, path) {
				return fs.SkipDir
			}
			return nil
		}
		kind, ok := classifyPath(path)
		if !ok {
			if shouldTrackUnknown(root, path) {
				unknownItems = append(unknownItems, UnknownItem{
					SourceRoot: root,
					Path:       path,
					Kind:       "unclassified_candidate",
					Reason:     "candidate file did not match known discovery kinds",
				})
			}
			return nil
		}
		item, err := newDiscoveryItem(root, path, kind)
		if err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return items, unknownItems, nil
}

func classifyPath(path string) (string, bool) {
	name := strings.ToLower(filepath.Base(path))
	switch name {
	case "config.toml":
		return "config_toml", true
	case "auth.json":
		return "auth_json", true
	case "credentials.json":
		return "credentials_json", true
	case "history.jsonl":
		return "history_jsonl", true
	case "session_index.jsonl":
		return "session_index_jsonl", true
	}
	if kind, ok := classifySQLite(name); ok {
		return kind, true
	}
	if strings.HasPrefix(name, "rollout-") && strings.HasSuffix(name, ".jsonl") {
		return rolloutKind(path), true
	}
	return "", false
}

func classifySQLite(name string) (string, bool) {
	if !strings.HasSuffix(name, ".sqlite") {
		return "", false
	}
	switch {
	case strings.HasPrefix(name, "state"):
		return "state_sqlite", true
	case strings.HasPrefix(name, "logs"):
		return "logs_sqlite", true
	default:
		return "", false
	}
}

func rolloutKind(path string) string {
	lowerPath := strings.ToLower(path)
	archivedMarkers := []string{
		string(filepath.Separator) + "archive" + string(filepath.Separator),
		string(filepath.Separator) + "archives" + string(filepath.Separator),
		string(filepath.Separator) + "archived" + string(filepath.Separator),
		string(filepath.Separator) + "archived_sessions" + string(filepath.Separator),
	}
	for _, marker := range archivedMarkers {
		if strings.Contains(lowerPath, marker) {
			return "archived_rollout_jsonl"
		}
	}
	return "rollout_jsonl"
}

func shouldSkipDirectory(root string, path string) bool {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	relativePath = strings.ToLower(filepath.ToSlash(relativePath))
	if relativePath == "." {
		return false
	}
	firstSegment, _, _ := strings.Cut(relativePath, "/")
	switch firstSegment {
	case ".tmp", "plugins", "vendor_imports", "cache", "cxline":
		return true
	default:
		return false
	}
}

func shouldTrackUnknown(root string, path string) bool {
	if !isHistoryCandidatePath(root, path) {
		return false
	}
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(name, ".jsonl"):
		return true
	case strings.HasSuffix(name, ".json"):
		return true
	case strings.HasSuffix(name, ".sqlite"):
		return true
	case strings.HasSuffix(name, ".toml"):
		return true
	default:
		return false
	}
}

func isHistoryCandidatePath(root string, path string) bool {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	relativePath = strings.ToLower(filepath.ToSlash(relativePath))
	if !strings.Contains(relativePath, "/") {
		return true
	}
	return strings.HasPrefix(relativePath, "sessions/") ||
		strings.HasPrefix(relativePath, "archived_sessions/") ||
		strings.HasPrefix(relativePath, "sqlite/")
}

func newDiscoveryItem(sourceRoot string, path string, kind string) (DiscoveryItem, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return DiscoveryItem{}, err
	}
	linkType, target := linkMetadata(path, info)
	return DiscoveryItem{
		SourceRoot: sourceRoot,
		Path:       path,
		Kind:       kind,
		Size:       info.Size(),
		MTimeUTC:   info.ModTime().UTC().Format(time.RFC3339),
		Attributes: fileAttributes(info),
		LinkType:   linkType,
		Target:     target,
	}, nil
}
