package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"codex-history-manager/internal/discovery"
	"codex-history-manager/internal/history"
	"codex-history-manager/internal/planning"
	_ "modernc.org/sqlite"
)

const (
	bindingThreadID = "019f3000-1111-7222-8333-abcdefabcdef"
	bindingOtherID  = "019f3000-9999-7222-8333-fedcbafedcba"
	bindingRollout  = "sessions\\2026\\07\\02\\rollout-2026-07-02T12-00-00-019f3000-1111-7222-8333-abcdefabcdef.jsonl"
)

func TestExportHistoryEvidencePackFillsReadOnlyArtifactsAndReports(t *testing.T) {
	buildBindingCodex(t)
	app := &App{
		discovery: discovery.NewService(),
		history:   history.NewService(),
		planning:  planning.NewService(),
	}

	plan, err := app.BuildHistoryDeletePlan(history.BuildPlanRequest{ThreadIDs: []string{bindingThreadID}})
	if err != nil {
		t.Fatalf("BuildHistoryDeletePlan() error = %v", err)
	}
	approved, err := app.ApproveHistoryDeletePlan(history.ApproveRequest{PlanPath: plan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveHistoryDeletePlan() error = %v", err)
	}
	execResult, err := app.ExecuteHistoryDeletePlan(history.ExecuteRequest{
		PlanPath:   approved.ApprovedPlanPath,
		Confirmed:  true,
		BackupOnly: true,
	})
	if err != nil {
		t.Fatalf("ExecuteHistoryDeletePlan() error = %v", err)
	}
	pack, err := app.ExportHistoryEvidencePack(history.EvidencePackRequest{
		RunID:               execResult.RunID,
		DeletePlanPath:      plan.PlanPath,
		ApprovedPlanPath:    execResult.ApprovedPlanPath,
		RollbackJournalPath: execResult.RollbackJournalPath,
		ExecResultPath:      execResult.ExecResultPath,
	})
	if err != nil {
		t.Fatalf("ExportHistoryEvidencePack() error = %v", err)
	}
	assertArtifactLabels(t, pack, []string{
		"discovery",
		"manifest_before",
		"duplicate_groups",
		"delete_plan",
		"approved_plan",
		"rollback_journal",
		"exec_result",
		"architecture",
		"context",
		"history",
	})
}

func buildBindingCodex(t *testing.T) {
	t.Helper()
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOME", homeDir)
	writeBindingFiles(t, root)
	writeBindingDatabases(t, root)
}

func writeBindingFiles(t *testing.T, root string) {
	t.Helper()
	liveRolloutPath := filepath.Join(root, "sessions", "live", "rollout-1.jsonl")
	writeBindingFile(t, filepath.Join(root, "config.toml"), "theme = \"light\"\n")
	writeBindingFile(t, filepath.Join(root, "auth.json"), "{}\n")
	writeBindingFile(t, filepath.Join(root, "credentials.json"), "{}\n")
	writeBindingFile(t, filepath.Join(root, "session_index.jsonl"), joinJSONLines(
		bindingJSONLine(map[string]any{"id": "session-keep", "path": liveRolloutPath, "model_provider": "fox", "status": "active"}),
		bindingJSONLine(map[string]any{"id": bindingThreadID, "thread_name": "Primary Thread", "updated_at": "2026-07-02T12:00:00Z"}),
		bindingJSONLine(map[string]any{"id": bindingOtherID, "thread_name": "Secondary Thread", "updated_at": "2026-07-02T11:00:00Z"}),
	))
	writeBindingFile(t, filepath.Join(root, "history.jsonl"), joinJSONLines(
		bindingJSONLine(map[string]any{"session_id": "session-keep", "ts": 1770901613, "text": "hello"}),
		bindingJSONLine(map[string]any{"session_id": bindingThreadID, "ts": 1782996210, "text": "hello"}),
		bindingJSONLine(map[string]any{"session_id": bindingOtherID, "ts": 1782996211, "text": "keep"}),
	))
	globalState := map[string]any{
		"projectless-thread-ids": []string{bindingThreadID, bindingOtherID},
		"thread-workspace-root-hints": map[string]any{
			"local:" + bindingThreadID: "E:\\repo",
			"local:" + bindingOtherID:  "E:\\keep",
		},
	}
	writeBindingFile(t, filepath.Join(root, ".codex-global-state.json"), prettyJSON(globalState))
	writeBindingFile(t, filepath.Join(root, ".codex-global-state.json.bak"), prettyJSON(globalState))
	writeBindingFile(t, filepath.Join(root, bindingRollout), joinJSONLines(
		bindingJSONLine(map[string]any{"timestamp": "2026-07-02T12:00:00Z", "type": "session_meta", "payload": map[string]any{"session_id": bindingThreadID, "id": bindingThreadID, "cwd": "E:\\repo", "source": "cli"}}),
		bindingJSONLine(map[string]any{"timestamp": "2026-07-02T12:00:01Z", "type": "message", "payload": map[string]any{"text": "hello"}}),
	))
	writeBindingFile(t, filepath.Join(root, "sessions", "2026", "07", "02", "rollout-keep-"+bindingOtherID+".jsonl"), bindingJSONLine(map[string]any{
		"timestamp": "2026-07-02T11:00:00Z",
		"type":      "session_meta",
		"payload":   map[string]any{"session_id": bindingOtherID, "id": bindingOtherID, "cwd": "E:\\keep", "source": "cli"},
	})+"\n")
	writeBindingFile(t, filepath.Join(root, "sessions", "live", "rollout-1.jsonl"), bindingJSONLine(map[string]any{
		"timestamp": "2026-07-02T12:00:00Z",
		"type":      "session_meta",
		"payload":   map[string]any{"session_id": "session-keep", "id": "session-keep", "cwd": "/mnt/c/Work/Repo", "originator": "codex_cli_rs", "source": "cli"},
	})+"\n")
	writeBindingFile(t, filepath.Join(root, "sessions", "archived", "rollout-2.jsonl"), bindingJSONLine(map[string]any{
		"timestamp": "2026-07-01T11:00:00Z",
		"type":      "session_meta",
		"payload":   map[string]any{"session_id": "session-keep", "id": "session-keep", "cwd": "/mnt/c/Work/Repo", "originator": "codex_cli_rs", "source": "cli"},
	})+"\n")
}

func writeBindingDatabases(t *testing.T, root string) {
	t.Helper()
	writeBindingStateDB(t, filepath.Join(root, "state_5.sqlite"), root)
	writeBindingSimpleDB(t, filepath.Join(root, "logs_2.sqlite"), `create table logs (thread_id text, ts integer);`, `insert into logs(thread_id, ts) values (?, ?)`, bindingThreadID, 1782996210)
	writeBindingSimpleDB(t, filepath.Join(root, "goals_1.sqlite"), `create table thread_goals (thread_id text, goal_id text);`, `insert into thread_goals(thread_id, goal_id) values (?, 'goal-1')`, bindingThreadID)
}

func writeBindingStateDB(t *testing.T, path string, root string) {
	t.Helper()
	db := openBindingDB(t, path)
	defer db.Close()
	for _, statement := range []string{
		`create table threads (id text primary key, rollout_path text, created_at integer, updated_at integer, title text, source text, model_provider text, thread_source text, cwd text, archived integer, first_user_message text, preview text, created_at_ms integer, updated_at_ms integer);`,
		`create table thread_dynamic_tools (thread_id text, position integer, name text);`,
		`create table thread_spawn_edges (parent_thread_id text, child_thread_id text, status text);`,
		`create table agent_job_items (assigned_thread_id text);`,
	} {
		mustBindingExec(t, db, statement)
	}
	insertBindingThread(t, db, bindingThreadID, filepath.Join(root, bindingRollout), "source-primary", "E:\\repo", 1782996210)
	insertBindingThread(t, db, bindingOtherID, filepath.Join(root, "sessions", "2026", "07", "02", "rollout-keep-"+bindingOtherID+".jsonl"), "source-secondary", "E:\\keep", 1782996200)
	mustBindingExec(t, db, `insert into thread_dynamic_tools(thread_id, position, name) values (?, 0, 'tool')`, bindingThreadID)
	mustBindingExec(t, db, `insert into thread_spawn_edges(parent_thread_id, child_thread_id, status) values (?, ?, 'done')`, bindingThreadID, bindingOtherID)
	mustBindingExec(t, db, `insert into agent_job_items(assigned_thread_id) values (?)`, bindingThreadID)
}

func writeBindingSimpleDB(t *testing.T, path string, createSQL string, insertSQL string, args ...any) {
	t.Helper()
	db := openBindingDB(t, path)
	defer db.Close()
	mustBindingExec(t, db, createSQL)
	mustBindingExec(t, db, insertSQL, args...)
}

func assertArtifactLabels(t *testing.T, pack history.EvidencePackResult, labels []string) {
	t.Helper()
	got := map[string]bool{}
	for _, artifact := range pack.Artifacts {
		got[artifact.Label] = true
	}
	for _, label := range labels {
		if !got[label] {
			t.Fatalf("missing artifact label: %s", label)
		}
	}
}

func openBindingDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	return db
}

func mustBindingExec(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err != nil {
		t.Fatalf("Exec(%q) error = %v", statement, err)
	}
}

func insertBindingThread(t *testing.T, db *sql.DB, id string, rolloutPath string, title string, cwd string, updatedAt int64) {
	t.Helper()
	source := "vscode"
	modelProvider := "hi_code"
	threadSource := "user"
	if id == bindingOtherID {
		source = "cli"
		modelProvider = "openai"
	}
	mustBindingExec(t, db, `insert into threads(id, rollout_path, created_at, updated_at, title, source, model_provider, thread_source, cwd, archived, first_user_message, preview, created_at_ms, updated_at_ms) values (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)`,
		id, rolloutPath, updatedAt, updatedAt, title, source, modelProvider, threadSource, cwd, title, title, updatedAt*1000, updatedAt*1000,
	)
}

func writeBindingFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func bindingJSONLine(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func joinJSONLines(lines ...string) string {
	return stringsJoin(lines, "\n") + "\n"
}

func stringsJoin(lines []string, separator string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for _, line := range lines[1:] {
		result += separator + line
	}
	return result
}

func prettyJSON(value any) string {
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data) + "\n"
}
