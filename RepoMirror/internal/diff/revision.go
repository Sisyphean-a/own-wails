package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateRevisionTargets(root string, paths []string, ignored map[string]struct{}) error {
	checked := make(map[string]struct{})
	for _, path := range paths {
		if _, protected := ignored[path]; protected || isProtected(path) {
			continue
		}
		parts := strings.Split(path, "/")
		for index := range parts {
			relative := strings.Join(parts[:index+1], "/")
			if _, exists := checked[relative]; exists {
				continue
			}
			info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
			if os.IsNotExist(err) {
				break
			}
			if err != nil {
				return err
			}
			// Rule: 版本复制不穿过目标符号链接，也不隐式替换目录或特殊文件。
			if info.Mode()&os.ModeSymlink != 0 || (index < len(parts)-1 && !info.IsDir()) || (index == len(parts)-1 && !info.Mode().IsRegular()) {
				return fmt.Errorf("目标路径不是可安全覆盖的普通文件: %s", relative)
			}
			checked[relative] = struct{}{}
		}
	}
	return nil
}
