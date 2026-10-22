package gitops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"RepoMirror/internal/model"
	"RepoMirror/internal/testutil"
)

func TestRevisionBrowsingAndPagination(t *testing.T) {
	root := t.TempDir()
	testutil.InitRepo(t, root)
	testutil.WriteFile(t, root, "file.txt", "content\n")
	testutil.CommitAll(t, root, "first")
	for index := 0; index < commitPageSize; index++ {
		testutil.RunGitOutput(t, root, "commit", "--allow-empty", "-m", fmt.Sprintf("提交 %d", index))
	}
	testutil.RunGitOutput(t, root, "branch", "feature")
	service := NewService(NewExecRunner())
	branches, err := service.ListBranches(root)
	if err != nil || len(branches) != 2 {
		t.Fatalf("branches: %+v %v", branches, err)
	}
	page, err := service.ListCommits(root, "refs/heads/feature", 0)
	if err != nil || len(page.Commits) != commitPageSize || !page.HasMore {
		t.Fatalf("page: %+v %v", page, err)
	}
	if page.Commits[0].Subject != fmt.Sprintf("提交 %d", commitPageSize-1) || page.Commits[0].Author == "" || page.Commits[0].Date == "" {
		t.Fatalf("metadata: %+v", page.Commits[0])
	}
	last, err := service.ListCommits(root, page.Commits[0].Hash, commitPageSize)
	if err != nil || len(last.Commits) != 1 || last.HasMore || last.Commits[0].Subject != "first" {
		t.Fatalf("last: %+v %v", last, err)
	}
	if _, err := service.ListCommits(root, "HEAD", -1); err == nil {
		t.Fatal("negative offset accepted")
	}
	if _, err := service.ResolveRevision(root, "--help"); err == nil {
		t.Fatal("option-like revision accepted")
	}
}

func TestSnapshotStreamsExactBlobsAndCleansUp(t *testing.T) {
	root := t.TempDir()
	testutil.InitRepo(t, root)
	content := strings.Repeat("binary\x00\r\n", 100000)
	testutil.WriteFile(t, root, "data.bin", content)
	testutil.WriteFile(t, root, ".gitattributes", "data.bin export-ignore\n")
	testutil.CommitAll(t, root, "snapshot")
	service := NewService(NewExecRunner())
	hash, err := service.ResolveRevision(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.ExportRevision(root, model.SourceSelection{Mode: "branch", Commit: hash})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { snapshot.Close() })
	got, err := os.ReadFile(filepath.Join(snapshot.Root, "data.bin"))
	if err != nil || string(got) != content {
		t.Fatalf("blob bytes changed: %v", err)
	}
	if err := snapshot.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(snapshot.Root); !os.IsNotExist(err) {
		t.Fatal("snapshot not released")
	}
}

func TestSnapshotRejectsLinksAndUnsafePaths(t *testing.T) {
	root := t.TempDir()
	testutil.InitRepo(t, root)
	service := NewService(NewExecRunner())
	blob := testutil.RunGitOutput(t, root, "hash-object", "-w", "--stdin")
	testutil.RunGitOutput(t, root, "update-index", "--add", "--cacheinfo", "120000,"+blob+",link")
	testutil.RunGitOutput(t, root, "commit", "-m", "link")
	hash, err := service.ResolveRevision(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExportRevision(root, model.SourceSelection{Mode: "branch", Commit: hash}); err == nil {
		t.Fatal("symbolic link was accepted")
	}
	for _, path := range []string{"../outside", "/absolute", "C:/escape", "a\\escape", "a/../b", "dir./file"} {
		if err := validateSnapshotPath(path); err == nil {
			t.Errorf("unsafe path accepted: %s", path)
		}
	}
}
