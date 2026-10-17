package syncer

import (
	"net"
	"testing"

	"github.com/grandcat/zeroconf"
)

func TestDiscoveryPinsIdentityWhenPortChanges(t *testing.T) {
	a, b := paired(t)
	must(t, b.Close())
	restarted, err := New(b.dir)
	must(t, err)
	must(t, restarted.Start("127.0.0.1:0", false))
	t.Cleanup(func() { must(t, restarted.Close()) })
	entry := &zeroconf.ServiceEntry{}
	entry.Port = restarted.port
	entry.Text = []string{"id=" + b.id, "name=笔记本"}
	entry.AddrIPv4 = []net.IP{net.ParseIP("127.0.0.1")}
	a.observe(entry)
	if a.Snapshot().Peers[0].Address != address(restarted) {
		t.Fatal("发现新的端口后未更新已配对设备")
	}
	outsider := newDevice(t)
	entry.Port = outsider.port
	a.observe(entry)
	if a.Snapshot().Peers[0].Address != address(restarted) {
		t.Fatal("身份不符的广播覆盖了已配对地址")
	}
}
