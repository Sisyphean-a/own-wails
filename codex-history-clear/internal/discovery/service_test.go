package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestPreviewScanEnumeratesKnownItems(t *testing.T) {
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".codex")
	buildFixtureRootAt(t, root)
	service := newTestService()
	service.now = func() time.Time { return time.Date(2026, 6, 30, 7, 8, 9, 1, time.UTC) }
	service.userHomeDir = func() (string, error) { return homeDir, nil }
	cleanupOutputDir(t, service)

	result, err := service.RunReadOnlyScan()
	if err != nil {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}

	if result.Summary != (ScanSummary{RootCount: 1, ItemCount: 9, UnknownCount: 0}) {
		t.Fatalf("Summary = %#v", result.Summary)
	}
	assertKinds(t, result.Items, []string{
		"auth_json", "config_toml", "credentials_json", "history_jsonl",
		"logs_sqlite", "rollout_jsonl", "session_index_jsonl", "state_sqlite",
		"archived_rollout_jsonl",
	})

	discoveryDoc := readJSON[discoveryDocument](t, result.DiscoveryPath)
	if discoveryDoc.RunID != result.RunID {
		t.Fatalf("discovery run_id = %q, want %q", discoveryDoc.RunID, result.RunID)
	}

	manifest := readJSON[[]ManifestRecord](t, result.ManifestPath)
	if len(manifest) != 5 {
		t.Fatalf("manifest records = %d, want 5", len(manifest))
	}
	assertStorageKinds(t, manifest, []string{
		"codex_history_jsonl",
		"codex_rollout_jsonl",
		"codex_rollout_jsonl",
		"codex_sqlite",
		"codex_sqlite",
	})

	unknownItems := readJSON[[]UnknownItem](t, result.UnknownItemsPath)
	if len(unknownItems) != 0 {
		t.Fatalf("unknown items = %d, want 0", len(unknownItems))
	}
}

func TestPreviewScanEnrichesRolloutMetadata(t *testing.T) {
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".codex")
	buildFixtureRootAt(t, root)
	service := newTestService()
	service.now = func() time.Time { return time.Date(2026, 6, 30, 7, 8, 9, 4, time.UTC) }
	service.userHomeDir = func() (string, error) { return homeDir, nil }
	cleanupOutputDir(t, service)

	result, err := service.RunReadOnlyScan()
	if err != nil {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}

	manifest := readJSON[[]ManifestRecord](t, result.ManifestPath)
	live := findManifestRecordBySuffix(t, manifest, filepath.Join("sessions", "live", "rollout-1.jsonl"))
	if live.SessionUID == nil || *live.SessionUID != "session-keep" {
		t.Fatalf("live session_uid = %#v", live.SessionUID)
	}
	if live.CwdRaw == nil || *live.CwdRaw != "/mnt/c/Work/Repo" {
		t.Fatalf("live cwd_raw = %#v", live.CwdRaw)
	}
	if live.CwdNorm != `c:\work\repo` {
		t.Fatalf("live cwd_norm = %q", live.CwdNorm)
	}
	if live.CanonicalPath != live.SourcePath {
		t.Fatalf("live canonical_path = %q, source_path = %q", live.CanonicalPath, live.SourcePath)
	}
	if !containsEvidence(live.Evidence, "cli-visible") {
		t.Fatalf("live evidence = %v", live.Evidence)
	}

	archived := findManifestRecordBySuffix(t, manifest, filepath.Join("sessions", "archived", "rollout-2.jsonl"))
	if archived.SessionUID == nil || *archived.SessionUID != "session-keep" {
		t.Fatalf("archived session_uid = %#v", archived.SessionUID)
	}
	if archived.CanonicalPath != live.SourcePath {
		t.Fatalf("archived canonical_path = %q, want %q", archived.CanonicalPath, live.SourcePath)
	}
	if containsEvidence(archived.Evidence, "cli-visible") {
		t.Fatalf("archived evidence should not be cli-visible: %v", archived.Evidence)
	}
}

func TestPreviewScanUsesHomeFallback(t *testing.T) {
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".codex")
	writeFixtureFile(t, root, "history.jsonl", "{}\n")

	service := newTestService()
	service.now = func() time.Time { return time.Date(2026, 6, 30, 7, 8, 9, 2, time.UTC) }
	service.userHomeDir = func() (string, error) { return homeDir, nil }
	cleanupOutputDir(t, service)

	result, err := service.RunReadOnlyScan()
	if err != nil {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}

	if result.Summary != (ScanSummary{RootCount: 1, ItemCount: 1, UnknownCount: 0}) {
		t.Fatalf("Summary = %#v", result.Summary)
	}
	if result.DiscoveryPath == "" || result.ManifestPath == "" || result.UnknownItemsPath == "" {
		t.Fatalf("artifact paths are empty: %#v", result)
	}
	unknownItems := readJSON[[]UnknownItem](t, result.UnknownItemsPath)
	if len(unknownItems) != 0 {
		t.Fatalf("unknown items = %d, want 0", len(unknownItems))
	}
}

