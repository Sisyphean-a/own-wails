package profileutil

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var windowsDrivePrefix = regexp.MustCompile(`^[A-Za-z]:/+`)

func NormalizeRelativePath(sourcePath string) string {
	path := strings.ReplaceAll(sourcePath, "\\", "/")
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	path = windowsDrivePrefix.ReplaceAllString(path, "")
	path = strings.TrimPrefix(path, "{{HOME}}/")
	path = strings.TrimPrefix(path, "/")
	path = strings.ReplaceAll(path, ":", "")
	return path
}

func GenerateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func GenerateScopedID(prefix string, scope string) string {
	return fmt.Sprintf("%s-%s-%d", prefix, scope, time.Now().UnixNano())
}
