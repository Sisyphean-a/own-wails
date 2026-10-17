package syncer

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMultipleDirectoriesAndPausedRemoval(t *testing.T) {
	a, b := paired(t)
	ra, rb, rc, rd := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	write(t, ra, "a.md", "目录甲")
	write(t, rd, "b.txt", "目录乙")
	first := taskPair(t, a, b, ra, rb)
	second := taskPair(t, a, b, rc, rd)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, id := range []string{first, second} {
		wg.Add(1)
		go func(id string) { defer wg.Done(); errs <- a.Sync(id) }(id)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		must(t, err)
	}
	if read(t, rb, "a.md") != "目录甲" || read(t, rc, "b.txt") != "目录乙" {
		t.Fatal("多目录并发同步失败")
	}
	must(t, a.Pause(first, true))
	must(t, b.Pause(first, true))
	must(t, a.RemoveTask(first))
	if _, err := a.task(first); err == nil {
		t.Fatal("本机配置未移除")
	}
	if _, err := b.task(first); err == nil {
		t.Fatal("对端配置未移除")
	}
	if read(t, ra, "a.md") != "目录甲" || read(t, rb, "a.md") != "目录甲" {
		t.Fatal("移除配置删除了文件")
	}
	must(t, a.rpc(b.id, "/task/remove", map[string]string{"taskId": first}, nil))
}
func TestStaleMarkdownDecisionDoesNotOverwrite(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "a.md", "甲")
	write(t, rb, "a.md", "乙")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	comparison, err := a.Compare(id, "a.md")
	must(t, err)
	write(t, rb, "a.md", "查看差异期间的新修改")
	if err = a.Resolve(id, "a.md", comparison.LocalHash, comparison.RemoteHash, "过期结果", false); err == nil {
		t.Fatal("允许过期决策覆盖新版本")
	}
	if read(t, rb, "a.md") != "查看差异期间的新修改" {
		t.Fatal("过期决策丢失内容")
	}
}
func TestGitIgnoredParentIsVisibleAndNotCopied(t *testing.T) {
	a, b := paired(t)
	parent := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(parent, ".git"), 0700))
	write(t, parent, ".gitignore", "private/\n")
	ra := filepath.Join(parent, "private")
	must(t, os.MkdirAll(ra, 0700))
	write(t, ra, "secret.md", "不应同步")
	rb := t.TempDir()
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	if _, err := os.Stat(filepath.Join(rb, "secret.md")); !os.IsNotExist(err) {
		t.Fatal("祖先 Git 忽略规则未生效")
	}
	s := a.Snapshot().Tasks[0].Status
	if len(s.Skipped) == 0 {
		t.Fatal("没有提供跳过记录")
	}
}
