package syncer

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func write(t *testing.T, root, p, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(p))
	must(t, os.MkdirAll(filepath.Dir(full), 0755))
	must(t, os.WriteFile(full, []byte(content), 0644))
}
func read(t *testing.T, root, p string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	must(t, err)
	return string(b)
}
func newDevice(t *testing.T) *Engine {
	t.Helper()
	e, err := New(t.TempDir())
	must(t, err)
	must(t, e.Start("127.0.0.1:0", false))
	t.Cleanup(func() { must(t, e.Close()) })
	return e
}
func address(e *Engine) string { return net.JoinHostPort("127.0.0.1", fmtInt(e.port)) }
func fmtInt(v int) string      { return fmt.Sprintf("%d", v) }
func paired(t *testing.T) (*Engine, *Engine) {
	t.Helper()
	a, b := newDevice(t), newDevice(t)
	must(t, a.Connect(address(b)))
	var id string
	for key := range a.pairings {
		id = key
	}
	if len(a.Snapshot().Peers) != 0 {
		t.Fatal("尚未确认就已配对")
	}
	must(t, a.ApprovePair(id))
	if len(a.Snapshot().Peers) != 0 {
		t.Fatal("单端确认就已配对")
	}
	must(t, b.ApprovePair(id))
	if len(a.Snapshot().Peers) != 1 || len(b.Snapshot().Peers) != 1 {
		t.Fatal("双端确认未完成")
	}
	return a, b
}
func taskPair(t *testing.T, a, b *Engine, rootA, rootB string) string {
	t.Helper()
	must(t, a.Invite(b.id, "资料", rootA))
	var id string
	for key := range b.invitations {
		id = key
	}
	must(t, b.AcceptInvite(id, rootB))
	must(t, a.Begin(id))
	return id
}
func TestTwoDevicesSyncBackupAndRestore(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "notes/a.md", "初始\n")
	write(t, rb, "b.txt", "B 的资料")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	if read(t, rb, "notes/a.md") != "初始\n" || read(t, ra, "b.txt") != "B 的资料" {
		t.Fatal("双向新增失败")
	}
	versions, err := b.Versions(id)
	must(t, err)
	if len(versions) != 0 {
		t.Fatal("新增文件不应备份")
	}
	write(t, ra, "notes/a.md", "第二版\n")
	must(t, a.Sync(id))
	versions, err = b.Versions(id)
	must(t, err)
	if len(versions) != 1 {
		t.Fatalf("覆盖前未备份：%+v", versions)
	}
	must(t, a.Sync(id))
	again, err := b.Versions(id)
	must(t, err)
	if len(again) != 1 {
		t.Fatal("相同哈希重复生成备份")
	}
	export := t.TempDir()
	must(t, b.ExportVersion(id, versions[0].ID, export))
	entries, err := os.ReadDir(export)
	must(t, err)
	if read(t, export, entries[0].Name()) != "初始\n" {
		t.Fatal("备份恢复内容错误")
	}
	must(t, b.Restore(id, versions[0].ID))
	must(t, a.Sync(id))
	if read(t, ra, "notes/a.md") != "初始\n" {
		t.Fatal("恢复未同步")
	}
}
func TestOfflineEditsMergeAndConflict(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "文档.md", "标题\n第一段\n中间\n最后一段\n")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	write(t, ra, "文档.md", "标题\n本机第一段\n中间\n最后一段\n")
	write(t, rb, "文档.md", "标题\n第一段\n中间\n对端最后一段\n")
	must(t, a.Sync(id))
	expected := "标题\n本机第一段\n中间\n对端最后一段\n"
	if read(t, ra, "文档.md") != expected || read(t, rb, "文档.md") != expected {
		t.Fatal("无冲突编辑未自动合并")
	}
	write(t, ra, "文档.md", "甲\n")
	write(t, rb, "文档.md", "乙\n")
	must(t, a.Sync(id))
	issues := a.existingIssues(id)
	if len(issues) != 1 || issues[0].Kind != "conflict" {
		t.Fatalf("未保留冲突：%+v", issues)
	}
	copyPath := issues[0].Copy
	if read(t, ra, "文档.md") == read(t, ra, copyPath) {
		t.Fatal("冲突内容丢失")
	}
	for _, p := range []string{"文档.md", copyPath} {
		if read(t, ra, p) != read(t, rb, p) {
			t.Fatal("双方保留版本不一致")
		}
	}
	must(t, a.Sync(id))
	entries, err := os.ReadDir(ra)
	must(t, err)
	if len(entries) != 2 {
		t.Fatal("重复产生冲突副本")
	}
	comparison, err := a.Compare(id, "文档.md")
	must(t, err)
	must(t, a.Resolve(id, "文档.md", comparison.LocalHash, comparison.RemoteHash, "最终结果\n", false))
	must(t, a.Sync(id))
	if read(t, rb, "文档.md") != "最终结果\n" {
		t.Fatal("手动合并失败")
	}
	if _, err = os.Stat(filepath.Join(ra, copyPath)); !os.IsNotExist(err) {
		t.Fatal("合并后冲突副本未收起")
	}
}
func TestDeleteRequiresConfirmation(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "a.txt", "保留")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	must(t, os.Remove(filepath.Join(ra, "a.txt")))
	must(t, a.Sync(id))
	if read(t, rb, "a.txt") != "保留" {
		t.Fatal("删除被自动传播")
	}
	if _, err := os.Stat(filepath.Join(ra, "a.txt")); !os.IsNotExist(err) {
		t.Fatal("删除被自动恢复")
	}
	issues := a.existingIssues(id)
	if len(issues) != 1 || issues[0].Kind != "delete" {
		t.Fatal("缺失删除确认")
	}
	must(t, a.ConfirmDelete(id, "a.txt", issues[0].Local.Hash, issues[0].Remote.Hash))
	if _, err := os.Stat(filepath.Join(rb, "a.txt")); !os.IsNotExist(err) {
		t.Fatal("确认后未删除")
	}
	versions, err := b.Versions(id)
	must(t, err)
	if len(versions) != 1 {
		t.Fatal("确认删除前未备份")
	}
}
func TestBackupFailureAndCompareAndSwap(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "a.txt", "旧")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	bad := filepath.Join(t.TempDir(), "not-a-directory")
	must(t, os.WriteFile(bad, []byte("x"), 0600))
	b.mu.Lock()
	b.settings.BackupDir = bad
	b.mu.Unlock()
	write(t, ra, "a.txt", "新")
	must(t, a.Sync(id))
	if read(t, rb, "a.txt") != "旧" {
		t.Fatal("备份失败仍覆盖")
	}
	issues := a.existingIssues(id)
	if len(issues) == 0 || issues[0].Kind != "error" {
		t.Fatal("备份失败未显示")
	}
	task, err := a.task(id)
	must(t, err)
	err = a.apply(task, "a.txt", "错误哈希", File{Hash: fingerprint([]byte("覆盖")), Size: 6, Modified: time.Now().UnixNano()}, strings.NewReader("覆盖"), "测试")
	if err == nil || read(t, ra, "a.txt") != "新" {
		t.Fatal("CAS 未保护新修改")
	}
}
func TestUnauthorizedAndUnsafePaths(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "a.txt", "安全")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	outsider := newDevice(t)
	var manifest Manifest
	if err := outsider.rpcAddress(address(b), b.id, "/manifest", map[string]string{"taskId": id}, &manifest); err == nil {
		t.Fatal("未配对设备能访问目录")
	}
	for _, p := range []string{"../escape", "/absolute", "x/../../escape", ".git/config", "x\\y", "C:/x", "CON.txt", ".lansync-evil", "trail. "} {
		if validPath(p) == nil {
			t.Fatalf("允许危险路径 %q", p)
		}
	}
	task, err := a.task(id)
	must(t, err)
	for _, p := range []string{"../escape", ".git/config"} {
		if err = a.apply(task, p, "", File{Hash: fingerprint([]byte("x")), Size: 1}, strings.NewReader("x"), "测试"); err == nil {
			t.Fatal("写入越界")
		}
	}
	must(t, os.MkdirAll(filepath.Join(ra, ".git"), 0700))
	write(t, ra, ".gitignore", "private.txt\n")
	write(t, ra, "private.txt", "秘密")
	must(t, a.Sync(id))
	body, err := b.remoteFile(b.tasks[id], "private.txt", File{Hash: fingerprint([]byte("秘密")), Size: 6})
	if err == nil {
		io.Copy(io.Discard, body)
		body.Close()
		t.Fatal("忽略文件可以通过接口读取")
	}
}
func TestInitialConflictAndBinary(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "document.docx", "A")
	write(t, rb, "document.docx", "B")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	issues := a.existingIssues(id)
	if len(issues) != 1 || issues[0].Copy == "" {
		t.Fatal("首次冲突未保留")
	}
	if _, err := a.Compare(id, "document.docx"); err == nil {
		t.Fatal("非 Markdown 提供差异")
	}
	if err := a.Resolve(id, "document.docx", "", "", "", true); err != nil {
		t.Fatal(err)
	}
}
func TestMergeMarkdown(t *testing.T) {
	tests := []struct {
		name, base, left, right, want string
		ok                            bool
	}{
		{"不同段落", "a\nb\nc\n", "A\nb\nc\n", "a\nb\nC\n", "A\nb\nC\n", true},
		{"同一行", "a\n", "A\n", "B\n", "", false},
		{"相同修改", "a\n", "A\n", "A\n", "A\n", true},
		{"同点插入", "", "甲", "乙", "", false},
		{"空文件单侧新增", "", "甲", "", "甲", true},
		{"相邻行", "a\nb\n", "A\nb\n", "a\nB\n", "A\nB\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := MergeMarkdown([]byte(tt.base), []byte(tt.left), []byte(tt.right))
			if ok != tt.ok || ok && string(got) != tt.want {
				t.Fatalf("got %q,%v; want %q,%v", got, ok, tt.want, tt.ok)
			}
		})
	}
	if _, ok := MergeMarkdown([]byte{}, bytes.Repeat([]byte("x"), MarkdownLimit+1), nil); ok {
		t.Fatal("过大文档进入合并")
	}
}
