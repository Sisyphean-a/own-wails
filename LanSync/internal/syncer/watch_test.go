package syncer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func awaitFile(t *testing.T, root, p, want string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err == nil && string(b) == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("后台同步未完成：%s", p)
}
func TestAutomaticSyncAfterDirectoryInvitation(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "a.md", "初始")
	taskPair(t, a, b, ra, rb)
	awaitFile(t, rb, "a.md", "初始")
	write(t, rb, "new/文档.md", "对端新内容")
	awaitFile(t, ra, "new/文档.md", "对端新内容")
	write(t, rb, "new/文档.md", "对端再次修改")
	awaitFile(t, ra, "new/文档.md", "对端再次修改")
}
