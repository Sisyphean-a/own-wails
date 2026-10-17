package syncer

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

func localIPs() []net.IP {
	out := []net.IP{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err == nil && ip.To4() != nil && privateIP(ip) && !ip.IsLoopback() {
			out = append(out, ip)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}
func (e *Engine) discoveryLoop() {
	defer e.wg.Done()
	var server *zeroconf.Server
	previous := ""
	defer func() {
		if server != nil {
			server.Shutdown()
		}
	}()
	for {
		ips := []string{}
		for _, ip := range localIPs() {
			ips = append(ips, ip.String())
		}
		e.mu.RLock()
		name, port := e.settings.Name, e.port
		e.mu.RUnlock()
		signature := strings.Join(ips, ",") + name
		if signature != previous || server == nil {
			if server != nil {
				server.Shutdown()
				server = nil
			}
			if len(ips) > 0 {
				var err error
				server, err = zeroconf.RegisterProxy("LanSync-"+e.id[:12], "_lansync._tcp", "local.", port, "lansync-"+e.id[:12]+".local.", ips, []string{"id=" + e.id, "name=" + name}, nil)
				if err != nil {
					e.mu.Lock()
					e.networkError = "局域网发现不可用，请使用 IP:端口连接：" + err.Error()
					e.mu.Unlock()
				}
			}
			previous = signature
		}
		resolver, err := zeroconf.NewResolver(nil)
		if err == nil {
			ctx, cancel := context.WithTimeout(e.ctx, 4*time.Second)
			entries := make(chan *zeroconf.ServiceEntry, 32)
			err = resolver.Browse(ctx, "_lansync._tcp", "local.", entries)
			if err == nil {
			loop:
				for {
					select {
					case <-ctx.Done():
						break loop
					case entry, ok := <-entries:
						if !ok {
							break loop
						}
						e.observe(entry)
					}
				}
			}
			cancel()
		}
		if err != nil {
			e.mu.Lock()
			e.networkError = "局域网扫描失败，可输入地址连接：" + err.Error()
			e.mu.Unlock()
		}
		select {
		case <-e.ctx.Done():
			return
		case <-time.After(8 * time.Second):
		}
	}
}
func (e *Engine) observe(entry *zeroconf.ServiceEntry) {
	id, name := "", ""
	for _, s := range entry.Text {
		if strings.HasPrefix(s, "id=") {
			id = strings.TrimPrefix(s, "id=")
		}
		if strings.HasPrefix(s, "name=") {
			name = strings.TrimPrefix(s, "name=")
		}
	}
	if len(id) != 64 || id == e.id || len(name) > 120 {
		return
	}
	for _, ip := range entry.AddrIPv4 {
		if !privateIP(ip) {
			continue
		}
		address := net.JoinHostPort(ip.String(), fmt.Sprint(entry.Port))
		e.mu.Lock()
		e.devices[id] = Device{ID: id, Name: name, Address: address, Seen: now()}
		p, paired := e.peers[id]
		e.mu.Unlock()
		if paired {
			var hello struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Port int    `json:"port"`
			}
			if err := e.rpcAddress(address, id, "/hello", struct{}{}, &hello); err == nil && hello.ID == id {
				p.Address = address
				p.Name = hello.Name
				if err = e.store.put("peer/"+id, p); err != nil {
					e.mu.Lock()
					e.networkError = err.Error()
					e.mu.Unlock()
					return
				}
				e.mu.Lock()
				e.peers[id] = p
				for _, t := range e.tasks {
					if t.PeerID == id {
						if s := e.statuses[t.ID]; s.State == "offline" || s.State == "error" {
							e.dirty[t.ID] = true
						}
					}
				}
				e.mu.Unlock()
				return
			}
		}
	}
}
