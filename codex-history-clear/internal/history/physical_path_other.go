//go:build !windows

package history

import "path/filepath"

func resolvePhysicalPath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
