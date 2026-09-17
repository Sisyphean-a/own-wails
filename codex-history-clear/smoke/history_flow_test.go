package smoke

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-history-manager/internal/discovery"
	"codex-history-manager/internal/history"
	"codex-history-manager/internal/planning"
	_ "modernc.org/sqlite"
)

const (
	smokeThreadID = "019f3000-1111-7222-8333-abcdefabcdef"
	smokeOtherID  = "019f3000-9999-7222-8333-fedcbafedcba"
	smokeRollout  = "sessions\\2026\\07\\02\\rollout-2026-07-02T12-00-00-019f3000-1111-7222-8333-abcdefabcdef.jsonl"
)

func TestHistoryFlowSmoke(t *testing.T) {
	root := buildSmokeCodex(t)

	scanResult, planResult := runSmokeReadOnlyFlow(t)
	if !fileExists(scanResult.ManifestPath) {
		t.Fatalf("manifest missing: %s", scanResult.ManifestPath)
	}
	if planResult.Summary.GroupCount == 0 {
		t.Fatalf("GroupCount = %d", planResult.Summary.GroupCount)
	}
	runSmokeHistoryFlow(t, root)
}

func runSmokeReadOnlyFlow(t *testing.T) (discovery.ScanResult, planning.Result) {
	t.Helper()
	scanResult, err := discovery.NewService().RunReadOnlyScan()
	if err != nil {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}
	planResult, err := planning.NewService().BuildDeletePlan(scanResult.ManifestPath)
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	return scanResult, planResult
}

