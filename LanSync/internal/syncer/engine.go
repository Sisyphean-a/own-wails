package syncer

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Engine struct {
	mu           sync.RWMutex
	filesMu      sync.Mutex
	watchRefresh chan struct{}
	closeOnce    sync.Once
	closeErr     error
	closing      bool
	requests     sync.WaitGroup
	store        *Store
	dir          string
	cert         tls.Certificate
	id           string
	settings     Settings
	peers        map[string]Peer
	tasks        map[string]Task
	statuses     map[string]Status
	pairings     map[string]Pairing
	invitations  map[string]Invitation
	devices      map[string]Device
	contacts     map[string]int64
	locks        map[string]*sync.Mutex
	dirty        map[string]bool
	lastScan     map[string]time.Time
	server       *http.Server
	listener     net.Listener
	port         int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	slots        chan struct{}
	networkError string
	aiClient     *http.Client
	aiCancel     context.CancelFunc
}

func decode(b []byte, v any) error { return json.Unmarshal(b, v) }
func New(dir string) (*Engine, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	store, err := OpenStore(absolute)
	if err != nil {
		return nil, err
	}
	cert, err := loadIdentity(absolute)
	if err != nil {
		store.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{store: store, dir: absolute, cert: cert, id: fingerprint(cert.Certificate[0]), ctx: ctx, cancel: cancel, peers: map[string]Peer{}, tasks: map[string]Task{}, statuses: map[string]Status{}, pairings: map[string]Pairing{}, invitations: map[string]Invitation{}, devices: map[string]Device{}, contacts: map[string]int64{}, locks: map[string]*sync.Mutex{}, dirty: map[string]bool{}, lastScan: map[string]time.Time{}, slots: make(chan struct{}, 2), watchRefresh: make(chan struct{}, 1)}
	hostname, err := os.Hostname()
	if err != nil {
		store.Close()
		cancel()
		return nil, err
	}
	e.aiClient = newAIClient()
	e.settings = Settings{Name: hostname, BackupDir: filepath.Join(absolute, "backups"), Port: 49632}
	if _, err = store.get("settings", &e.settings); err != nil {
		store.Close()
		cancel()
		return nil, err
	}
	for _, prefix := range []string{"peer/", "task/", "invitation/", "status/"} {
		values, er := store.list(prefix)
		if er != nil {
			store.Close()
			cancel()
			return nil, er
		}
		for _, v := range values {
			switch prefix {
			case "peer/":
				var p Peer
				err = decode(v, &p)
				e.peers[p.ID] = p
			case "task/":
				var t Task
				err = decode(v, &t)
				e.tasks[t.ID] = t
				e.locks[t.ID] = &sync.Mutex{}
				e.dirty[t.ID] = true
			case "invitation/":
				var i Invitation
				err = decode(v, &i)
				e.invitations[i.ID] = i
			case "status/":
				var s Status
				err = decode(v, &s)
				s.State = "offline"
				s.Message = "等待连接"
				e.statuses[s.TaskID] = s
			}
			if err != nil {
				store.Close()
				cancel()
				return nil, err
			}
		}
	}
	return e, nil
}
func (e *Engine) Start(address string, discover bool) error {
	e.mu.Lock()
	if e.listener != nil {
		e.mu.Unlock()
		return fmt.Errorf("服务已启动")
	}
	if address == "" {
		address = fmt.Sprintf(":%d", e.settings.Port)
	}
	ln, err := net.Listen("tcp", address)
	if err != nil && address == fmt.Sprintf(":%d", e.settings.Port) {
		ln, err = net.Listen("tcp", ":0")
	}
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.listener = ln
	e.port = ln.Addr().(*net.TCPAddr).Port
	e.server = &http.Server{Handler: e.routes(), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 32 << 10, TLSConfig: &tls.Config{Certificates: []tls.Certificate{e.cert}, MinVersion: tls.VersionTLS13, ClientAuth: tls.RequireAnyClientCert}}
	srv := e.server
	tasks := []Task{}
	for _, t := range e.tasks {
		tasks = append(tasks, t)
	}
	e.mu.Unlock()
	for _, t := range tasks {
		if er := cleanupTransfers(t.Root); er != nil {
			e.status(t.ID, "error", er.Error(), nil, 0, 0, false)
		}
	}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		err := srv.Serve(tls.NewListener(ln, srv.TLSConfig))
		if err != nil && err != http.ErrServerClosed {
			e.mu.Lock()
			e.networkError = err.Error()
			e.mu.Unlock()
		}
	}()
	if discover {
		e.wg.Add(1)
		go e.discoveryLoop()
	}
	e.wg.Add(1)
	go e.background()
	e.wg.Add(1)
	go e.heartbeat()
	return nil
}
func (e *Engine) Close() error {
	e.closeOnce.Do(func() {
		e.mu.Lock()
		e.closing = true
		srv := e.server
		e.mu.Unlock()
		e.cancel()
		if srv != nil {
			e.closeErr = srv.Close()
		}
		e.requests.Wait()
		e.wg.Wait()
		if err := e.store.Close(); e.closeErr == nil {
			e.closeErr = err
		}
	})
	return e.closeErr
}
func (e *Engine) Snapshot() Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	s := Snapshot{ID: e.id, Settings: e.settings, Addresses: []string{}, Devices: []Device{}, Peers: []Peer{}, Pairings: []Pairing{}, Invitations: []Invitation{}, Tasks: []TaskView{}, NetworkError: e.networkError}
	if e.listener != nil && e.listener.Addr().(*net.TCPAddr).IP.IsLoopback() {
		s.Addresses = append(s.Addresses, e.listener.Addr().String())
	} else {
		for _, ip := range localIPs() {
			s.Addresses = append(s.Addresses, net.JoinHostPort(ip.String(), fmt.Sprint(e.port)))
		}
	}
	for _, d := range e.devices {
		if now()-d.Seen < 45000 {
			s.Devices = append(s.Devices, d)
		}
	}
	for _, p := range e.peers {
		p.Certificate = nil
		p.Online = now()-e.contacts[p.ID] < 30000
		s.Peers = append(s.Peers, p)
	}
	for _, p := range e.pairings {
		if p.Expires > now() && !(p.LocalApproved && p.RemoteApproved) {
			s.Pairings = append(s.Pairings, p)
		}
	}
	for _, i := range e.invitations {
		s.Invitations = append(s.Invitations, i)
	}
	for _, t := range e.tasks {
		status := e.statuses[t.ID]
		if status.State == "" {
			status = Status{TaskID: t.ID, State: "offline", Message: "等待连接", Issues: []Issue{}}
		}
		if t.RemoteRoot == "" {
			status.State = "invited"
			status.Message = "等待对端选择目录"
		} else if !t.Started {
			status.State = "ready"
			status.Message = "等待首次同步"
		} else if t.Paused {
			status.State = "paused"
			status.Message = "已暂停"
		} else if now()-e.contacts[t.PeerID] >= 30000 && status.State != "syncing" {
			status.State = "offline"
			status.Message = "离线 · 等待连接"
		}
		s.Tasks = append(s.Tasks, TaskView{t, status, e.peers[t.PeerID].Name})
	}
	sort.Slice(s.Tasks, func(i, j int) bool { return s.Tasks[i].Name < s.Tasks[j].Name })
	sort.Slice(s.Devices, func(i, j int) bool { return s.Devices[i].Name < s.Devices[j].Name })
	return s
}
func (e *Engine) SaveSettings(s Settings) error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" || len(s.Name) > 120 {
		return fmt.Errorf("设备名称需要 1–120 字节")
	}
	if s.Port < 0 || s.Port > 65535 {
		return fmt.Errorf("端口范围为 0–65535")
	}
	if err := os.MkdirAll(s.BackupDir, 0700); err != nil {
		return wrap("备份目录不可用，请重新选择", err)
	}
	dir, err := validateDirectory(s.BackupDir)
	if err != nil {
		return err
	}
	s.BackupDir = dir
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, t := range e.tasks {
		if containsPath(t.Root, dir) || containsPath(dir, t.Root) {
			return fmt.Errorf("备份目录不能与同步目录重叠")
		}
	}
	if err = e.store.put("settings", s); err != nil {
		return err
	}
	e.settings = s
	return nil
}
func (e *Engine) checkRoot(root string, except string) (string, error) {
	root, err := validateDirectory(root)
	if err != nil {
		return "", err
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if containsPath(root, e.dir) || containsPath(e.dir, root) || containsPath(root, e.settings.BackupDir) || containsPath(e.settings.BackupDir, root) {
		return "", fmt.Errorf("同步目录不能与应用数据或备份目录重叠")
	}
	for _, t := range e.tasks {
		if t.ID != except && (containsPath(root, t.Root) || containsPath(t.Root, root)) {
			return "", fmt.Errorf("同步目录不能相互重叠")
		}
	}
	for d := root; ; d = filepath.Dir(d) {
		if strings.EqualFold(filepath.Base(d), ".git") {
			return "", fmt.Errorf("不能同步 Git 元数据")
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	return root, nil
}
func (e *Engine) task(id string) (Task, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	t, ok := e.tasks[id]
	if !ok {
		return t, fmt.Errorf("同步目录不存在")
	}
	return t, nil
}
func (e *Engine) saveTask(t Task) error {
	if err := e.store.put("task/"+t.ID, t); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tasks[t.ID] = t
	if e.locks[t.ID] == nil {
		e.locks[t.ID] = &sync.Mutex{}
	}
	e.dirty[t.ID] = true
	select {
	case e.watchRefresh <- struct{}{}:
	default:
	}
	return nil
}
func (e *Engine) status(id, state, message string, issues []Issue, done, total int, completed bool) {
	e.mu.Lock()
	s := e.statuses[id]
	s.TaskID = id
	s.State = state
	s.Message = message
	s.Done = done
	s.Total = total
	if issues != nil {
		s.Issues = issues
	}
	if completed {
		s.LastSync = now()
	}
	e.statuses[id] = s
	e.mu.Unlock()
	if state != "syncing" {
		if err := e.store.put("status/"+id, s); err != nil {
			e.mu.Lock()
			e.networkError = "保存同步状态失败：" + err.Error()
			e.mu.Unlock()
		}
	}
}
func (e *Engine) Pause(id string, paused bool) error {
	t, err := e.task(id)
	if err != nil {
		return err
	}
	t.Paused = paused
	return e.saveTask(t)
}
func (e *Engine) RemoveTask(id string) error {
	t, err := e.task(id)
	if err != nil {
		return err
	}
	e.mu.RLock()
	lock := e.locks[id]
	e.mu.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	if err = e.rpc(t.PeerID, "/task/remove", map[string]string{"taskId": id}, nil); err != nil {
		return err
	}
	return e.removeTaskLocal(id)
}
func (e *Engine) removeTaskLocal(id string) error {
	if t, err := e.task(id); err == nil {
		if err = e.store.put("removed/"+id, t.PeerID); err != nil {
			return err
		}
	}
	if err := e.store.remove("task/" + id); err != nil {
		return err
	}
	e.mu.Lock()
	delete(e.tasks, id)
	delete(e.dirty, id)
	delete(e.statuses, id)
	e.mu.Unlock()
	return nil
}
func (e *Engine) Preview(id string) (Preview, error) {
	t, err := e.task(id)
	if err != nil {
		return Preview{}, err
	}
	a, err := scan(t.Root)
	if err != nil {
		return Preview{}, err
	}
	var b Manifest
	if err = e.rpc(t.PeerID, "/manifest", map[string]string{"taskId": id}, &b); err != nil {
		return Preview{}, err
	}
	p := Preview{}
	for file, l := range a.Files {
		if excluded(b, file) {
			continue
		}
		r := b.Files[file]
		if r.Hash == "" {
			p.AddedRemote++
		} else if l.Hash != r.Hash {
			p.Changed++
			p.Decisions++
		}
	}
	for file := range b.Files {
		if !excluded(a, file) && a.Files[file].Hash == "" {
			p.AddedLocal++
		}
	}
	return p, nil
}
func (e *Engine) Begin(id string) error {
	t, err := e.task(id)
	if err != nil {
		return err
	}
	if t.Coordinator != e.id {
		return e.rpc(t.PeerID, "/task/begin", map[string]string{"taskId": id}, nil)
	}
	if err = e.rpc(t.PeerID, "/task/start", map[string]string{"taskId": id}, nil); err != nil {
		return err
	}
	t.Started = true
	if err = e.saveTask(t); err != nil {
		return err
	}
	return e.RequestSync(id)
}
func (e *Engine) RequestSync(id string) error {
	t, err := e.task(id)
	if err != nil {
		return err
	}
	if t.Paused {
		return fmt.Errorf("目录已暂停")
	}
	if !t.Started {
		return fmt.Errorf("请先预览并开始首次同步")
	}
	if t.Coordinator != e.id {
		return e.rpc(t.PeerID, "/sync", map[string]string{"taskId": id}, nil)
	}
	e.mu.Lock()
	e.dirty[id] = true
	e.mu.Unlock()
	return nil
}
func (e *Engine) background() {
	defer e.wg.Done()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		e.mu.Lock()
		e.networkError = "文件监听失败：" + err.Error()
		e.mu.Unlock()
	}
	if watcher != nil {
		defer watcher.Close()
	}
	refresh := func() {
		if watcher == nil {
			return
		}
		for _, p := range watcher.WatchList() {
			watcher.Remove(p)
		}
		e.mu.RLock()
		tasks := make([]Task, 0, len(e.tasks))
		for _, t := range e.tasks {
			tasks = append(tasks, t)
		}
		e.mu.RUnlock()
		for _, t := range tasks {
			filepath.WalkDir(t.Root, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					if strings.EqualFold(d.Name(), ".git") {
						return filepath.SkipDir
					}
					if er := watcher.Add(p); er != nil {
						e.mu.Lock()
						e.networkError = "部分目录无法监听，将定期扫描：" + er.Error()
						e.mu.Unlock()
					}
				}
				return nil
			})
		}
	}
	refresh()
	lastRefresh := time.Now()
	needsRefresh := false
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	var events <-chan fsnotify.Event
	var errors <-chan error
	if watcher != nil {
		events = watcher.Events
		errors = watcher.Errors
	}
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-e.watchRefresh:
			refresh()
			lastRefresh = time.Now()
			needsRefresh = false
		case event := <-events:
			if strings.HasPrefix(filepath.Base(event.Name), ".lansync-") {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				if st, er := os.Stat(event.Name); er == nil && st.IsDir() {
					needsRefresh = true
				}
			}
			e.mu.Lock()
			for _, t := range e.tasks {
				if containsPath(t.Root, event.Name) {
					e.dirty[t.ID] = true
				}
			}
			e.mu.Unlock()
		case er := <-errors:
			if er != nil {
				e.mu.Lock()
				e.networkError = "文件监听异常：" + er.Error()
				e.mu.Unlock()
			}
		case <-ticker.C:
			if needsRefresh || time.Since(lastRefresh) > 30*time.Second {
				refresh()
				lastRefresh = time.Now()
				needsRefresh = false
			}
			e.mu.Lock()
			ids := []string{}
			for _, t := range e.tasks {
				if t.Started && !t.Paused && (e.dirty[t.ID] || time.Since(e.lastScan[t.ID]) > 60*time.Second) {
					ids = append(ids, t.ID)
					e.dirty[t.ID] = false
					e.lastScan[t.ID] = time.Now()
				}
			}
			for id, p := range e.pairings {
				if p.Expires < now() {
					delete(e.pairings, id)
				}
			}
			e.mu.Unlock()
			for _, id := range ids {
				select {
				case e.slots <- struct{}{}:
					e.wg.Add(1)
					go func(id string) {
						defer e.wg.Done()
						defer func() { <-e.slots }()
						t, er := e.task(id)
						if er == nil {
							if t.Coordinator == e.id {
								er = e.Sync(id)
							} else {
								er = e.rpc(t.PeerID, "/sync", map[string]string{"taskId": id}, nil)
							}
						}
						if er != nil {
							e.status(id, "offline", er.Error(), nil, 0, 0, false)
						}
					}(id)
				default:
					e.mu.Lock()
					e.dirty[id] = true
					e.mu.Unlock()
				}
			}
		}
	}
}
