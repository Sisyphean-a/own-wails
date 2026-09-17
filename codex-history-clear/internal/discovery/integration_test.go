package discovery_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"codex-history-manager/internal/discovery"
	"codex-history-manager/internal/planning"
)

func TestRunReadOnlyScanFeedsDeletePlanWithEnrichedManifest(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOME", homeDir)

	root := filepath.Join(homeDir, ".codex")
	buildIntegrationFixtureRoot(t, root)

	scanResult, err := discovery.NewService().RunReadOnlyScan()
	if err != nil {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}

	planResult, err := planning.NewService().BuildDeletePlan(scanResult.ManifestPath)
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}

	if planResult.Summary != (planning.Summary{
		GroupCount:     1,
		CandidateCount: 2,
		ReviewCount:    0,
		PlannedCount:   2,
	}) {
		t.Fatalf("Summary = %#v", planResult.Summary)
	}

	group := planResult.Groups[0]
	livePath := filepath.Join(root, "sessions", "live", "rollout-1.jsonl")
	archivedPath := filepath.Join(root, "sessions", "archived", "rollout-2.jsonl")
	if group.PreferredPath != livePath {
		t.Fatalf("PreferredPath = %q, want %q", group.PreferredPath, livePath)
	}

	preferred := findCandidateByPath(t, group, livePath)
	if !preferred.Preferred || preferred.Action != "keep" || preferred.ReasonCode != "cli-visible-preferred" {
		t.Fatalf("preferred candidate = %#v", preferred)
	}

	archived := findCandidateByPath(t, group, archivedPath)
	if archived.Relation != "physical-copy" || archived.Action != "quarantine" || archived.QuarantinePath == nil {
		t.Fatalf("archived candidate = %#v", archived)
	}
}

func buildIntegrationFixtureRoot(t *testing.T, root string) {
	t.Helper()
	liveRolloutPath := filepath.Join(root, "sessions", "live", "rollout-1.jsonl")
	writeIntegrationFixtureFile(t, root, "config.toml", "theme = \"light\"\n")
	writeIntegrationFixtureFile(t, root, "auth.json", "{}\n")
	writeIntegrationFixtureFile(t, root, "credentials.json", "{}\n")
	writeIntegrationFixtureFile(t, root, "history.jsonl", integrationJSONLine(map[string]any{
		"session_id": "session-keep",
		"ts":         1770901613,
		"text":       "hello",
	}))
	writeIntegrationFixtureFile(t, root, "session_index.jsonl", integrationJSONLine(map[string]any{
		"id":             "session-keep",
		"path":           liveRolloutPath,
		"model_provider": "fox",
		"status":         "active",
	}))
	writeIntegrationFixtureFile(t, root, filepath.Join("sqlite", "state_main.sqlite"), "sqlite")
	writeIntegrationFixtureFile(t, root, filepath.Join("sqlite", "logs_main.sqlite"), "sqlite")
	writeIntegrationFixtureFile(t, root, filepath.Join("sessions", "live", "rollout-1.jsonl"),
		buildIntegrationRolloutFixture("session-keep", "/mnt/c/Work/Repo", "2026-06-30T07:08:09Z"),
	)
	writeIntegrationFixtureFile(t, root, filepath.Join("sessions", "archived", "rollout-2.jsonl"),
		buildIntegrationRolloutFixture("session-keep", "/mnt/c/Work/Repo", "2026-06-29T07:08:09Z"),
	)
}

func writeIntegrationFixtureFile(t *testing.T, root string, relativePath string, content string) {
	t.Helper()
	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func buildIntegrationRolloutFixture(sessionUID string, cwd string, timestamp string) string {
	return integrationJSONLine(map[string]any{
		"timestamp": timestamp,
		"type":      "session_meta",
		"payload": map[string]any{
			"id":         sessionUID,
			"cwd":        cwd,
			"originator": "codex_cli_rs",
			"source":     "cli",
		},
	}) + integrationJSONLine(map[string]any{
		"type": "message",
		"payload": map[string]any{
			"text": "hello",
		},
	})
}

func integrationJSONLine(payload any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(data) + "\n"
}

func findCandidateByPath(t *testing.T, group planning.DuplicateGroup, sourcePath string) planning.GroupCandidate {
	t.Helper()
	for _, candidate := range group.Candidates {
		if candidate.SourcePath == sourcePath {
			return candidate
		}
	}
	t.Fatalf("candidate not found: %s", sourcePath)
	return planning.GroupCandidate{}
}
