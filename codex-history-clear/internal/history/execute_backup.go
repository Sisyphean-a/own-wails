package history

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func createBackups(outputDir string, document planDocument) ([]BackupArtifact, rollbackJournal, error) {
	backupDir := filepath.Join(outputDir, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, rollbackJournal{}, err
	}
	paths := collectBackupPaths(document.Targets)
	backups := make([]BackupArtifact, 0, len(paths))
	entries := make([]RollbackEntry, 0, len(paths))
	for index, originalPath := range paths {
		backupPath := filepath.Join(backupDir, fmt.Sprintf("%03d-%s", index+1, filepath.Base(originalPath)))
		if err := copyFile(originalPath, backupPath); err != nil {
			return nil, rollbackJournal{}, err
		}
		backups = append(backups, BackupArtifact{OriginalPath: originalPath, BackupPath: backupPath})
		entries = append(entries, RollbackEntry{OriginalPath: originalPath, BackupPath: backupPath})
	}
	return backups, rollbackJournal{
		RunID:    document.RunID,
		PlanPath: filepath.Join(outputDir, "approved-plan.json"),
		Entries:  entries,
	}, nil
}

func collectBackupPaths(targets []PlanTarget) []string {
	seen := map[string]struct{}{}
	paths := []string{}
	for _, target := range targets {
		for _, store := range target.Stores {
			if !store.Exists || store.Path == "" {
				continue
			}
			addBackupPath(store.Path, seen, &paths)
			if strings.HasSuffix(store.Path, ".sqlite") {
				addBackupPath(store.Path+"-wal", seen, &paths)
				addBackupPath(store.Path+"-shm", seen, &paths)
			}
		}
	}
	sort.Strings(paths)
	return paths
}

func addBackupPath(path string, seen map[string]struct{}, paths *[]string) {
	if !fileExists(path) {
		return
	}
	if _, ok := seen[path]; ok {
		return
	}
	seen[path] = struct{}{}
	*paths = append(*paths, path)
}

func copyFile(source string, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	output, err := os.Create(target)
	if err != nil {
		return err
	}
	defer output.Close()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Close()
}
