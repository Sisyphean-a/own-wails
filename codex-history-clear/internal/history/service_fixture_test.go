package history

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codex-history-manager/internal/discovery"
)

const (
	testThreadID   = "019f3000-1111-7222-8333-abcdefabcdef"
	testOtherID    = "019f3000-9999-7222-8333-fedcbafedcba"
	testRolloutRel = "sessions\\2026\\07\\02\\rollout-2026-07-02T12-00-00-019f3000-1111-7222-8333-abcdefabcdef.jsonl"
)

func newFixtureService(t *testing.T) *Service {
	t.Helper()
	homeDir := t.TempDir()
	codexHome := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	buildFixtureFiles(t, codexHome)
	buildFixtureDatabases(t, codexHome)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOME", homeDir)
	return &Service{
		now: func() time.Time {
			return time.Date(2026, 7, 2, 12, 0, 0, 123456789, time.UTC)
		},
		userHomeDir: func() (string, error) {
			return homeDir, nil
		},
		newDiscovery: discovery.NewService,
	}
}

func buildFixtureFiles(t *testing.T, codexHome string) {
	t.Helper()
	rolloutPath := filepath.Join(codexHome, testRolloutRel)
	writeFixtureFile(t, filepath.Join(codexHome, "config.toml"), "theme = \"light\"\n")
	writeFixtureFile(t, filepath.Join(codexHome, "auth.json"), "{}\n")
	writeFixtureFile(t, filepath.Join(codexHome, "credentials.json"), "{}\n")
	writeFixtureFile(t, filepath.Join(codexHome, "session_index.jsonl"), strings.Join([]string{
		jsonLine(sessionIndexRow{ID: testThreadID, ThreadName: "展示标题", UpdatedAt: "2026-07-02T12:00:00Z"}),
		jsonLine(sessionIndexRow{ID: testOtherID, ThreadName: "另一个标题", UpdatedAt: "2026-07-02T11:00:00Z"}),
	}, "\n")+"\n")
	writeFixtureFile(t, filepath.Join(codexHome, "history.jsonl"), strings.Join([]string{
		jsonLine(map[string]any{"session_id": testThreadID, "ts": 1782996210, "text": "hello"}),
		jsonLine(map[string]any{"session_id": testOtherID, "ts": 1782996211, "text": "keep"}),
	}, "\n")+"\n")
	writeFixtureGlobalState(t, codexHome)
	writeFixtureRollouts(t, codexHome, rolloutPath)
}

func writeFixtureGlobalState(t *testing.T, codexHome string) {
	t.Helper()
	globalState := map[string]any{
		"projectless-thread-ids": []string{testThreadID, testOtherID},
		"thread-workspace-root-hints": map[string]any{
			"local:" + testThreadID: "E:\\repo",
			"local:" + testOtherID:  "E:\\keep",
		},
	}
	writeFixtureFile(t, filepath.Join(codexHome, ".codex-global-state.json"), jsonPretty(globalState))
	writeFixtureFile(t, filepath.Join(codexHome, ".codex-global-state.json.bak"), jsonPretty(globalState))
}

func writeFixtureRollouts(t *testing.T, codexHome string, rolloutPath string) {
	t.Helper()
	writeFixtureFile(t, rolloutPath, strings.Join([]string{
		jsonLine(map[string]any{
			"timestamp": "2026-07-02T12:00:00Z",
			"type":      "session_meta",
			"payload": map[string]any{
				"session_id": testThreadID,
				"id":         testThreadID,
				"cwd":        "E:\\repo",
				"source":     "cli",
			},
		}),
		jsonLine(map[string]any{
			"timestamp": "2026-07-02T12:00:01Z",
			"type":      "message",
			"payload":   map[string]any{"text": "hello"},
		}),
	}, "\n")+"\n")
	writeFixtureFile(t, filepath.Join(codexHome, "sessions", "2026", "07", "02", "rollout-keep-"+testOtherID+".jsonl"), jsonLine(map[string]any{
		"timestamp": "2026-07-02T11:00:00Z",
		"type":      "session_meta",
		"payload":   map[string]any{"session_id": testOtherID, "id": testOtherID, "cwd": "E:\\keep", "source": "cli"},
	})+"\n")
}

