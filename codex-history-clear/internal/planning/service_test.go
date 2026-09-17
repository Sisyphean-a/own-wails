package planning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"codex-history-manager/internal/discovery"
)

func TestBuildDeletePlanReturnsEmptyPlaceholder(t *testing.T) {
	manifestPath := writeManifest(t, []discovery.ManifestRecord{})
	result, err := NewService().BuildDeletePlan(manifestPath)
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	if result.Summary != (Summary{}) {
		t.Fatalf("Summary = %#v", result.Summary)
	}
	assertPlanDocument(t, result.DeletePlanPath, 0, false)
	assertGroupsArtifact(t, result.DuplicateGroupsPath, 0)
}

func TestBuildDeletePlanGroupsAliasesAndCopies(t *testing.T) {
	sessionUID := "session-1"
	records := []discovery.ManifestRecord{
		manifestRecord(sessionUID, `C:\Users\A\.codex\sessions\live\rollout-keep.jsonl`, `C:\Users\A\.codex\sessions\live\rollout-keep.jsonl`, `2026-06-30T10:00:00Z`, []string{"cli-visible", "rollout-file"}),
		manifestRecord(sessionUID, `D:\mirror\rollout-keep.jsonl`, `C:\Users\A\.codex\sessions\live\rollout-keep.jsonl`, `2026-06-30T09:59:00Z`, []string{"rollout-file"}),
		manifestRecord(sessionUID, `E:\backup\rollout-keep.jsonl`, `E:\backup\rollout-keep.jsonl`, `2026-06-29T09:59:00Z`, []string{"rollout-file"}),
	}
	result, err := NewService().BuildDeletePlan(writeManifest(t, records))
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	if result.Summary.GroupCount != 1 || result.Summary.ReviewCount != 0 {
		t.Fatalf("Summary = %#v", result.Summary)
	}
	group := result.Groups[0]
	if group.PreferredPath != records[0].SourcePath {
		t.Fatalf("PreferredPath = %q", group.PreferredPath)
	}
	assertCandidateAction(t, group.Candidates[0], "keep", "preferred")
	assertCandidateAction(t, group.Candidates[1], "repair_index", "path-alias")
	assertCandidateAction(t, group.Candidates[2], "quarantine", "physical-copy")
}

func TestBuildDeletePlanMarksAmbiguousGroupReviewNeeded(t *testing.T) {
	sessionUID := "session-2"
	records := []discovery.ManifestRecord{
		manifestRecord(sessionUID, `C:\Users\A\.codex\sessions\live\rollout-a.jsonl`, `C:\Users\A\.codex\sessions\live\rollout-a.jsonl`, `2026-06-30T10:00:00Z`, []string{"rollout-file"}),
		manifestRecord(sessionUID, `E:\backup\rollout-b.jsonl`, `E:\backup\rollout-b.jsonl`, `2026-06-30T10:00:00Z`, []string{"rollout-file"}),
	}
	result, err := NewService().BuildDeletePlan(writeManifest(t, records))
	if err != nil {
		t.Fatalf("BuildDeletePlan() error = %v", err)
	}
	if result.Summary.ReviewCount != 1 {
		t.Fatalf("Summary = %#v", result.Summary)
	}
	group := result.Groups[0]
	if !group.ReviewNeeded || group.Warning == "" {
		t.Fatalf("Group = %#v", group)
	}
	for _, candidate := range group.Candidates {
		if candidate.Action != "keep" || candidate.ReasonCode != "review-needed" {
			t.Fatalf("candidate = %#v", candidate)
		}
	}
}

func manifestRecord(sessionUID string, sourcePath string, realPath string, updatedAt string, evidence []string) discovery.ManifestRecord {
	return discovery.ManifestRecord{
		SessionUID:     &sessionUID,
		StorageKind:    "codex_rollout_jsonl",
		SourcePath:     sourcePath,
		CanonicalPath:  sourcePath,
		RealPath:       realPath,
		ReparseKind:    "none",
		CwdNorm:        `c:\repo`,
		UpdatedAt:      updatedAt,
		ContentHash:    "sha256:same",
		Preferred:      false,
		Evidence:       evidence,
		DuplicateGroup: nil,
	}
}

func writeManifest(t *testing.T, records []discovery.ManifestRecord) string {
	t.Helper()
	outputDir := filepath.Join(t.TempDir(), "20260630-100000")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", outputDir, err)
	}
	path := filepath.Join(outputDir, "manifest-before.json")
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}

func assertPlanDocument(t *testing.T, path string, wantItems int, wantApproved bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var document DeletePlanDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}
	if len(document.Items) != wantItems || document.Approved != wantApproved {
		t.Fatalf("document = %#v", document)
	}
}

func assertGroupsArtifact(t *testing.T, path string, wantGroups int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var groups []DuplicateGroup
	if err := json.Unmarshal(data, &groups); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}
	if len(groups) != wantGroups {
		t.Fatalf("groups = %#v", groups)
	}
}

func assertCandidateAction(t *testing.T, candidate GroupCandidate, wantAction string, wantRelation string) {
	t.Helper()
	if candidate.Action != wantAction || candidate.Relation != wantRelation {
		t.Fatalf("candidate = %#v", candidate)
	}
}
