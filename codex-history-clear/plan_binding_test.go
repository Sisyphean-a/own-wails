package main

import (
	"encoding/json"
	"testing"

	"codex-history-manager/internal/planning"
)

func TestMapDeletePlanResultSerializesEmptyWarningsAsArray(t *testing.T) {
	result := mapDeletePlanResult(planning.Result{
		RunID:               "run-1",
		ManifestPath:        "manifest-before.json",
		DuplicateGroupsPath: "duplicate-groups.json",
		DeletePlanPath:      "delete-plan.json",
		Summary:             planning.Summary{},
		Groups: []planning.DuplicateGroup{{
			DuplicateGroup: "dup-000001",
			PreferredPath:  "keep.jsonl",
			Candidates: []planning.GroupCandidate{{
				SourcePath: "keep.jsonl",
				Action:     "keep",
			}},
		}},
		Items: []planning.DeletePlanItem{{
			DuplicateGroup: "dup-000001",
			SourcePath:     "keep.jsonl",
			Action:         "keep",
		}},
		Warnings: []string{},
	})

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	text := string(data)
	if !containsJSONEmptyArray(text, "\"warnings\":[]") {
		t.Fatalf("warnings should serialize as []: %s", text)
	}
}

func containsJSONEmptyArray(text string, fragment string) bool {
	return len(text) >= len(fragment) && contains(text, fragment)
}

func contains(text string, fragment string) bool {
	for index := 0; index+len(fragment) <= len(text); index++ {
		if text[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
