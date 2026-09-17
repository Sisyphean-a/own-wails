package discovery

import (
	"os"
	"path/filepath"
)

func fileAttributes(info os.FileInfo) []string {
	attributes := []string{}
	if info.Mode().Perm()&0o222 == 0 {
		attributes = append(attributes, "ReadOnly")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		attributes = append(attributes, "ReparsePoint")
	}
	return attributes
}

func linkMetadata(path string, info os.FileInfo) (*string, *string) {
	if info.Mode()&os.ModeSymlink == 0 {
		return nil, nil
	}
	linkType := "symlink"
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return &linkType, nil
	}
	return &linkType, &target
}
