package syncer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const jsonLimit = 8 << 20

func privateIP(ip net.IP) bool {
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast())
}
func validAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("请输入 IP:端口")
	}
	if !privateIP(net.ParseIP(host)) {
		return fmt.Errorf("只允许局域网或本机 IP")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("端口无效")
	}
	return nil
}
func (e *Engine) client(pin string) *http.Client {
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 15 * time.Second, DisableKeepAlives: true, TLSClientConfig: &tls.Config{
		Certificates: []tls.Certificate{e.cert}, MinVersion: tls.VersionTLS13,
		// Rule：自签设备证书使用持久指纹校验，不依赖 IP、DNS 名或公共 CA。
		InsecureSkipVerify: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) != 1 {
				return fmt.Errorf("设备证书无效")
			}
			cert := cs.PeerCertificates[0]
			id := fingerprint(cert.Raw)
			if pin != "" && id != pin {
				return fmt.Errorf("设备身份与配对记录不符")
			}
			return cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature)
		},
	}}
	return &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("不允许重定向") }}
}
func (e *Engine) request(ctx context.Context, address, pin, method, route string, body io.Reader) (*http.Response, error) {
	if err := validAddress(address); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, "https://"+address+route, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := e.client(pin)
	if strings.HasPrefix(route, "/apply?") || strings.HasPrefix(route, "/file?") {
		client.Transport.(*http.Transport).ResponseHeaderTimeout = 0
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, wrap("设备未连接", err)
	}
	if res.TLS != nil && len(res.TLS.PeerCertificates) == 1 {
		e.mu.Lock()
		e.contacts[fingerprint(res.TLS.PeerCertificates[0].Raw)] = now()
		e.mu.Unlock()
	}
	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(b)))
	}
	return res, nil
}
func (e *Engine) rpcAddress(address, pin, route string, in, out any) error {
	ctx, cancel := context.WithTimeout(e.ctx, 20*time.Second)
	defer cancel()
	res, err := e.request(ctx, address, pin, "POST", route, bytes.NewReader(encode(in)))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if out == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(res.Body, jsonLimit))
		return err
	}
	return json.NewDecoder(io.LimitReader(res.Body, jsonLimit)).Decode(out)
}
func (e *Engine) rpc(peerID, route string, in, out any) error {
	e.mu.RLock()
	p, ok := e.peers[peerID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("设备尚未配对")
	}
	return e.rpcAddress(p.Address, p.ID, route, in, out)
}
func requester(r *http.Request) (string, []byte, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !privateIP(net.ParseIP(host)) {
		return "", nil, fmt.Errorf("仅允许局域网访问")
	}
	if r.TLS == nil || len(r.TLS.PeerCertificates) != 1 {
		return "", nil, fmt.Errorf("缺少设备证书")
	}
	cert := r.TLS.PeerCertificates[0]
	if err = cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
		return "", nil, fmt.Errorf("设备证书签名无效")
	}
	return fingerprint(cert.Raw), cert.Raw, nil
}
func input(r *http.Request, v any) error {
	defer r.Body.Close()
	d := json.NewDecoder(io.LimitReader(r.Body, jsonLimit+1))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("请求无效：%w", err)
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return fmt.Errorf("请求包含多余数据")
	}
	return nil
}

type taskRequest struct {
	TaskID string `json:"taskId"`
}

