package history

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type snapshotIndex map[string][]string

func indexShellSnapshots(dir string) snapshotIndex {
	if !fileExists(dir) {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	snapshots := make(snapshotIndex)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sh") {
			continue
		}
		threadID, _, found := strings.Cut(name, ".")
		if !found || threadID == "" {
			continue
		}
		snapshots[threadID] = append(snapshots[threadID], filepath.Join(dir, name))
	}
	for _, paths := range snapshots {
		sort.Strings(paths)
	}
	return snapshots
}
