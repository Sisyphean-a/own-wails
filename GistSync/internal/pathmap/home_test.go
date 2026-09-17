package pathmap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHomePath_ReplacesPlaceholder(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	got, err := ExpandHomePath("{{HOME}}/.ssh/id_rsa")
	if err != nil {
		t.Fatalf("ExpandHomePath returned error: %v", err)
	}

	want := filepath.Join(homeDir, ".ssh", "id_rsa")
	if got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
}

func TestExpandHomePath_AbsolutePathUnchanged(t *testing.T) {
	input := filepath.Join("C:\\", "Temp", "file.txt")
	got, err := ExpandHomePath(input)
	if err != nil {
		t.Fatalf("ExpandHomePath returned error: %v", err)
	}
	if got != input {
		t.Fatalf("absolute path should stay unchanged: got %q want %q", got, input)
	}
}

func TestExpandHomePath_EmptyInput(t *testing.T) {
	_, err := ExpandHomePath("")
	if err == nil {
		t.Fatalf("expected error for empty input")
	}
}

func TestCompactHomePath_ReplacesSiblingUserHomeWithPlaceholder(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	if filepath.Dir(homeDir) == homeDir {
		t.Skip("home dir does not have a parent user directory")
	}

	input := filepath.Join(filepath.Dir(homeDir), "Fusi", ".ssh", "config")
	got := CompactHomePath(input)
	want := "{{HOME}}/.ssh/config"
	if got != want {
		t.Fatalf("compact path mismatch: got %q want %q", got, want)
	}
}