func (e *Engine) authorized(r *http.Request, id string, write bool) (Task, error) {
	peer, _, err := requester(r)
	if err != nil {
		return Task{}, err
	}
	t, err := e.task(id)
	if err != nil {
		return t, err
	}
	if t.PeerID != peer {
		return t, fmt.Errorf("无权访问此同步目录")
	}
	e.mu.RLock()
	_, paired := e.peers[peer]
	e.mu.RUnlock()
	if !paired {
		return t, fmt.Errorf("设备未配对")
	}
	if t.Paused && r.URL.Path != "/task/remove" && r.URL.Path != "/task/info" {
		return t, fmt.Errorf("对端已暂停目录")
	}
	if write && !t.Started {
		return t, fmt.Errorf("尚未确认首次同步")
	}
	return t, nil
}
func respond(w http.ResponseWriter, v any, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if v == nil {
		v = struct{}{}
	}
	json.NewEncoder(w).Encode(v)
}
func (e *Engine) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /hello", func(w http.ResponseWriter, r *http.Request) {
		peer, _, err := requester(r)
		if err != nil {
			respond(w, nil, err)
			return
		}
		e.mu.RLock()
		name, port := e.settings.Name, e.port
		_, paired := e.peers[peer]
		e.mu.RUnlock()
		respond(w, map[string]any{"id": e.id, "name": name, "port": port, "paired": paired}, nil)
	})
	mux.HandleFunc("POST /pair/request", e.receivePair)
	mux.HandleFunc("POST /pair/approve", e.receiveApproval)
	mux.HandleFunc("POST /invite", e.receiveInvite)
	mux.HandleFunc("POST /invite/accept", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TaskID string `json:"taskId"`
			Root   string `json:"root"`
		}
		if err := input(r, &req); err != nil {
			respond(w, nil, err)
			return
		}
		t, err := e.authorized(r, req.TaskID, false)
		if err == nil {
			if t.RemoteRoot != "" && t.RemoteRoot != req.Root {
				err = fmt.Errorf("已关联其他目录，请移除后重新关联")
			} else {
				t.RemoteRoot = req.Root
				err = e.saveTask(t)
			}
		}
		respond(w, nil, err)
	})
	mux.HandleFunc("POST /task/info", func(w http.ResponseWriter, r *http.Request) {
		var req taskRequest
		if err := input(r, &req); err != nil {
			respond(w, nil, err)
			return
		}
		t, err := e.authorized(r, req.TaskID, false)
		respond(w, t, err)
	})
	for _, route := range []string{"/task/start", "/task/begin", "/task/remove", "/sync", "/manifest", "/baseline", "/status", "/delete", "/resolve"} {
		mux.HandleFunc("POST "+route, func(w http.ResponseWriter, r *http.Request) { e.taskRoute(w, r) })
	}
	mux.HandleFunc("GET /file", e.getFile)
	mux.HandleFunc("POST /apply", e.putFile)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.mu.Lock()
		if e.closing {
			e.mu.Unlock()
			http.Error(w, "正在退出", http.StatusServiceUnavailable)
			return
		}
		e.requests.Add(1)
		e.mu.Unlock()
		defer e.requests.Done()
		if id, _, err := requester(r); err == nil {
			e.mu.Lock()
			if _, ok := e.peers[id]; ok {
				e.contacts[id] = now()
			}
			e.mu.Unlock()
		}
		mux.ServeHTTP(w, r)
	})
}
func (e *Engine) taskRoute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID     string    `json:"taskId"`
		Path       string    `json:"path,omitempty"`
		Baseline   *Baseline `json:"baseline,omitempty"`
		Status     *Status   `json:"status,omitempty"`
		LocalHash  string    `json:"localHash,omitempty"`
		RemoteHash string    `json:"remoteHash,omitempty"`
		Content    string    `json:"content,omitempty"`
		KeepBoth   bool      `json:"keepBoth,omitempty"`
	}
	if err := input(r, &req); err != nil {
		respond(w, nil, err)
		return
	}
	t, err := e.authorized(r, req.TaskID, false)
	if r.URL.Path == "/task/remove" && err != nil {
		peer, _, authErr := requester(r)
		var removedPeer string
		ok, readErr := e.store.get("removed/"+req.TaskID, &removedPeer)
		if authErr == nil && readErr == nil && ok && peer == removedPeer {
			respond(w, nil, nil)
			return
		}
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	var result any
	switch r.URL.Path {
	case "/task/start":
		if t.Coordinator == t.PeerID {
			t.Started = true
			err = e.saveTask(t)
		} else {
			err = fmt.Errorf("仅协调设备可开始同步")
		}
	case "/task/begin":
		if t.Coordinator != e.id {
			err = fmt.Errorf("仅协调设备可执行")
		} else {
			err = e.Begin(t.ID)
		}
	case "/task/remove":
		err = e.removeTaskLocal(t.ID)
	case "/sync":
		if t.Coordinator != e.id {
			err = fmt.Errorf("仅协调设备可执行")
		} else {
			err = e.RequestSync(t.ID)
		}
	case "/manifest":
		if t.Started && t.Coordinator == t.PeerID {
			e.status(t.ID, "syncing", "正在核对", nil, 0, 0, false)
		}
		result, err = scan(t.Root)
	case "/baseline":
		if t.Coordinator != t.PeerID || !t.Started {
			err = fmt.Errorf("无权更新共同版本")
			break
		}
		if err = validPath(req.Path); err != nil {
			break
		}
		if req.Baseline == nil {
			err = e.store.remove("base/" + t.ID + "/" + req.Path)
		} else {
			err = e.store.put("base/"+t.ID+"/"+req.Path, req.Baseline)
		}
	case "/status":
		if t.Coordinator != t.PeerID || req.Status == nil {
			err = fmt.Errorf("无权更新状态")
			break
		}
		s := *req.Status
		s.TaskID = t.ID
		e.mu.Lock()
		e.statuses[t.ID] = s
		e.mu.Unlock()
		err = e.store.put("status/"+t.ID, s)
	case "/delete":
		if t.Coordinator != e.id {
			err = fmt.Errorf("仅协调设备可执行")
		} else {
			err = e.ConfirmDelete(t.ID, req.Path, req.LocalHash, req.RemoteHash)
		}
	case "/resolve":
		if t.Coordinator != e.id {
			err = fmt.Errorf("仅协调设备可执行")
		} else {
			err = e.Resolve(t.ID, req.Path, req.LocalHash, req.RemoteHash, req.Content, req.KeepBoth)
		}
	}
	respond(w, result, err)
}
func (e *Engine) getFile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	t, err := e.authorized(r, q.Get("task"), false)
	if err != nil {
		respond(w, nil, err)
		return
	}
	p := q.Get("path")
	if err = validPath(p); err != nil {
		respond(w, nil, err)
		return
	}
	ignored, err := ignoredPath(t.Root, p)
	if err != nil || ignored {
		if err == nil {
			err = fmt.Errorf("文件已被忽略，禁止读取")
		}
		respond(w, nil, err)
		return
	}
	root, err := openRoot(t.Root)
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer root.Close()
	f, err := fileInfo(root, p)
	if err != nil || f.Hash == "" || f.Hash != q.Get("hash") {
		if err == nil {
			err = fmt.Errorf("文件已变化")
		}
		respond(w, nil, err)
		return
	}
	file, err := root.Open(p)
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprint(f.Size))
	io.Copy(w, file)
}
func (e *Engine) putFile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	t, err := e.authorized(r, q.Get("task"), true)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if t.Coordinator != t.PeerID {
		respond(w, nil, fmt.Errorf("仅协调设备可写入"))
		return
	}
	size, err := strconv.ParseInt(q.Get("size"), 10, 64)
	if err != nil || size < 0 || size == int64(^uint64(0)>>1) {
		respond(w, nil, fmt.Errorf("文件长度无效"))
		return
	}
	modified, err := strconv.ParseInt(q.Get("modified"), 10, 64)
	if err != nil {
		respond(w, nil, err)
		return
	}
	hash := q.Get("hash")
	if hash != "" && len(hash) != 64 {
		respond(w, nil, fmt.Errorf("哈希无效"))
		return
	}
	err = e.apply(t, q.Get("path"), q.Get("expected"), File{Hash: hash, Size: size, Modified: modified}, r.Body, "同步前版本")
	respond(w, nil, err)
}
func (e *Engine) remoteFile(t Task, p string, f File) (io.ReadCloser, error) {
	e.mu.RLock()
	peer := e.peers[t.PeerID]
	e.mu.RUnlock()
	route := "/file?" + url.Values{"task": {t.ID}, "path": {p}, "hash": {f.Hash}}.Encode()
	res, err := e.request(e.ctx, peer.Address, peer.ID, "GET", route, nil)
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}
func (e *Engine) remoteApply(t Task, p, expected string, f File, body io.Reader) error {
	e.mu.RLock()
	peer := e.peers[t.PeerID]
	e.mu.RUnlock()
	route := "/apply?" + url.Values{"task": {t.ID}, "path": {p}, "expected": {expected}, "hash": {f.Hash}, "size": {fmt.Sprint(f.Size)}, "modified": {fmt.Sprint(f.Modified)}}.Encode()
	res, err := e.request(e.ctx, peer.Address, peer.ID, "POST", route, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	return err
}

type pairRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Port int    `json:"port"`
}