func TestPreviewScanTracksUnknownCandidateFiles(t *testing.T) {
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".codex")
	buildFixtureRootAt(t, root)
	writeFixtureFile(t, root, filepath.Join("sessions", "live", "mystery-session.jsonl"), "{}\n")
	service := newTestService()
	service.now = func() time.Time { return time.Date(2026, 6, 30, 7, 8, 9, 3, time.UTC) }
	service.userHomeDir = func() (string, error) { return homeDir, nil }
	cleanupOutputDir(t, service)

	result, err := service.RunReadOnlyScan()
	if err != nil {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}

	unknownItems := readJSON[[]UnknownItem](t, result.UnknownItemsPath)
	if len(unknownItems) != 1 {
		t.Fatalf("unknown items = %d, want 1", len(unknownItems))
	}
	if !containsUnknownPath(unknownItems, "mystery-session.jsonl") {
		t.Fatalf("unknown items missing mystery-session.jsonl: %#v", unknownItems)
	}
}

func TestPreviewScanSkipsPluginCacheNoise(t *testing.T) {
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".codex")
	buildFixtureRootAt(t, root)
	writeFixtureFile(t, root, filepath.Join(".tmp", "plugins", "noise.json"), "{}\n")
	service := newTestService()
	service.now = func() time.Time { return time.Date(2026, 6, 30, 7, 8, 9, 7, time.UTC) }
	service.userHomeDir = func() (string, error) { return homeDir, nil }
	cleanupOutputDir(t, service)

	result, err := service.RunReadOnlyScan()
	if err != nil {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}

	unknownItems := readJSON[[]UnknownItem](t, result.UnknownItemsPath)
	if len(unknownItems) != 0 {
		t.Fatalf("unknown items = %#v", unknownItems)
	}
}

func TestPreviewScanKeepsBrokenRolloutMetadataVisible(t *testing.T) {
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, ".codex")
	buildFixtureRootAt(t, root)
	writeFixtureFile(t, root, filepath.Join("sessions", "live", "rollout-1.jsonl"), "")
	service := newTestService()
	service.now = func() time.Time { return time.Date(2026, 6, 30, 7, 8, 9, 6, time.UTC) }
	service.userHomeDir = func() (string, error) { return homeDir, nil }
	cleanupOutputDir(t, service)

	result, err := service.RunReadOnlyScan()
	if err != nil {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}

	manifest := readJSON[[]ManifestRecord](t, result.ManifestPath)
	live := findManifestRecordBySuffix(t, manifest, filepath.Join("sessions", "live", "rollout-1.jsonl"))
	if !containsEvidence(live.Evidence, "rollout-metadata-missing") {
		t.Fatalf("live evidence = %v", live.Evidence)
	}
	if live.CwdNorm != "" {
		t.Fatalf("live cwd_norm = %q", live.CwdNorm)
	}
}

func TestClassifyPathMarksArchivedSessionsAsArchivedRollout(t *testing.T) {
	path := filepath.Join(`C:\Users\A\.codex`, "archived_sessions", "rollout-sample.jsonl")
	kind, ok := classifyPath(path)
	if !ok || kind != "archived_rollout_jsonl" {
		t.Fatalf("classifyPath(%q) = (%q, %t)", path, kind, ok)
	}
}

func TestPreviewScanRejectsMissingRoot(t *testing.T) {
	service := newTestService()
	service.userHomeDir = func() (string, error) { return filepath.Join(t.TempDir(), "missing"), nil }
	_, err := service.RunReadOnlyScan()
	if err == nil || !strings.Contains(err.Error(), "扫描目录不存在") {
		t.Fatalf("RunReadOnlyScan() error = %v", err)
	}
}

func TestValidateRootPreservesUnexpectedStatErrors(t *testing.T) {
	err := validateRoot(string([]byte{0}))
	if err == nil || !strings.Contains(err.Error(), "扫描目录不可用") {
		t.Fatalf("validateRoot() error = %v", err)
	}
}

func TestBuildRunIDIncludesNanoseconds(t *testing.T) {
	first := buildRunID(time.Date(2026, 6, 30, 7, 8, 9, 123456789, time.UTC))
	second := buildRunID(time.Date(2026, 6, 30, 7, 8, 9, 987654321, time.UTC))

	if first == second {
		t.Fatalf("buildRunID() should distinguish scans in the same second: %q", first)
	}
	if !strings.Contains(first, "-123456789") {
		t.Fatalf("buildRunID() = %q, want nanosecond suffix", first)
	}
}

func TestContentHashForLargeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large-history.jsonl")
	payload := strings.Repeat("{\"event\":\"scan\"}\n", 32*1024)
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	got, err := contentHashFor(path)
	if err != nil {
		t.Fatalf("contentHashFor() error = %v", err)
	}

	sum := sha256.Sum256([]byte(payload))
	want := "sha256:" + hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("contentHashFor() = %q, want %q", got, want)
	}
}