func runSmokeHistoryFlow(t *testing.T, root string) {
	t.Helper()
	service := history.NewService()
	listResult, err := service.ListThreads(history.ListRequest{Limit: 10})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if len(listResult.Items) != 2 {
		t.Fatalf("ListThreads() count = %d", len(listResult.Items))
	}
	deletePlan, err := service.BuildDeletePlan(history.BuildPlanRequest{ThreadIDs: []string{smokeThreadID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	approved, err := service.ApproveDeletePlan(history.ApproveRequest{PlanPath: deletePlan.PlanPath})
	if err != nil {
		t.Fatalf("ApproveDeletePlan() error = %v", err)
	}
	execResult, err := service.ExecuteDeletePlan(history.ExecuteRequest{
		PlanPath:   approved.ApprovedPlanPath,
		Confirmed:  true,
		BackupOnly: true,
	})
	if err != nil {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
	if execResult.Mode != "backup-only" {
		t.Fatalf("Mode = %q", execResult.Mode)
	}
	pack, err := service.ExportEvidencePack(history.EvidencePackRequest{
		RunID:               execResult.RunID,
		DeletePlanPath:      deletePlan.PlanPath,
		ApprovedPlanPath:    execResult.ApprovedPlanPath,
		RollbackJournalPath: execResult.RollbackJournalPath,
		ExecResultPath:      execResult.ExecResultPath,
	})
	if err != nil {
		t.Fatalf("ExportEvidencePack() error = %v", err)
	}
	if !fileExists(pack.EvidencePackPath) {
		t.Fatalf("evidence pack missing: %s", pack.EvidencePackPath)
	}
	if !fileExists(filepath.Join(root, smokeRollout)) {
		t.Fatalf("backup-only should keep rollout file")
	}
}

func TestHistoryFlowRejectsUnapprovedPlan(t *testing.T) {
	buildSmokeCodex(t)

	service := history.NewService()
	plan, err := service.BuildDeletePlan(history.BuildPlanRequest{ThreadIDs: []string{smokeThreadID}})
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	_, err = service.ExecuteDeletePlan(history.ExecuteRequest{PlanPath: plan.PlanPath, Confirmed: true})
	if err == nil || !strings.Contains(err.Error(), "尚未批准") {
		t.Fatalf("ExecuteDeletePlan() error = %v", err)
	}
}

func buildSmokeCodex(t *testing.T) string {
	t.Helper()
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOME", homeDir)
	writeSmokeFiles(t, root)
	writeSmokeDatabases(t, root)
	return root
}

func writeSmokeFiles(t *testing.T, root string) {
	t.Helper()
	rolloutPath := filepath.Join(root, smokeRollout)
	liveRolloutPath := filepath.Join(root, "sessions", "live", "rollout-1.jsonl")
	writeSmokeFile(t, filepath.Join(root, "config.toml"), "theme = \"light\"\n")
	writeSmokeFile(t, filepath.Join(root, "auth.json"), "{}\n")
	writeSmokeFile(t, filepath.Join(root, "credentials.json"), "{}\n")
	writeSmokeSessionIndex(t, root, liveRolloutPath)
	writeSmokeHistory(t, root)
	globalState := map[string]any{
		"projectless-thread-ids": []string{smokeThreadID, smokeOtherID},
		"thread-workspace-root-hints": map[string]any{
			"local:" + smokeThreadID: "E:\\repo",
			"local:" + smokeOtherID:  "E:\\keep",
		},
	}
	writeSmokeFile(t, filepath.Join(root, ".codex-global-state.json"), smokeJSONPretty(globalState))
	writeSmokeFile(t, filepath.Join(root, ".codex-global-state.json.bak"), smokeJSONPretty(globalState))
	writeSmokeFile(t, rolloutPath, strings.Join([]string{
		smokeJSONLine(map[string]any{
			"timestamp": "2026-07-02T12:00:00Z",
			"type":      "session_meta",
			"payload":   map[string]any{"session_id": smokeThreadID, "id": smokeThreadID, "cwd": "E:\\repo", "source": "cli"},
		}),
		smokeJSONLine(map[string]any{
			"timestamp": "2026-07-02T12:00:01Z",
			"type":      "message",
			"payload":   map[string]any{"text": "hello"},
		}),
	}, "\n")+"\n")
	writeSmokeFile(t, filepath.Join(root, "sessions", "2026", "07", "02", "rollout-keep-"+smokeOtherID+".jsonl"), smokeJSONLine(map[string]any{
		"timestamp": "2026-07-02T11:00:00Z",
		"type":      "session_meta",
		"payload":   map[string]any{"session_id": smokeOtherID, "id": smokeOtherID, "cwd": "E:\\keep", "source": "cli"},
	})+"\n")
	writeSmokeDiscoveryRollouts(t, root)
}

func writeSmokeSessionIndex(t *testing.T, root string, liveRolloutPath string) {
	t.Helper()
	writeSmokeFile(t, filepath.Join(root, "session_index.jsonl"), strings.Join([]string{
		smokeJSONLine(map[string]any{"id": "session-keep", "path": liveRolloutPath, "model_provider": "fox", "status": "active"}),
		smokeJSONLine(map[string]any{"id": smokeThreadID, "thread_name": "展示标题", "updated_at": "2026-07-02T12:00:00Z"}),
		smokeJSONLine(map[string]any{"id": smokeOtherID, "thread_name": "另一个标题", "updated_at": "2026-07-02T11:00:00Z"}),
	}, "\n")+"\n")
}

func writeSmokeHistory(t *testing.T, root string) {
	t.Helper()
	writeSmokeFile(t, filepath.Join(root, "history.jsonl"), strings.Join([]string{
		smokeJSONLine(map[string]any{"session_id": "session-keep", "ts": 1770901613, "text": "hello"}),
		smokeJSONLine(map[string]any{"session_id": smokeThreadID, "ts": 1782996210, "text": "hello"}),
		smokeJSONLine(map[string]any{"session_id": smokeOtherID, "ts": 1782996211, "text": "keep"}),
	}, "\n")+"\n")
}

func writeSmokeDiscoveryRollouts(t *testing.T, root string) {
	t.Helper()
	writeSmokeFile(t, filepath.Join(root, "sessions", "live", "rollout-1.jsonl"), smokeJSONLine(map[string]any{
		"timestamp": "2026-07-02T12:00:00Z",
		"type":      "session_meta",
		"payload":   map[string]any{"session_id": "session-keep", "id": "session-keep", "cwd": "/mnt/c/Work/Repo", "originator": "codex_cli_rs", "source": "cli"},
	})+"\n")
	writeSmokeFile(t, filepath.Join(root, "sessions", "archived", "rollout-2.jsonl"), smokeJSONLine(map[string]any{
		"timestamp": "2026-07-01T11:00:00Z",
		"type":      "session_meta",
		"payload":   map[string]any{"session_id": "session-keep", "id": "session-keep", "cwd": "/mnt/c/Work/Repo", "originator": "codex_cli_rs", "source": "cli"},
	})+"\n")
}

func writeSmokeDatabases(t *testing.T, root string) {
	t.Helper()
	writeStateDB(t, filepath.Join(root, "state_5.sqlite"), root)
	writeLogsDB(t, filepath.Join(root, "logs_2.sqlite"))
	writeGoalsDB(t, filepath.Join(root, "goals_1.sqlite"))
}

func writeStateDB(t *testing.T, path string, root string) {
	t.Helper()
	db := openSmokeDB(t, path)
	defer db.Close()
	execSmokeSQL(t, db, []string{
		`create table threads (id text primary key, rollout_path text, created_at integer, updated_at integer, title text, source text, model_provider text, thread_source text, cwd text, archived integer, first_user_message text, preview text, created_at_ms integer, updated_at_ms integer);`,
		`create table thread_dynamic_tools (thread_id text, position integer, name text);`,
		`create table thread_spawn_edges (parent_thread_id text, child_thread_id text, status text);`,
		`create table agent_job_items (assigned_thread_id text);`,
	})
	insertSmokeThread(t, db, smokeThreadID, filepath.Join(root, smokeRollout), "源标题", "E:\\repo", 1782996210)
	insertSmokeThread(t, db, smokeOtherID, filepath.Join(root, "sessions", "2026", "07", "02", "rollout-keep-"+smokeOtherID+".jsonl"), "保留标题", "E:\\keep", 1782996200)
	execSmoke(t, db, `insert into thread_dynamic_tools(thread_id, position, name) values (?, 0, 'tool')`, smokeThreadID)
	execSmoke(t, db, `insert into thread_spawn_edges(parent_thread_id, child_thread_id, status) values (?, ?, 'done')`, smokeThreadID, smokeOtherID)
	execSmoke(t, db, `insert into agent_job_items(assigned_thread_id) values (?)`, smokeThreadID)
}

func writeLogsDB(t *testing.T, path string) {
	t.Helper()
	db := openSmokeDB(t, path)
	defer db.Close()
	execSmokeSQL(t, db, []string{`create table logs (thread_id text, ts integer);`})
	execSmoke(t, db, `insert into logs(thread_id, ts) values (?, ?)`, smokeThreadID, 1782996210)
}

func writeGoalsDB(t *testing.T, path string) {
	t.Helper()
	db := openSmokeDB(t, path)
	defer db.Close()
	execSmokeSQL(t, db, []string{`create table thread_goals (thread_id text, goal_id text);`})
	execSmoke(t, db, `insert into thread_goals(thread_id, goal_id) values (?, 'goal-1')`, smokeThreadID)
}

func openSmokeDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	return db
}

func execSmokeSQL(t *testing.T, db *sql.DB, statements []string) {
	t.Helper()
	for _, statement := range statements {
		execSmoke(t, db, statement)
	}
}

func execSmoke(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err != nil {
		t.Fatalf("Exec(%q) error = %v", statement, err)
	}
}

func insertSmokeThread(t *testing.T, db *sql.DB, id string, rolloutPath string, title string, cwd string, updatedAt int64) {
	t.Helper()
	source := "vscode"
	modelProvider := "hi_code"
	threadSource := "user"
	if id == smokeOtherID {
		source = "cli"
		modelProvider = "openai"
	}
	execSmoke(t, db, `insert into threads(id, rollout_path, created_at, updated_at, title, source, model_provider, thread_source, cwd, archived, first_user_message, preview, created_at_ms, updated_at_ms) values (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)`,
		id, rolloutPath, updatedAt, updatedAt, title, source, modelProvider, threadSource, cwd, title, title, updatedAt*1000, updatedAt*1000,
	)
}

func writeSmokeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func smokeJSONLine(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func smokeJSONPretty(value any) string {
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data) + "\n"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