func pairCode(id, a, b string) string {
	ids := []string{a, b}
	sort.Strings(ids)
	h := sha256.Sum256([]byte(id + ids[0] + ids[1]))
	s := strings.ToUpper(hex.EncodeToString(h[:4]))
	return s[:4] + " " + s[4:]
}
func (e *Engine) Connect(address string) error {
	if err := validAddress(address); err != nil {
		return err
	}
	var hello struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Port   int    `json:"port"`
		Paired bool   `json:"paired"`
	}
	ctx, cancel := context.WithTimeout(e.ctx, 10*time.Second)
	defer cancel()
	res, err := e.request(ctx, address, "", "POST", "/hello", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	cert := res.TLS.PeerCertificates[0].Raw
	err = json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&hello)
	res.Body.Close()
	if err != nil {
		return err
	}
	peerID := fingerprint(cert)
	if hello.ID != peerID || peerID == e.id {
		return fmt.Errorf("不能连接当前设备或身份无效")
	}
	e.mu.Lock()
	if p, ok := e.peers[peerID]; ok && hello.Paired {
		p.Address = address
		err = e.store.put("peer/"+p.ID, p)
		if err == nil {
			e.peers[p.ID] = p
		}
		e.mu.Unlock()
		return err
	}
	if len(e.pairings) >= 16 {
		e.mu.Unlock()
		return fmt.Errorf("待配对请求过多")
	}
	id := randomID()
	p := Pairing{ID: id, PeerID: peerID, Name: hello.Name, Address: address, Code: pairCode(id, e.id, peerID), Expires: now() + 600000, Certificate: cert}
	e.pairings[id] = p
	name, port := e.settings.Name, e.port
	e.mu.Unlock()
	err = e.rpcAddress(address, peerID, "/pair/request", pairRequest{id, name, port}, nil)
	if err != nil {
		e.mu.Lock()
		delete(e.pairings, id)
		e.mu.Unlock()
	}
	return err
}
func (e *Engine) receivePair(w http.ResponseWriter, r *http.Request) {
	id, cert, err := requester(r)
	if err != nil {
		respond(w, nil, err)
		return
	}
	var req pairRequest
	if err = input(r, &req); err != nil {
		respond(w, nil, err)
		return
	}
	if !validID(req.ID) || len(req.Name) > 120 || req.Port < 1 || req.Port > 65535 {
		respond(w, nil, fmt.Errorf("配对请求无效"))
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	e.mu.Lock()
	if len(e.pairings) >= 16 {
		e.mu.Unlock()
		respond(w, nil, fmt.Errorf("待配对请求过多"))
		return
	}
	if _, exists := e.pairings[req.ID]; !exists {
		e.pairings[req.ID] = Pairing{ID: req.ID, PeerID: id, Name: req.Name, Address: net.JoinHostPort(host, strconv.Itoa(req.Port)), Code: pairCode(req.ID, e.id, id), Expires: now() + 600000, Certificate: cert}
	}
	e.mu.Unlock()
	respond(w, nil, nil)
}
func (e *Engine) finishPair(p Pairing) error {
	if !p.LocalApproved || !p.RemoteApproved {
		return nil
	}
	peer := Peer{ID: p.PeerID, Name: p.Name, Address: p.Address, Certificate: p.Certificate}
	if err := e.store.put("peer/"+peer.ID, peer); err != nil {
		return err
	}
	e.peers[peer.ID] = peer
	e.contacts[peer.ID] = now()
	// Rule：短期保留已确认请求，允许丢失响应后的幂等确认。
	return nil
}
func (e *Engine) ApprovePair(id string) error {
	e.mu.Lock()
	p, ok := e.pairings[id]
	if !ok || p.Expires < now() {
		e.mu.Unlock()
		return fmt.Errorf("配对请求已过期")
	}
	p.LocalApproved = true
	e.pairings[id] = p
	e.mu.Unlock()
	if err := e.rpcAddress(p.Address, p.PeerID, "/pair/approve", map[string]string{"id": id}, nil); err != nil {
		e.mu.Lock()
		if _, paired := e.peers[p.PeerID]; !paired {
			current := e.pairings[id]
			current.LocalApproved = false
			e.pairings[id] = current
		}
		e.mu.Unlock()
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	p = e.pairings[id]
	return e.finishPair(p)
}
func (e *Engine) receiveApproval(w http.ResponseWriter, r *http.Request) {
	peer, _, err := requester(r)
	if err != nil {
		respond(w, nil, err)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err = input(r, &req); err != nil {
		respond(w, nil, err)
		return
	}
	e.mu.Lock()
	p, ok := e.pairings[req.ID]
	if !ok || p.PeerID != peer || p.Expires < now() {
		err = fmt.Errorf("配对请求已过期")
	} else {
		p.RemoteApproved = true
		e.pairings[p.ID] = p
		err = e.finishPair(p)
	}
	e.mu.Unlock()
	respond(w, nil, err)
}
func (e *Engine) DismissPair(id string) { e.mu.Lock(); delete(e.pairings, id); e.mu.Unlock() }
func (e *Engine) Invite(peerID, name, root string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return fmt.Errorf("目录名称需要 1–120 字节")
	}
	root, err := e.checkRoot(root, "")
	if err != nil {
		return err
	}
	e.mu.RLock()
	_, ok := e.peers[peerID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("请先配对设备")
	}
	t := Task{ID: randomID(), Name: name, PeerID: peerID, Root: root, Coordinator: e.id}
	if err = e.saveTask(t); err != nil {
		return err
	}
	inv := Invitation{ID: t.ID, Name: name, PeerID: e.id, RemoteRoot: root, Coordinator: e.id}
	if err = e.rpc(peerID, "/invite", inv, nil); err != nil {
		e.removeTaskLocal(t.ID)
		return err
	}
	return nil
}
func (e *Engine) receiveInvite(w http.ResponseWriter, r *http.Request) {
	peer, _, err := requester(r)
	if err != nil {
		respond(w, nil, err)
		return
	}
	e.mu.RLock()
	_, paired := e.peers[peer]
	e.mu.RUnlock()
	if !paired {
		respond(w, nil, fmt.Errorf("设备未配对"))
		return
	}
	var inv Invitation
	if err = input(r, &inv); err != nil {
		respond(w, nil, err)
		return
	}
	if !validID(inv.ID) || inv.Coordinator != peer || inv.PeerID != peer || len(inv.Name) > 120 {
		respond(w, nil, fmt.Errorf("邀请无效"))
		return
	}
	e.mu.Lock()
	if _, ok := e.tasks[inv.ID]; !ok {
		err = e.store.put("invitation/"+inv.ID, inv)
		if err == nil {
			e.invitations[inv.ID] = inv
		}
	}
	e.mu.Unlock()
	respond(w, nil, err)
}
func (e *Engine) AcceptInvite(id, root string) error {
	e.mu.RLock()
	inv, ok := e.invitations[id]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("邀请不存在")
	}
	root, err := e.checkRoot(root, id)
	if err != nil {
		return err
	}
	t := Task{ID: id, Name: inv.Name, PeerID: inv.PeerID, Root: root, RemoteRoot: inv.RemoteRoot, Coordinator: inv.Coordinator}
	if existing, er := e.task(id); er == nil {
		if existing.Root != root {
			return fmt.Errorf("已选择其他目录")
		}
		t = existing
	}
	if err = e.saveTask(t); err != nil {
		return err
	}
	if err = e.rpc(t.PeerID, "/invite/accept", map[string]string{"taskId": id, "root": root}, nil); err != nil {
		return err
	}
	if err = e.store.remove("invitation/" + id); err != nil {
		return err
	}
	e.mu.Lock()
	delete(e.invitations, id)
	e.mu.Unlock()
	return nil
}