func buildFixtureDatabases(t *testing.T, codexHome string) {
	t.Helper()
	buildStateDB(t, filepath.Join(codexHome, "state_5.sqlite"), codexHome)
	buildLogsDB(t, filepath.Join(codexHome, "logs_2.sqlite"))
	buildGoalsDB(t, filepath.Join(codexHome, "goals_1.sqlite"))
}

func buildStateDB(t *testing.T, path string, codexHome string) {
	t.Helper()
	db, err := openDatabase(path)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}
	defer db.Close()
	execStatements(t, db, []string{
		`create table threads (id text primary key, rollout_path text, created_at integer, updated_at integer, title text, source text, model_provider text, thread_source text, cwd text, archived integer, first_user_message text, preview text, created_at_ms integer, updated_at_ms integer);`,
		`create table thread_dynamic_tools (thread_id text, position integer, name text);`,
		`create table thread_spawn_edges (parent_thread_id text, child_thread_id text, status text);`,
		`create table agent_job_items (assigned_thread_id text);`,
	})
	rolloutPath := filepath.Join(codexHome, testRolloutRel)
	insertThread(t, db, testThreadID, rolloutPath, "源标题", "E:\\repo", 1782996210)
	insertThread(t, db, testOtherID, filepath.Join(codexHome, "sessions", "2026", "07", "02", "rollout-keep-"+testOtherID+".jsonl"), "保留标题", "E:\\keep", 1782996200)
	mustExec(t, db, `insert into thread_dynamic_tools(thread_id, position, name) values (?, 0, 'tool')`, testThreadID)
	mustExec(t, db, `insert into thread_spawn_edges(parent_thread_id, child_thread_id, status) values (?, ?, 'done')`, testThreadID, testOtherID)
	mustExec(t, db, `insert into agent_job_items(assigned_thread_id) values (?)`, testThreadID)
}

func buildLogsDB(t *testing.T, path string) {
	t.Helper()
	db, err := openDatabase(path)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}
	defer db.Close()
	execStatements(t, db, []string{`create table logs (thread_id text, ts integer);`})
	mustExec(t, db, `insert into logs(thread_id, ts) values (?, ?)`, testThreadID, 1782996210)
}

func buildGoalsDB(t *testing.T, path string) {
	t.Helper()
	db, err := openDatabase(path)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}
	defer db.Close()
	execStatements(t, db, []string{`create table thread_goals (thread_id text, goal_id text);`})
	mustExec(t, db, `insert into thread_goals(thread_id, goal_id) values (?, 'goal-1')`, testThreadID)
}

func insertThread(t *testing.T, db *sql.DB, id string, rolloutPath string, title string, cwd string, updatedAt int64) {
	t.Helper()
	source := "vscode"
	modelProvider := "hi_code"
	threadSource := "user"
	if id == testOtherID {
		source = "cli"
		modelProvider = "openai"
	}
	mustExec(t, db, `insert into threads(id, rollout_path, created_at, updated_at, title, source, model_provider, thread_source, cwd, archived, first_user_message, preview, created_at_ms, updated_at_ms) values (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)`,
		id, rolloutPath, updatedAt, updatedAt, title, source, modelProvider, threadSource, cwd, title, title, updatedAt*1000, updatedAt*1000,
	)
}

func execStatements(t *testing.T, db *sql.DB, statements []string) {
	t.Helper()
	for _, statement := range statements {
		mustExec(t, db, statement)
	}
}

func mustExec(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err != nil {
		t.Fatalf("Exec(%q) error = %v", statement, err)
	}
}

func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func jsonLine(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func jsonPretty(value any) string {
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data) + "\n"
}
