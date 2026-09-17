package history

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestListThreadsRejectsNonUUIDCloneID(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	writeCloneRollout(t, filepath.Join(paths.sessionsDir, "rollout-short.jsonl"), "a", testThreadID)

	result, err := service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if hasThread(result.Items, "a") {
		t.Fatalf("non-UUID clone should not be listed")
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Code != "invalid-session-id" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestListThreadsClearsTimeWhenTranscriptScanIsIncomplete(t *testing.T) {
	service := newFixtureService(t)
	paths, err := service.resolvePaths()
	if err != nil {
		t.Fatalf("resolvePaths() error = %v", err)
	}
	path := filepath.Join(paths.sessionsDir, "rollout-large-"+testCloneID+".jsonl")
	first := jsonLine(map[string]any{
		"timestamp": "2026-07-16T10:00:00Z", "type": "session_meta",
		"payload": map[string]any{"id": testCloneID, "cloned_from": testThreadID},
	})
	writeFixtureFile(t, path, first+"\n"+strings.Repeat("x", 5*1024*1024)+"\n")

	result, err := service.ListThreads(ListRequest{Limit: -1, All: true})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if findThread(t, result.Items, testCloneID).UpdatedAt != "" {
		t.Fatalf("incomplete transcript must not expose an activity time")
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Code != "read-error" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}
