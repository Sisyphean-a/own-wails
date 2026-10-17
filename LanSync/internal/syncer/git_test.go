package syncer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	gitindex "github.com/go-git/go-git/v5/plumbing/format/index"
)

func TestNestedRepositoryTrackedFiles(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	repo := filepath.Join(ra, "project")
	must(t, os.MkdirAll(filepath.Join(repo, ".git"), 0700))
	f, err := os.Create(filepath.Join(repo, ".git", "index"))
	must(t, err)
	must(t, gitindex.NewEncoder(f).Encode(&gitindex.Index{Version: 2, Entries: []*gitindex.Entry{{Name: "tracked.txt", Mode: filemode.Regular}}}))
	must(t, f.Close())
	write(t, repo, ".gitignore", "*.txt\n")
	write(t, repo, "tracked.txt", "跟踪")
	write(t, repo, "ignored.txt", "忽略")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	if read(t, rb, "project/tracked.txt") != "跟踪" {
		t.Fatal("嵌套仓库已跟踪文件被跳过")
	}
	if _, err = os.Stat(filepath.Join(rb, "project", "ignored.txt")); !os.IsNotExist(err) {
		t.Fatal("嵌套仓库忽略规则失效")
	}
}
func TestWorktreeCommonExclude(t *testing.T) {
	root, meta := t.TempDir(), t.TempDir()
	worktree := filepath.Join(meta, "worktrees", "laptop")
	must(t, os.MkdirAll(worktree, 0700))
	write(t, worktree, "commondir", "../..\n")
	write(t, root, ".git", "gitdir: "+filepath.ToSlash(worktree)+"\n")
	write(t, meta, "info/exclude", "secret.md\n")
	write(t, root, "secret.md", "机密")
	write(t, root, "public.md", "公开")
	m, err := scan(root)
	must(t, err)
	if m.Files["secret.md"].Hash != "" || m.Files["public.md"].Hash == "" {
		t.Fatal("Git 工作树公共忽略规则失效")
	}
}
