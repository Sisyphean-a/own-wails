package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoDocPathFindsMonorepoDocumentFromNestedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "codex-history-clear", "build", "bin")
	document := filepath.Join(root, ".codestable", "architecture", "INDEX.md")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll(nested) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(document), 0o755); err != nil {
		t.Fatalf("MkdirAll(document) error = %v", err)
	}
	if err := os.WriteFile(document, []byte("index\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(document) error = %v", err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(original) }()
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir(nested) error = %v", err)
	}

	if got := repoDocPath(".codestable", "architecture", "INDEX.md"); got != document {
		t.Fatalf("repoDocPath() = %q, want %q", got, document)
	}
}
