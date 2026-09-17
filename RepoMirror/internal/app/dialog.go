package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type DirectorySelector interface {
	Open(ctx context.Context, defaultDirectory string, title string) (string, error)
}

type WailsDirectorySelector struct{}

func NewWailsDirectorySelector() *WailsDirectorySelector {
	return &WailsDirectorySelector{}
}

func (selector *WailsDirectorySelector) Open(
	ctx context.Context,
	defaultDirectory string,
	title string,
) (string, error) {
	options := runtime.OpenDialogOptions{
		Title:                title,
		DefaultDirectory:     existingDirectory(defaultDirectory),
		CanCreateDirectories: true,
	}
	return runtime.OpenDirectoryDialog(ctx, options)
}

func existingDirectory(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	for current := path; current != ""; {
		info, err := os.Stat(current)
		if err == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}
