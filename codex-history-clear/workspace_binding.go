package main

import (
	"os"
	"path/filepath"
	"strings"

	"codex-history-manager/internal/codexhome"
)

type CleanupWorkspaceConfig struct {
	CodexHome    string `json:"codexHome"`
	BackupRoot   string `json:"backupRoot"`
	UsingDefault bool   `json:"usingDefault"`
}

func (a *App) GetCleanupWorkspaceConfig() (CleanupWorkspaceConfig, error) {
	override := strings.TrimSpace(a.historyOverride())
	return buildCleanupWorkspaceConfig(override)
}

func (a *App) SetCleanupWorkspaceRoot(root string) (CleanupWorkspaceConfig, error) {
	override := strings.TrimSpace(root)
	if override == "" {
		a.history.SetCodexHomeOverride("")
		a.discovery.SetCodexHomeOverride("")
		return buildCleanupWorkspaceConfig("")
	}
	resolved, err := codexhome.Resolve(override, os.UserHomeDir, "扫描目录")
	if err != nil {
		return CleanupWorkspaceConfig{}, err
	}
	a.history.SetCodexHomeOverride(resolved)
	a.discovery.SetCodexHomeOverride(resolved)
	return buildCleanupWorkspaceConfig(resolved)
}

func buildCleanupWorkspaceConfig(override string) (CleanupWorkspaceConfig, error) {
	codexHome, err := codexhome.Resolve(override, os.UserHomeDir, "扫描目录")
	if err != nil {
		return CleanupWorkspaceConfig{}, err
	}
	backupRoot := filepath.Join(os.TempDir(), "codex-history-manager", "history-runs")
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		return CleanupWorkspaceConfig{}, err
	}
	return CleanupWorkspaceConfig{
		CodexHome:    codexHome,
		BackupRoot:   backupRoot,
		UsingDefault: strings.TrimSpace(override) == "",
	}, nil
}

func (a *App) historyOverride() string {
	return a.history.CodexHomeOverride()
}
