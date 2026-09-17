package syncflow

import "testing"

func TestComputeLineDiff_InsertionKeepsAlignment(t *testing.T) {
	local := "a\nb\nc"
	remote := "a\nx\nb\nc"
	lines := computeLineDiff(local, remote)

	// Expect: context a, add x, context b, context c — NOT a cascade of mismatches.
	want := []diffLine{
		{Kind: diffKindContext, Text: "a"},
		{Kind: diffKindAdd, Text: "x"},
		{Kind: diffKindContext, Text: "b"},
		{Kind: diffKindContext, Text: "c"},
	}
	if len(lines) != len(want) {
		t.Fatalf("line count mismatch: got %d want %d (%#v)", len(lines), len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d mismatch: got %+v want %+v", i, lines[i], want[i])
		}
	}
}

func TestComputeLineDiff_Stats(t *testing.T) {
	local := "keep\nold1\nold2"
	remote := "keep\nnew1"
	lines := computeLineDiff(local, remote)
	added, removed := diffCounts(lines)
	if added != 1 {
		t.Fatalf("expected 1 added line, got %d (%#v)", added, lines)
	}
	if removed != 2 {
		t.Fatalf("expected 2 removed lines, got %d (%#v)", removed, lines)
	}
}

func TestComputeLineDiff_Identical(t *testing.T) {
	lines := computeLineDiff("same\ntext", "same\ntext")
	added, removed := diffCounts(lines)
	if added != 0 || removed != 0 {
		t.Fatalf("expected no changes, got +%d -%d", added, removed)
	}
}

func TestBuildUnifiedDiff_FormatCompatibility(t *testing.T) {
	diff := buildUnifiedDiff("line-1\nlocal-line\n", "line-1\nremote-line\n")
	for _, part := range []string{"--- local", "+++ remote", "-local-line", "+remote-line"} {
		if !contains(diff, part) {
			t.Fatalf("diff missing %q:\n%s", part, diff)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
