package pathmap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const homePlaceholder = "{{HOME}}"

var errEmptyPath = errors.New("path cannot be empty")

func ExpandHomePath(path string) (string, error) {
	if path == "" {
		return "", errEmptyPath
	}
	if !strings.Contains(path, homePlaceholder) {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	replaced := strings.Replace(path, homePlaceholder, homeDir, 1)
	return filepath.Clean(replaced), nil
}

func CompactHomePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return path
	}
	if strings.Contains(path, homePlaceholder) {
		return normalizePlaceholderPath(path)
	}
	relative, ok := detectHomeRelativePath(path)
	if !ok {
		return filepath.Clean(path)
	}
	return joinHomePlaceholder(relative)
}

func detectHomeRelativePath(path string) (string, bool) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	if relative, ok := relativeToBase(filepath.Clean(homeDir), path); ok {
		return relative, true
	}
	homeParent := filepath.Dir(homeDir)
	if homeParent == homeDir || homeParent == filepath.Dir(homeParent) {
		return "", false
	}
	relative, ok := relativeToBase(homeParent, path)
	if !ok {
		return "", false
	}
	parts := strings.Split(relative, "/")
	if len(parts) < 2 {
		return "", false
	}
	return strings.Join(parts[1:], "/"), true
}

func relativeToBase(base string, target string) (string, bool) {
	relative, err := filepath.Rel(base, filepath.Clean(target))
	if err != nil || relative == "." || relative == "" || filepath.IsAbs(relative) {
		return "", false
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(relative), true
}

func joinHomePlaceholder(relative string) string {
	trimmed := strings.TrimPrefix(filepath.ToSlash(relative), "/")
	if trimmed == "" || trimmed == "." {
		return homePlaceholder
	}
	return homePlaceholder + "/" + trimmed
}

func normalizePlaceholderPath(path string) string {
	normalized := filepath.ToSlash(path)
	normalized = strings.Replace(normalized, homePlaceholder+"//", homePlaceholder+"/", 1)
	return strings.TrimSuffix(normalized, "/.")
}