func newTestService() *Service {
	service := NewService()
	service.now = func() time.Time { return time.Date(2026, 6, 30, 7, 8, 9, 0, time.UTC) }
	service.userHomeDir = func() (string, error) { return os.TempDir(), nil }
	return service
}

func cleanupOutputDir(t *testing.T, service *Service) {
	t.Helper()
	outputDir := filepath.Join(os.TempDir(), "codex-history-manager", "runs", buildRunID(service.now().UTC()))
	if err := os.RemoveAll(outputDir); err != nil {
		t.Fatalf("RemoveAll(%q) error = %v", outputDir, err)
	}
}

func buildFixtureRootAt(t *testing.T, root string) {
	t.Helper()
	liveRolloutPath := filepath.Join(root, "sessions", "live", "rollout-1.jsonl")
	writeFixtureFile(t, root, "config.toml", "theme = \"light\"\n")
	writeFixtureFile(t, root, "auth.json", "{}\n")
	writeFixtureFile(t, root, "credentials.json", "{}\n")
	writeFixtureFile(t, root, "history.jsonl", jsonLine(map[string]any{
		"session_id": "session-keep",
		"ts":         1770901613,
		"text":       "hello",
	}))
	writeFixtureFile(t, root, "session_index.jsonl",
		jsonLine(map[string]any{
			"id":             "session-keep",
			"path":           liveRolloutPath,
			"model_provider": "fox",
			"status":         "active",
		})+
			jsonLine(map[string]any{
				"id":          "session-title-only",
				"thread_name": "ignored",
				"updated_at":  "2026-06-30T07:08:00Z",
			}),
	)
	writeFixtureFile(t, root, filepath.Join("sqlite", "state_main.sqlite"), "sqlite")
	writeFixtureFile(t, root, filepath.Join("sqlite", "logs_main.sqlite"), "sqlite")
	writeFixtureFile(t, root, filepath.Join("sessions", "live", "rollout-1.jsonl"),
		buildRolloutFixture("session-keep", "/mnt/c/Work/Repo", "2026-06-30T07:08:09Z"),
	)
	writeFixtureFile(t, root, filepath.Join("sessions", "archived", "rollout-2.jsonl"),
		buildRolloutFixture("session-keep", "/mnt/c/Work/Repo", "2026-06-29T07:08:09Z"),
	)
}

func writeFixtureFile(t *testing.T, root string, relativePath string, content string) {
	t.Helper()
	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertKinds(t *testing.T, items []DiscoveryItem, expected []string) {
	t.Helper()
	kinds := make([]string, 0, len(items))
	for _, item := range items {
		if !filepath.IsAbs(item.Path) {
			t.Fatalf("item path is not absolute: %q", item.Path)
		}
		kinds = append(kinds, item.Kind)
	}
	slices.Sort(kinds)
	slices.Sort(expected)
	if !slices.Equal(kinds, expected) {
		t.Fatalf("Kinds = %v, want %v", kinds, expected)
	}
}

func assertStorageKinds(t *testing.T, records []ManifestRecord, expected []string) {
	t.Helper()
	kinds := make([]string, 0, len(records))
	for _, record := range records {
		if !strings.HasPrefix(record.ContentHash, "sha256:") {
			t.Fatalf("content hash = %q", record.ContentHash)
		}
		kinds = append(kinds, record.StorageKind)
	}
	slices.Sort(kinds)
	slices.Sort(expected)
	if !slices.Equal(kinds, expected) {
		t.Fatalf("StorageKinds = %v, want %v", kinds, expected)
	}
}

func readJSON[T any](t *testing.T, path string) T {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var payload T
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}
	return payload
}

func containsUnknownPath(items []UnknownItem, suffix string) bool {
	for _, item := range items {
		if strings.HasSuffix(item.Path, suffix) {
			return true
		}
	}
	return false
}

func findManifestRecordBySuffix(t *testing.T, records []ManifestRecord, suffix string) ManifestRecord {
	t.Helper()
	for _, record := range records {
		if strings.HasSuffix(record.SourcePath, suffix) {
			return record
		}
	}
	t.Fatalf("manifest record not found: %s", suffix)
	return ManifestRecord{}
}

func containsEvidence(evidence []string, token string) bool {
	for _, item := range evidence {
		if item == token {
			return true
		}
	}
	return false
}

func buildRolloutFixture(sessionUID string, cwd string, timestamp string) string {
	return jsonLine(map[string]any{
		"timestamp": timestamp,
		"type":      "session_meta",
		"payload": map[string]any{
			"id":         sessionUID,
			"cwd":        cwd,
			"originator": "codex_cli_rs",
			"source":     "cli",
		},
	}) + jsonLine(map[string]any{
		"type": "message",
		"payload": map[string]any{
			"text": "hello",
		},
	})
}

func jsonLine(payload any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(data) + "\n"
}
