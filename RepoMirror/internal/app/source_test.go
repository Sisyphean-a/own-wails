package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"RepoMirror/internal/config"
	"RepoMirror/internal/model"
	"RepoMirror/internal/testutil"
)

func revisionTestService(t *testing.T, source, target string) *Service {
	t.Helper()
	cfg := model.DefaultConfig()
	cfg.ProjectA, cfg.ProjectB = source, target
	return newTestService(t, config.NewStoreWithPath(filepath.Join(t.TempDir(), "config.json")), &selectorStub{}, cfg)
}

func TestCommitSourceCopiesHistoricalFilesAndOnlyExplicitDeletions(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	testutil.InitRepo(t, source)
	testutil.InitRepo(t, target)
	testutil.WriteFile(t, source, "changed.txt", "before\n")
	testutil.WriteFile(t, source, "deleted.txt", "remove\n")
	testutil.WriteFile(t, source, "old-name.txt", "rename\n")
	testutil.WriteFile(t, source, "untouched.txt", "source\n")
	testutil.CommitAll(t, source, "initial")
	testutil.WriteFile(t, source, "changed.txt", "selected version\n")
	testutil.WriteFile(t, source, "中文/新增.txt", "historical\n")
	if err := os.Remove(filepath.Join(source, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitOutput(t, source, "mv", "old-name.txt", "new-name.txt")
	testutil.CommitAll(t, source, "selected")
	hash := testutil.RunGitOutput(t, source, "rev-parse", "HEAD")
	testutil.WriteFile(t, source, "changed.txt", "later committed version\n")
	testutil.CommitAll(t, source, "later")
	testutil.WriteFile(t, source, "changed.txt", "uncommitted version\n")
	testutil.WriteFile(t, target, "changed.txt", "target edits\n")
	testutil.WriteFile(t, target, "deleted.txt", "remove\n")
	testutil.WriteFile(t, target, "old-name.txt", "rename\n")
	testutil.WriteFile(t, target, "untouched.txt", "target stays\n")
	testutil.WriteFile(t, target, "target-only.txt", "target stays\n")
	testutil.CommitAll(t, target, "target")
	before := testutil.RunGitOutput(t, source, "status", "--porcelain=v1")
	branch := testutil.CurrentBranch(t, source)
	service := revisionTestService(t, source, target)
	state, err := service.SelectSource("commit", hash)
	if err != nil {
		t.Fatal(err)
	}
	if state.SourceSelection.Commit != hash || state.Summary.Total != 5 || state.Summary.Deleted != 2 {
		t.Fatalf("unexpected selected scope: %+v", state)
	}
	for _, entry := range state.Differences {
		if entry.Path == "untouched.txt" || entry.Path == "target-only.txt" {
			t.Fatalf("unrelated entry: %+v", entry)
		}
	}
	state, err = service.SyncRepositories()
	if err != nil {
		t.Fatal(err)
	}
	if state.Summary.Total != 0 {
		t.Fatalf("sync should be idempotent: %+v", state.Summary)
	}
	for path, want := range map[string]string{"changed.txt": "selected version\n", "中文/新增.txt": "historical\n", "new-name.txt": "rename\n", "untouched.txt": "target stays\n", "target-only.txt": "target stays\n"} {
		assertRevisionFile(t, target, path, want)
	}
	for _, path := range []string{"deleted.txt", "old-name.txt"} {
		if _, err := os.Stat(filepath.Join(target, path)); !os.IsNotExist(err) {
			t.Fatalf("expected deletion %s: %v", path, err)
		}
	}
	if after := testutil.RunGitOutput(t, source, "status", "--porcelain=v1"); before != after || branch != testutil.CurrentBranch(t, source) {
		t.Fatal("source worktree or index changed")
	}
	state, err = service.SetDirection(string(model.DirectionBToA))
	if err != nil {
		t.Fatal(err)
	}
	if state.SourceSelection.Mode != "working" {
		t.Fatal("direction change must clear selected revision")
	}
}

func TestBranchSourceIsPinnedAndPreservesProtectionRules(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	testutil.InitRepo(t, source)
	testutil.InitRepo(t, target)
	testutil.WriteFile(t, source, "file.txt", "pinned\n")
	testutil.WriteFile(t, source, "protected.txt", "source\n")
	testutil.WriteFile(t, source, ".gitignore", "source-ignore\n")
	testutil.WriteFile(t, source, ".gitattributes", "file.txt export-ignore\n")
	testutil.CommitAll(t, source, "selected branch")
	testutil.RunGitOutput(t, source, "branch", "snapshot")
	testutil.WriteFile(t, target, ".gitignore", "protected.txt\n")
	testutil.WriteFile(t, target, "protected.txt", "protected\n")
	testutil.RunGitOutput(t, target, "add", "-f", "protected.txt")
	testutil.WriteFile(t, target, "target-only.txt", "delete by snapshot\n")
	testutil.CommitAll(t, target, "target")
	service := revisionTestService(t, source, target)
	state, err := service.SelectSource("branch", "refs/heads/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	selected := state.SourceSelection.Commit
	testutil.WriteFile(t, source, "file.txt", "moved branch\n")
	testutil.CommitAll(t, source, "later")
	testutil.RunGitOutput(t, source, "branch", "-f", "snapshot", "HEAD")
	state, err = service.Refresh()
	if err != nil {
		t.Fatal(err)
	}
	if state.SourceSelection.Commit != selected {
		t.Fatal("refresh silently moved the selected version")
	}
	if _, err := service.SelectSource("commit", "missing-version"); err == nil {
		t.Fatal("invalid revision should fail")
	}
	if service.currentSourceSelection().Commit != selected {
		t.Fatal("failed selection replaced valid source")
	}
	if _, err := service.SyncRepositories(); err != nil {
		t.Fatal(err)
	}
	assertRevisionFile(t, target, "file.txt", "pinned\n")
	assertRevisionFile(t, target, "protected.txt", "protected\n")
	assertRevisionFile(t, target, ".gitignore", "protected.txt\n")
	if _, err := os.Stat(filepath.Join(target, "target-only.txt")); !os.IsNotExist(err) {
		t.Fatal("branch snapshot should delete target-only files")
	}
}

func TestRootEmptyAndMergeCommitScopes(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	testutil.InitRepo(t, source)
	testutil.InitRepo(t, target)
	testutil.WriteFile(t, source, "base.txt", "base\n")
	testutil.CommitAll(t, source, "root")
	service := revisionTestService(t, source, target)
	state, err := service.SelectSource("commit", "HEAD")
	if err != nil || state.Summary.Added != 1 {
		t.Fatalf("root: %+v %v", state.Summary, err)
	}
	testutil.RunGitOutput(t, source, "commit", "--allow-empty", "-m", "empty")
	state, err = service.SelectSource("commit", "HEAD")
	if err != nil || state.Summary.Total != 0 {
		t.Fatalf("empty: %+v %v", state.Summary, err)
	}
	main := testutil.CurrentBranch(t, source)
	testutil.RunGitOutput(t, source, "checkout", "-b", "feature")
	testutil.WriteFile(t, source, "feature.txt", "feature\n")
	testutil.CommitAll(t, source, "feature")
	testutil.RunGitOutput(t, source, "checkout", main)
	testutil.RunGitOutput(t, source, "merge", "--no-ff", "feature", "-m", "merge")
	state, err = service.SelectSource("commit", "HEAD")
	if err != nil || len(state.Differences) != 1 || state.Differences[0].Path != "feature.txt" {
		t.Fatalf("merge: %+v %v", state.Differences, err)
	}
}

func TestRevisionRejectsTargetSymlinkBeforeAnyWrites(t *testing.T) {
	source, target, outside := t.TempDir(), t.TempDir(), t.TempDir()
	testutil.InitRepo(t, source)
	testutil.InitRepo(t, target)
	testutil.WriteFile(t, source, "a.txt", "new\n")
	testutil.WriteFile(t, source, "linked/file.txt", "must not escape\n")
	testutil.CommitAll(t, source, "files")
	testutil.WriteFile(t, target, "a.txt", "original\n")
	testutil.WriteFile(t, outside, "file.txt", "outside\n")
	if err := os.Symlink(outside, filepath.Join(target, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	service := revisionTestService(t, source, target)
	if _, err := service.SelectSource("commit", "HEAD"); err == nil || !strings.Contains(err.Error(), "安全覆盖") {
		t.Fatalf("expected explicit safety failure, got %v", err)
	}
	assertRevisionFile(t, target, "a.txt", "original\n")
	assertRevisionFile(t, outside, "file.txt", "outside\n")
}

func assertRevisionFile(t *testing.T, root, path, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil || string(got) != want {
		t.Fatalf("%s: got %q, want %q: %v", path, got, want, err)
	}
}
