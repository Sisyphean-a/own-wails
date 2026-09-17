package syncflow

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"GistSync/internal/pathmap"
	"GistSync/internal/profileutil"
	"GistSync/internal/settings"
)

func buildSelectedSet(ids []string) map[string]bool {
	if len(ids) == 0 {
		return nil
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		if empty(id) {
			continue
		}
		set[id] = true
	}
	return set
}

func buildSet(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		if empty(id) {
			continue
		}
		set[id] = true
	}
	return set
}

func isSelected(itemID string, selectedSet map[string]bool) bool {
	if selectedSet == nil {
		return true
	}
	return selectedSet[itemID]
}

func shouldSkip(path string, overwrite bool) bool {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return err == nil && !overwrite
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write target file: %w", err)
	}
	return nil
}

func resolveTargetPath(item manifestSnapshotItem, mode string, restoreRoot string) (string, error) {
	if mode == restoreRooted {
		if empty(restoreRoot) {
			return "", ErrEmptyRestoreRoot
		}
		return filepath.Join(restoreRoot, filepath.FromSlash(resolveRelativePath(item.SourcePathTemplate, item.RelativePath))), nil
	}
	return pathmap.ExpandHomePath(pathmap.CompactHomePath(item.SourcePathTemplate))
}

func normalizeRelative(item settings.ProfileItem) string {
	return resolveRelativePath(item.SourcePathTemplate, item.RelativePath)
}

func resolveRelativePath(sourcePathTemplate string, relativePath string) string {
	templatePath := pathmap.CompactHomePath(sourcePathTemplate)
	normalized := profileutil.NormalizeRelativePath(templatePath)
	if !empty(normalized) {
		return normalized
	}
	if !empty(relativePath) {
		return relativePath
	}
	return profileutil.NormalizeRelativePath(sourcePathTemplate)
}

func chooseRestoreMode(requested string, profileDefault string) string {
	if requested == restoreRooted || requested == restoreOriginal {
		return requested
	}
	if profileDefault == restoreRooted {
		return restoreRooted
	}
	return restoreOriginal
}

func buildBlobFileName(profileID string, itemID string, now int64) string {
	key := fmt.Sprintf("%s|%s|%d", profileID, itemID, now)
	sum := sha256.Sum256([]byte(key))
	return "blob_" + hex.EncodeToString(sum[:]) + ".enc"
}

func buildID(prefix string, profileID string) string {
	return profileutil.GenerateScopedID(prefix, profileID)
}

func empty(value string) bool {
	return strings.TrimSpace(value) == ""
}
