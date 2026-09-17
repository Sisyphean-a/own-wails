package codexhome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Resolve(explicit string, userHomeDir func() (string, error), label string) (string, error) {
	root := strings.TrimSpace(explicit)
	if root == "" {
		homeDir, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取用户目录失败: %w", err)
		}
		homeDir = strings.TrimSpace(homeDir)
		if homeDir == "" {
			return "", fmt.Errorf("未找到用户目录")
		}
		root = filepath.Join(homeDir, ".codex")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("解析%s失败: %w", label, err)
	}
	cleanRoot := filepath.Clean(absoluteRoot)
	info, err := os.Stat(cleanRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s不存在: %s", label, cleanRoot)
		}
		return "", fmt.Errorf("%s不可用: %s: %w", label, cleanRoot, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s不是文件夹: %s", label, cleanRoot)
	}
	return cleanRoot, nil
}
