package syncer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	gitindex "github.com/go-git/go-git/v5/plumbing/format/index"
)

func TestRestartAtNewPortRetainsIdentityAndBaseline(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "a.md", "头\n中\n尾\n")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	oldID, oldPort := b.id, b.port
	must(t, b.Close())
	tempName := ".lansync-" + randomID()
	write(t, rb, tempName, "中断传输")
	write(t, rb, "draft.tmp", "用户文件")
	write(t, ra, "a.md", "新头\n中\n尾\n")
	write(t, rb, "a.md", "头\n中\n新尾\n")
	if err := a.Sync(id); err == nil {
		t.Fatal("离线时报告同步成功")
	}
	restarted, err := New(b.dir)
	must(t, err)
	must(t, restarted.Start("127.0.0.1:0", false))
	t.Cleanup(func() { must(t, restarted.Close()) })
	if _, err = os.Stat(filepath.Join(rb, tempName)); !os.IsNotExist(err) {
		t.Fatal("重启未清理保留命名空间的传输临时文件")
	}
	if read(t, rb, "draft.tmp") != "用户文件" {
		t.Fatal("清理误删用户文件")
	}
	if restarted.id != oldID {
		t.Fatal("重启更换了设备身份")
	}
	if restarted.port == oldPort {
		t.Log("操作系统复用了端口，身份与状态恢复仍可验证")
	}
	if len(restarted.peers) != 1 || len(restarted.peers[a.id].Certificate) == 0 {
		t.Fatal("配对证书未持久化")
	}
	must(t, a.Connect(address(restarted)))
	if len(a.Snapshot().Pairings) != 0 {
		t.Fatal("已配对设备换端口后要求再次配对")
	}
	must(t, a.Sync(id))
	want := "新头\n中\n新尾\n"
	if read(t, ra, "a.md") != want || read(t, rb, "a.md") != want {
		t.Fatal("重连未使用共同基准合并")
	}
}
func TestGitIgnoreNestedNegationAndTrackedFiles(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	must(t, os.MkdirAll(filepath.Join(ra, ".git", "info"), 0700))
	f, err := os.Create(filepath.Join(ra, ".git", "index"))
	must(t, err)
	must(t, gitindex.NewEncoder(f).Encode(&gitindex.Index{Version: 2, Entries: []*gitindex.Entry{{Name: "tracked.txt", Mode: filemode.Regular}}}))
	must(t, f.Close())
	write(t, ra, ".gitignore", "*.txt\n!keep.txt\nbuild/\n")
	write(t, ra, ".git/info/exclude", "local-only.md\n")
	for p, c := range map[string]string{"tracked.txt": "已跟踪", "keep.txt": "保留", "skip.txt": "忽略", "local-only.md": "本地规则", "build/data.md": "构建", "sub/.gitignore": "!nested.txt\n", "sub/nested.txt": "嵌套", "sub/skip.txt": "忽略"} {
		write(t, ra, p, c)
	}
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	for _, p := range []string{"tracked.txt", "keep.txt", "sub/nested.txt"} {
		if read(t, rb, p) == "" {
			t.Fatal("应同步的文件缺失", p)
		}
	}
	for _, p := range []string{"skip.txt", "local-only.md", "build/data.md", "sub/skip.txt", ".git/index"} {
		if _, err = os.Stat(filepath.Join(rb, filepath.FromSlash(p))); !os.IsNotExist(err) {
			t.Fatal("忽略文件被同步", p)
		}
	}
}
func TestHashDetectsSameSizeAndTimestampChanges(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "a.txt", "A")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	st, err := os.Stat(filepath.Join(ra, "a.txt"))
	must(t, err)
	write(t, ra, "a.txt", "B")
	must(t, os.Chtimes(filepath.Join(ra, "a.txt"), st.ModTime(), st.ModTime()))
	must(t, a.Sync(id))
	if read(t, rb, "a.txt") != "B" {
		t.Fatal("仅使用大小和时间判断文件相同")
	}
}
func TestDirectoryBoundariesAndInterruptedTransfer(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "a.txt", "原内容")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	if err := a.Invite(b.id, "重复", ra); err == nil {
		t.Fatal("允许重叠目录")
	}
	if err := a.Invite(b.id, "应用数据", a.dir); err == nil {
		t.Fatal("允许同步身份数据")
	}
	settings := a.settings
	settings.BackupDir = filepath.Join(ra, "backups")
	if err := a.SaveSettings(settings); err == nil {
		t.Fatal("允许在同步目录中备份")
	}
	task, err := a.task(id)
	must(t, err)
	old := fingerprint([]byte("原内容"))
	err = a.apply(task, "a.txt", old, File{Hash: fingerprint([]byte("完整新文件")), Size: 15}, &brokenReader{}, "中断")
	if err == nil || read(t, ra, "a.txt") != "原内容" {
		t.Fatal("中断传输破坏原内容")
	}
}

type brokenReader struct{}

func (*brokenReader) Read(p []byte) (int, error) { return 0, fmt.Errorf("模拟传输中断") }
func TestSymlinkDoesNotEscapeRoot(t *testing.T) {
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	outside := t.TempDir()
	write(t, outside, "secret.md", "外部内容")
	if err := os.Symlink(outside, filepath.Join(ra, "link")); err != nil {
		t.Skip("当前系统不允许创建符号链接：", err)
	}
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	if _, err := os.Stat(filepath.Join(rb, "link", "secret.md")); !os.IsNotExist(err) {
		t.Fatal("符号链接内容泄露")
	}
}
