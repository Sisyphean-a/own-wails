package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertThreadDeleted(t *testing.T, service *Service, threadID string) {
	t.Helper()
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	assertCount(t, paths.stateDB, `select count(*) from threads where id = ?`, threadID, 0)
	assertCount(t, paths.logsDB, `select count(*) from logs where thread_id = ?`, threadID, 0)
	assertCount(t, paths.goalsDB, `select count(*) from thread_goals where thread_id = ?`, threadID, 0)
	assertFileMissing(t, filepath.Join(paths.codexHome, testRolloutRel))
	assertTextContainsState(t, paths.sessionIndex, threadID, false)
	assertTextContainsState(t, paths.history, threadID, false)
}

func assertThreadRestored(t *testing.T, service *Service, threadID string) {
	t.Helper()
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	assertCount(t, paths.stateDB, `select count(*) from threads where id = ?`, threadID, 1)
	assertCount(t, paths.logsDB, `select count(*) from logs where thread_id = ?`, threadID, 1)
	assertCount(t, paths.goalsDB, `select count(*) from thread_goals where thread_id = ?`, threadID, 1)
	if !fileExists(filepath.Join(paths.codexHome, testRolloutRel)) {
		t.Fatalf("rollout file was not restored")
	}
	assertTextContainsState(t, paths.sessionIndex, threadID, true)
	assertTextContainsState(t, paths.history, threadID, true)
}

func assertCount(t *testing.T, path string, query string, arg string, want int) {
	t.Helper()
	db, err := openReadonlyDatabase(path)
	if err != nil {
		t.Fatalf("openReadonlyDatabase() error = %v", err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(query, arg).Scan(&count); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != want {
		t.Fatalf("count = %d", count)
	}
}

func assertTextContainsState(t *testing.T, path string, needle string, want bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	hasNeedle := strings.Contains(string(data), needle)
	if hasNeedle != want {
		t.Fatalf("contains(%s) = %v", needle, hasNeedle)
	}
}

func assertFileMissing(t *testing.T, path string) {
	t.Helper()
	if fileExists(path) {
		t.Fatalf("file still exists: %s", path)
	}
}
