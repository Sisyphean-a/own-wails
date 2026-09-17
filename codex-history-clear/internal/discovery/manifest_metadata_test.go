package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadRolloutSessionMetaIgnoresStructuredSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	content := `{"timestamp":"2026-02-14T14:14:40.296Z","type":"session_meta","payload":{"id":"session-subagent","cwd":"E:\\repo","originator":"codex_cli_rs","source":{"subagent":{"thread_spawn":{"parent_thread_id":"parent","depth":1}}}}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	metadata, err := readRolloutSessionMeta(path)
	if err != nil {
		t.Fatalf("readRolloutSessionMeta() error = %v", err)
	}
	if metadata.SessionUID != "session-subagent" {
		t.Fatalf("session_uid = %q", metadata.SessionUID)
	}
	if metadata.Cwd != `E:\repo` {
		t.Fatalf("cwd = %q", metadata.Cwd)
	}
}
