package syncer

import (
	"fmt"
	"os"
	"strings"
)

// Rule：只回收本应用保留命名空间内的普通临时文件，不进入 Git 元数据或符号链接。
func cleanupTransfers(directory string) error {
	root, err := openRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	var walk func(string) error
	walk = func(dir string) error {
		if dir != "." {
			if err := checkNoLinks(root, dir); err != nil {
				return err
			}
		}
		f, err := root.Open(dir)
		if err != nil {
			return err
		}
		entries, err := f.ReadDir(-1)
		f.Close()
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 || strings.EqualFold(entry.Name(), ".git") {
				continue
			}
			p := entry.Name()
			if dir != "." {
				p = dir + "/" + p
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".lansync-") {
					continue
				}
				if err = walk(p); err != nil {
					return err
				}
				continue
			}
			if entry.Type().IsRegular() && strings.HasPrefix(entry.Name(), ".lansync-") && validID(strings.TrimPrefix(entry.Name(), ".lansync-")) {
				if err = root.Remove(p); err != nil {
					return fmt.Errorf("清理中断传输失败：%w", err)
				}
			}
		}
		return nil
	}
	return walk(".")
}
