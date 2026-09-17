package history

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListThreadsScansThreeThousandCloneTranscripts(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	for index := 0; index < 3000; index++ {
		id := fmt.Sprintf("10000000-0000-7000-8000-%012x", index)
		writeCloneRollout(t, filepath.Join(paths.sessionsDir, "scale", "rollout-"+id+".jsonl"), id, testThreadID)
	}
	started := time.Now()
	result, err := service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if len(result.Items) != 3002 {
		t.Fatalf("ListThreads() count = %d", len(result.Items))
	}
	t.Logf("scanned 3000 clone transcripts in %s", time.Since(started))
}

func TestCatalogScansEachTranscriptRootOnce(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	if err := os.MkdirAll(paths.archivedSessionsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	metrics := &historyScanMetrics{}
	service.scanMetrics = metrics
	_, err = service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if metrics.transcriptWalks != 2 || metrics.catalogDatabaseOpen != 1 {
		t.Fatalf("scan metrics = %#v", metrics)
	}
}

func TestListThreadsKeepsCloneWithoutTimestampAndWarns(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	path := filepath.Join(paths.sessionsDir, "rollout-no-time-"+testCloneID+".jsonl")
	writeFixtureFile(t, path, jsonLine(map[string]any{
		"type":    "session_meta",
		"payload": map[string]any{"id": testCloneID, "cloned_from": testThreadID},
	})+"\n")

	result, err := service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if findThread(t, result.Items, testCloneID).UpdatedAt != "" {
		t.Fatalf("clone without timestamp should have empty updatedAt")
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Code != "missing-timestamp" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

const testCloneID = "019f3000-2222-7333-8444-aabbccddeeff"

func TestListThreadsIncludesOnlyUnregisteredMetadataClones(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	writeCloneRollout(t, filepath.Join(paths.sessionsDir, "2026", "07", "01", "rollout-clone-"+testCloneID+".jsonl"), testCloneID, testThreadID)
	writeFixtureFile(t, filepath.Join(paths.sessionsDir, "2026", "07", "01", "rollout-ordinary.jsonl"), jsonLine(map[string]any{
		"timestamp": "2026-07-01T10:00:00Z",
		"type":      "session_meta",
		"payload":   map[string]any{"id": "ordinary-id", "cwd": "E:\\ordinary"},
	})+"\n")

	result, err := service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	clone := findThread(t, result.Items, testCloneID)
	if !clone.IsClone || clone.ClonedFrom != testThreadID || clone.Registered {
		t.Fatalf("clone metadata = %#v", clone)
	}
	if clone.Source != "clone-file" || clone.UpdatedAt != "2026-07-01T10:05:00Z" {
		t.Fatalf("clone source/time = %q/%q", clone.Source, clone.UpdatedAt)
	}
	if hasThread(result.Items, "ordinary-id") {
		t.Fatalf("ordinary unregistered rollout was listed")
	}
}

func TestListThreadsMergesRegisteredCloneAndAllRolloutPaths(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	writeCloneRollout(t, filepath.Join(paths.sessionsDir, "2026", "07", "02", "rollout-live-"+testThreadID+".jsonl"), testThreadID, testOtherID)
	writeCloneRollout(t, filepath.Join(paths.archivedSessionsDir, "rollout-archived-"+testThreadID+".jsonl"), testThreadID, testOtherID)

	result, err := service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	thread := findThread(t, result.Items, testThreadID)
	if !thread.Registered || !thread.IsClone || thread.ClonedFrom != testOtherID {
		t.Fatalf("registered clone metadata = %#v", thread)
	}
	if thread.Title != "展示标题" || len(thread.RolloutPaths) != 3 {
		t.Fatalf("registered clone title/paths = %q/%#v", thread.Title, thread.RolloutPaths)
	}
}

func TestListThreadsAggregatesUnregisteredCloneMetadata(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	livePath := filepath.Join(paths.sessionsDir, "parent-archived_sessions", "rollout-live-"+testCloneID+".jsonl")
	archivedPath := filepath.Join(paths.archivedSessionsDir, "rollout-archived-"+testCloneID+".jsonl")
	writeCloneRolloutAt(t, livePath, testCloneID, testThreadID, "2026-06-01T10:00:00Z")
	writeCloneRolloutAt(t, archivedPath, testCloneID, testThreadID, "2026-07-16T10:00:00.9Z")

	thread := findThread(t, mustListThreads(t, service), testCloneID)
	if thread.Registered || thread.Archived || len(thread.RolloutPaths) != 2 {
		t.Fatalf("clone aggregation = %#v", thread)
	}
	if thread.UpdatedAt != "2026-07-16T10:05:00.9Z" {
		t.Fatalf("UpdatedAt = %q", thread.UpdatedAt)
	}
}

func TestListThreadsPreservesRegisteredSnapshotSize(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	snapshot := filepath.Join(paths.shellSnapshotsDir, testThreadID+".snapshot.sh")
	writeFixtureFile(t, snapshot, "snapshot-data")
	writeCloneRollout(t, filepath.Join(paths.archivedSessionsDir, "rollout-extra-"+testThreadID+".jsonl"), testThreadID, testOtherID)

	thread := findThread(t, mustListThreads(t, service), testThreadID)
	var rolloutBytes int64
	for _, path := range thread.RolloutPaths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Fatalf("Stat(%s) error = %v", path, statErr)
		}
		rolloutBytes += info.Size()
	}
	if thread.SizeBytes != rolloutBytes+int64(len("snapshot-data")) {
		t.Fatalf("SizeBytes = %d, rolloutBytes = %d", thread.SizeBytes, rolloutBytes)
	}
}

func TestListThreadsUsesMaximumTimestampWithinTranscript(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	path := filepath.Join(paths.sessionsDir, "rollout-unordered-"+testCloneID+".jsonl")
	records := []map[string]any{
		{"timestamp": "2026-06-01T10:00:00Z", "type": "session_meta", "payload": map[string]any{"id": testCloneID, "cloned_from": testThreadID}},
		{"timestamp": "2026-07-16T10:00:00Z", "type": "event_msg", "payload": map[string]any{}},
		{"timestamp": "2026-06-02T10:00:00Z", "type": "event_msg", "payload": map[string]any{}},
	}
	content := ""
	for _, record := range records {
		content += jsonLine(record) + "\n"
	}
	writeFixtureFile(t, path, content)

	thread := findThread(t, mustListThreads(t, service), testCloneID)
	if thread.UpdatedAt != "2026-07-16T10:00:00Z" {
		t.Fatalf("UpdatedAt = %q", thread.UpdatedAt)
	}
}

func TestListThreadsReturnsDeduplicatedScanWarnings(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	badPath := filepath.Join(paths.sessionsDir, "rollout-bad.jsonl")
	writeFixtureFile(t, badPath, "not json\nnot json either\n")

	result, err := service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if result.Summary.WarningCount != 1 || len(result.Warnings) != 1 {
		t.Fatalf("warnings = %#v summary = %#v", result.Warnings, result.Summary)
	}
	if result.Warnings[0].Path != badPath || result.Warnings[0].Code != "invalid-json" {
		t.Fatalf("warning = %#v", result.Warnings[0])
	}
}

func writeCloneRollout(t *testing.T, path string, id string, clonedFrom string) {
	writeCloneRolloutAt(t, path, id, clonedFrom, "2026-07-01T10:00:00Z")
}

func writeCloneRolloutAt(t *testing.T, path string, id string, clonedFrom string, timestamp string) {
	t.Helper()
	started, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
	writeFixtureFile(t, path, jsonLine(map[string]any{
		"timestamp": started.Format(time.RFC3339Nano),
		"type":      "session_meta",
		"payload": map[string]any{
			"id":                id,
			"cwd":               "E:\\clone",
			"source":            "cli",
			"model_provider":    "custom",
			"original_provider": "openai",
			"cloned_from":       clonedFrom,
		},
	})+"\n"+jsonLine(map[string]any{
		"timestamp": started.Add(5 * time.Minute).Format(time.RFC3339Nano),
		"type":      "event_msg",
		"payload":   map[string]any{"type": "task_started"},
	})+"\n")
}

func findThread(t *testing.T, threads []ThreadSummary, id string) ThreadSummary {
	t.Helper()
	for _, thread := range threads {
		if thread.ID == id {
			return thread
		}
	}
	t.Fatalf("thread %s not found", id)
	return ThreadSummary{}
}

func hasThread(threads []ThreadSummary, id string) bool {
	for _, thread := range threads {
		if thread.ID == id {
			return true
		}
	}
	return false
}
