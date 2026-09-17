//go:build windows

package history

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

func resolvePhysicalPath(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	buffer := make([]uint16, 32768)
	length, err := windows.GetFinalPathNameByHandle(
		windows.Handle(file.Fd()), &buffer[0], uint32(len(buffer)), 0,
	)
	if err != nil {
		return "", err
	}
	if length == 0 || length >= uint32(len(buffer)) {
		return "", fmt.Errorf("解析后的路径长度无效: %s", path)
	}
	return normalizeWindowsFinalPath(windows.UTF16ToString(buffer[:length])), nil
}

func normalizeWindowsFinalPath(path string) string {
	if strings.HasPrefix(path, `\\?\UNC\`) {
		return `\\` + strings.TrimPrefix(path, `\\?\UNC\`)
	}
	return strings.TrimPrefix(path, `\\?\`)
}
