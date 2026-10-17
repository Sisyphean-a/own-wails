package syncer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

func openRoot(p string) (*os.Root, error) {
	if _, err := validateDirectory(p); err != nil {
		return nil, err
	}
	return os.OpenRoot(p)
}
func (e *Engine) transfer(t Task, source, dest string, f File, expected string, fromLocal, toLocal bool) error {
	var body io.ReadCloser
	var err error
	if fromLocal {
		root, er := openRoot(t.Root)
		if er != nil {
			return er
		}
		if er = checkNoLinks(root, source); er != nil {
			root.Close()
			return er
		}
		body, err = root.Open(source)
		root.Close()
	} else {
		body, err = e.remoteFile(t, source, f)
	}
	if err != nil {
		return err
	}
	defer body.Close()
	if toLocal {
		return e.apply(t, dest, expected, f, body, "同步前版本")
	}
	return e.remoteApply(t, dest, expected, f, body)
}
func (e *Engine) baseline(t Task, p string, f File) error {
	b := Baseline{File: f}
	if strings.EqualFold(path.Ext(p), ".md") && f.Size <= MarkdownLimit && f.Hash != "" {
		content, err := readContent(t.Root, p, f.Hash, MarkdownLimit)
		if err != nil {
			return err
		}
		b.Content = content
	}
	if err := e.rpc(t.PeerID, "/baseline", map[string]any{"taskId": t.ID, "path": p, "baseline": b}, nil); err != nil {
		return err
	}
	return e.store.put("base/"+t.ID+"/"+p, b)
}
func (e *Engine) existingIssues(id string) []Issue {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]Issue{}, e.statuses[id].Issues...)
}
func (e *Engine) Sync(id string) (err error) {
	t, err := e.task(id)
	if err != nil {
		return err
	}
	if t.Coordinator != e.id {
		return fmt.Errorf("仅协调设备执行同步")
	}
	if t.Paused || !t.Started {
		return fmt.Errorf("目录尚未开始或已暂停")
	}
	e.mu.RLock()
	lock := e.locks[id]
	e.mu.RUnlock()
	if !lock.TryLock() {
		return nil
	}
	defer lock.Unlock()
	oldIssues := e.existingIssues(id)
	issues := []Issue{}
	for _, issue := range oldIssues {
		if issue.Kind == "conflict" {
			issues = append(issues, issue)
		}
	}
	e.status(id, "syncing", "正在核对", issues, 0, 0, false)
	defer func() {
		if err != nil {
			e.status(id, "error", err.Error(), issues, 0, 0, false)
		}
		e.mu.RLock()
		s := e.statuses[id]
		e.mu.RUnlock()
		if er := e.rpc(t.PeerID, "/status", map[string]any{"taskId": t.ID, "status": s}, nil); er != nil && err == nil {
			err = er
			e.status(id, "offline", "等待连接："+er.Error(), issues, 0, 0, false)
		}
	}()
	a, err := scan(t.Root)
	if err != nil {
		return err
	}
	var b Manifest
	if err = e.rpc(t.PeerID, "/manifest", map[string]string{"taskId": id}, &b); err != nil {
		e.status(id, "offline", "等待连接", issues, 0, 0, false)
		return err
	}
	skipped := map[string]bool{}
	for _, p := range a.Ignored {
		skipped[p] = true
	}
	for _, p := range b.Ignored {
		skipped[p] = true
	}
	list := []string{}
	for p := range skipped {
		list = append(list, p)
	}
	sort.Strings(list)
	e.mu.Lock()
	currentStatus := e.statuses[id]
	currentStatus.Skipped = list
	e.statuses[id] = currentStatus
	e.mu.Unlock()
	bases, err := e.store.baselineMetadata(id)
	if err != nil {
		return err
	}
	all := map[string]bool{}
	for p := range a.Files {
		all[p] = true
	}
	for p := range b.Files {
		all[p] = true
	}
	for k := range bases {
		all[strings.TrimPrefix(k, "base/"+id+"/")] = true
	}
	for p, msg := range a.Problems {
		issues = append(issues, Issue{Path: p, Kind: "error", Message: "本机：" + msg})
	}
	for p, msg := range b.Problems {
		issues = append(issues, Issue{Path: p, Kind: "error", Message: "对端：" + msg})
	}
	paths := []string{}
	for p := range all {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	merged, kept := 0, 0
	for index, p := range paths {
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		default:
		}
		current, er := e.task(id)
		if er != nil {
			return er
		}
		if current.Paused {
			return fmt.Errorf("已暂停")
		}
		if excluded(a, p) || excluded(b, p) {
			continue
		}
		left, right := a.Files[p], b.Files[p]
		e.status(id, "syncing", p, issues, index, len(paths), false)
		var base Baseline
		if file, ok := bases["base/"+id+"/"+p]; ok {
			base.File = file
		}
		switch {
		case left.Hash == right.Hash:
			if left.Hash != "" && base.File.Hash != left.Hash {
				if er = e.baseline(t, p, left); er != nil {
					return er
				}
			}
		case base.File.Hash != "" && (left.Hash == "" || right.Hash == ""):
			issues = append(issues, Issue{Path: p, Kind: "delete", Message: "删除待确认", Local: left, Remote: right})
		case left.Hash == "":
			er = e.transfer(t, p, p, right, "", false, true)
			if er == nil {
				er = e.baseline(t, p, right)
			}
		case right.Hash == "":
			er = e.transfer(t, p, p, left, "", true, false)
			if er == nil {
				er = e.baseline(t, p, left)
			}
		case left.Hash == base.File.Hash:
			er = e.transfer(t, p, p, right, left.Hash, false, true)
			if er == nil {
				er = e.baseline(t, p, right)
			}
		case right.Hash == base.File.Hash:
			er = e.transfer(t, p, p, left, right.Hash, true, false)
			if er == nil {
				er = e.baseline(t, p, left)
			}
		default:
			if base.File.Hash != "" && strings.EqualFold(path.Ext(p), ".md") && left.Size <= MarkdownLimit && right.Size <= MarkdownLimit && base.File.Size <= MarkdownLimit {
				if _, er = e.store.get("base/"+id+"/"+p, &base); er != nil {
					return er
				}
				if fingerprint(base.Content) != base.File.Hash {
					return fmt.Errorf("共同版本校验失败：%s", p)
				}
				l, readErr := readContent(t.Root, p, left.Hash, MarkdownLimit)
				if readErr != nil {
					return readErr
				}
				remote, readErr := e.remoteFile(t, p, right)
				if readErr != nil {
					return readErr
				}
				r, readErr := io.ReadAll(io.LimitReader(remote, MarkdownLimit+1))
				remote.Close()
				if readErr != nil {
					return readErr
				}
				if fingerprint(r) != right.Hash {
					return fmt.Errorf("对端文件已变化")
				}
				if content, ok := MergeMarkdown(base.Content, l, r); ok {
					info := File{Hash: fingerprint(content), Size: int64(len(content)), Modified: time.Now().UnixNano()}
					if er = e.apply(t, p, left.Hash, info, bytes.NewReader(content), "自动合并前版本"); er == nil {
						er = e.remoteApply(t, p, right.Hash, info, bytes.NewReader(content))
					}
					if er == nil {
						er = e.baseline(t, p, info)
						merged++
					}
					break
				}
			}
			// Rule：首次同名差异也保留双方，不把不可信的时钟当作丢弃依据。
			copyPath, info, conflictErr := e.preserveConflict(t, p, left, right, a, b)
			er = conflictErr
			if er == nil {
				kept++
				found := false
				for _, issue := range issues {
					if issue.Path == p && issue.Copy == copyPath {
						found = true
					}
				}
				if !found {
					issues = append(issues, Issue{Path: p, Kind: "conflict", Message: "已保留双方版本", Local: left, Remote: right, Copy: copyPath})
				}
				a.Files[p] = info
				b.Files[p] = info
			}
		}
		if er != nil {
			issues = append(issues, Issue{Path: p, Kind: "error", Message: er.Error(), Local: left, Remote: right})
		}
	}
	state, message := "synced", "已同步"
	if len(issues) > 0 {
		state = "attention"
		message = fmt.Sprintf("%d 项记录", len(issues))
	} else if merged > 0 {
		message = fmt.Sprintf("自动合并了 %d 项", merged)
	}
	if kept > 0 {
		message = fmt.Sprintf("保留了 %d 组版本", kept)
	}
	// Failure：部分文件失败不是一次成功同步。
	complete := true
	for _, issue := range issues {
		if issue.Kind == "error" || issue.Kind == "delete" {
			complete = false
		}
	}
	e.status(id, state, message, issues, len(paths), len(paths), complete)
	return nil
}
func conflictName(p, device, hash string) string {
	ext := path.Ext(p)
	stem := strings.TrimSuffix(path.Base(p), ext)
	for len(stem) > 100 {
		_, n := utf8.DecodeLastRuneInString(stem)
		stem = stem[:len(stem)-n]
	}
	return path.Join(path.Dir(p), stem+"（冲突-"+device[:8]+"-"+hash[:12]+"）"+ext)
}
func (e *Engine) preserveConflict(t Task, p string, left, right File, a, b Manifest) (string, File, error) {
	leftWins := left.Modified > right.Modified
	if left.Modified == right.Modified || left.Modified > time.Now().Add(5*time.Minute).UnixNano() || right.Modified > time.Now().Add(5*time.Minute).UnixNano() {
		leftWins = e.id < t.PeerID
	}
	loser, device := right, t.PeerID
	if !leftWins {
		loser, device = left, e.id
	}
	copyPath := conflictName(p, device, loser.Hash)
	if excluded(a, copyPath) || excluded(b, copyPath) {
		return "", File{}, fmt.Errorf("冲突副本路径被忽略或不可用")
	}
	for _, m := range []Manifest{a, b} {
		if existing := m.Files[copyPath]; existing.Hash != "" && existing.Hash != loser.Hash {
			return "", File{}, fmt.Errorf("冲突副本路径已有其他内容，未覆盖")
		}
	}
	if a.Files[copyPath].Hash != loser.Hash {
		if err := e.transfer(t, p, copyPath, loser, "", !leftWins, true); err != nil {
			return "", File{}, err
		}
	}
	if b.Files[copyPath].Hash != loser.Hash {
		if err := e.transfer(t, p, copyPath, loser, "", !leftWins, false); err != nil {
			return "", File{}, err
		}
	}
	if err := e.baseline(t, copyPath, loser); err != nil {
		return "", File{}, err
	}
	winner := left
	if leftWins {
		if err := e.transfer(t, p, p, left, right.Hash, true, false); err != nil {
			return "", File{}, err
		}
	} else {
		winner = right
		if err := e.transfer(t, p, p, right, left.Hash, false, true); err != nil {
			return "", File{}, err
		}
	}
	return copyPath, winner, e.baseline(t, p, winner)
}
func (e *Engine) ConfirmDelete(id, p, localHash, remoteHash string) error {
	t, err := e.task(id)
	if err != nil {
		return err
	}
	if t.Coordinator != e.id {
		return e.rpc(t.PeerID, "/delete", map[string]string{"taskId": id, "path": p, "localHash": localHash, "remoteHash": remoteHash}, nil)
	}
	e.mu.RLock()
	lock := e.locks[id]
	e.mu.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	if t.Paused || !t.Started {
		return fmt.Errorf("目录尚未开始或已暂停")
	}
	var b Manifest
	if err = e.rpc(t.PeerID, "/manifest", map[string]string{"taskId": id}, &b); err != nil {
		return err
	}
	root, err := openRoot(t.Root)
	if err != nil {
		return err
	}
	a, err := fileInfo(root, p)
	root.Close()
	if err != nil {
		return err
	}
	if a.Hash != localHash || b.Files[p].Hash != remoteHash {
		return fmt.Errorf("文件已变化，请重新查看删除记录")
	}
	if localHash != "" && remoteHash != "" {
		return fmt.Errorf("文件两端均存在，不能按删除记录执行")
	}
	if err = e.remoteApply(t, p, remoteHash, File{}, bytes.NewReader(nil)); err != nil {
		return err
	}
	if err = e.apply(t, p, localHash, File{}, bytes.NewReader(nil), "确认删除前版本"); err != nil {
		return err
	}
	if err = e.rpc(t.PeerID, "/baseline", map[string]any{"taskId": id, "path": p}, nil); err != nil {
		return err
	}
	if err = e.store.remove("base/" + id + "/" + p); err != nil {
		return err
	}
	return e.RequestSync(id)
}
func (e *Engine) Compare(id, p string) (Comparison, error) {
	t, err := e.task(id)
	if err != nil {
		return Comparison{}, err
	}
	if !strings.EqualFold(path.Ext(p), ".md") {
		return Comparison{}, fmt.Errorf("仅支持 Markdown 差异")
	}
	var issue Issue
	for _, i := range e.existingIssues(id) {
		if i.Path == p && i.Kind == "conflict" {
			issue = i
		}
	}
	if issue.Copy == "" {
		return Comparison{}, fmt.Errorf("冲突记录不存在")
	}
	root, err := openRoot(t.Root)
	if err != nil {
		return Comparison{}, err
	}
	defer root.Close()
	a, err := fileInfo(root, p)
	if err != nil {
		return Comparison{}, err
	}
	b, err := fileInfo(root, issue.Copy)
	if err != nil {
		return Comparison{}, err
	}
	left, err := readContent(t.Root, p, a.Hash, MarkdownLimit)
	if err != nil {
		return Comparison{}, err
	}
	right, err := readContent(t.Root, issue.Copy, b.Hash, MarkdownLimit)
	if err != nil {
		return Comparison{}, err
	}
	if !utf8.Valid(left) || !utf8.Valid(right) {
		return Comparison{}, fmt.Errorf("不是有效的 UTF-8 Markdown")
	}
	return Comparison{Path: p, Local: string(left), Remote: string(right), LocalHash: a.Hash, RemoteHash: b.Hash}, nil
}
func (e *Engine) Resolve(id, p, mainHash, copyHash, content string, keepBoth bool) error {
	t, err := e.task(id)
	if err != nil {
		return err
	}
	if t.Coordinator != e.id {
		return e.rpc(t.PeerID, "/resolve", map[string]any{"taskId": id, "path": p, "localHash": mainHash, "remoteHash": copyHash, "content": content, "keepBoth": keepBoth}, nil)
	}
	e.mu.RLock()
	lock := e.locks[id]
	e.mu.RUnlock()
	lock.Lock()
	defer lock.Unlock()
	if t.Paused || !t.Started {
		return fmt.Errorf("目录尚未开始或已暂停")
	}
	issues := e.existingIssues(id)
	var issue Issue
	for _, i := range issues {
		if i.Path == p && i.Kind == "conflict" {
			issue = i
		}
	}
	if issue.Copy == "" {
		return fmt.Errorf("冲突记录不存在")
	}
	if !keepBoth {
		if !strings.EqualFold(path.Ext(p), ".md") || len(content) > MarkdownLimit {
			return fmt.Errorf("仅支持不超过 2 MiB 的 Markdown 合并")
		}
		a, er := scan(t.Root)
		if er != nil {
			return er
		}
		var b Manifest
		if er = e.rpc(t.PeerID, "/manifest", map[string]string{"taskId": id}, &b); er != nil {
			return er
		}
		if a.Files[p].Hash != mainHash || b.Files[p].Hash != mainHash || a.Files[issue.Copy].Hash != copyHash || b.Files[issue.Copy].Hash != copyHash {
			return fmt.Errorf("版本已变化，请重新打开差异")
		}
		info := File{Hash: fingerprint([]byte(content)), Size: int64(len(content)), Modified: time.Now().UnixNano()}
		if err = e.apply(t, p, mainHash, info, strings.NewReader(content), "手动合并前版本"); err != nil {
			return err
		}
		if err = e.remoteApply(t, p, mainHash, info, strings.NewReader(content)); err != nil {
			return err
		}
		if err = e.baseline(t, p, info); err != nil {
			return err
		}
		if err = e.remoteApply(t, issue.Copy, copyHash, File{}, bytes.NewReader(nil)); err != nil {
			return err
		}
		if err = e.apply(t, issue.Copy, copyHash, File{}, bytes.NewReader(nil), "处理冲突前版本"); err != nil {
			return err
		}
		if err = e.store.remove("base/" + id + "/" + issue.Copy); err != nil {
			return err
		}
		if err = e.rpc(t.PeerID, "/baseline", map[string]string{"taskId": id, "path": issue.Copy}, nil); err != nil {
			return err
		}
	}
	remaining := []Issue{}
	for _, i := range issues {
		if !(i.Path == p && i.Copy == issue.Copy) {
			remaining = append(remaining, i)
		}
	}
	e.status(id, "attention", "已处理", remaining, 0, 0, false)
	e.mu.RLock()
	s := e.statuses[id]
	e.mu.RUnlock()
	if err = e.rpc(t.PeerID, "/status", map[string]any{"taskId": id, "status": s}, nil); err != nil {
		return err
	}
	return e.RequestSync(id)
}
func (e *Engine) Restore(id, versionID string) error {
	t, err := e.task(id)
	if err != nil {
		return err
	}
	var v Version
	ok, err := e.store.get("version/"+id+"/"+versionID, &v)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("历史版本不存在")
	}
	root, err := openRoot(t.Root)
	if err != nil {
		return err
	}
	f, err := fileInfo(root, v.Path)
	root.Close()
	if err != nil {
		return err
	}
	br, err := openRoot(v.BackupDir)
	if err != nil {
		return err
	}
	defer br.Close()
	src, err := br.Open(id + "/" + v.ID)
	if err != nil {
		return err
	}
	defer src.Close()
	info := File{Hash: v.Hash, Size: v.Size, Modified: time.Now().UnixNano()}
	if err = e.apply(t, v.Path, f.Hash, info, src, "恢复前版本"); err != nil {
		return err
	}
	return e.RequestSync(id)
}
func (e *Engine) ExportVersion(id, versionID, destination string) error {
	var v Version
	ok, err := e.store.get("version/"+id+"/"+versionID, &v)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("历史版本不存在")
	}
	destination, err = validateDirectory(destination)
	if err != nil {
		return err
	}
	e.mu.RLock()
	for _, t := range e.tasks {
		if containsPath(t.Root, destination) {
			e.mu.RUnlock()
			return fmt.Errorf("请导出到同步目录之外")
		}
	}
	e.mu.RUnlock()
	br, err := openRoot(v.BackupDir)
	if err != nil {
		return err
	}
	defer br.Close()
	src, err := br.Open(id + "/" + v.ID)
	if err != nil {
		return err
	}
	defer src.Close()
	root, err := openRoot(destination)
	if err != nil {
		return err
	}
	defer root.Close()
	name := strings.TrimSuffix(path.Base(v.Path), path.Ext(v.Path)) + "（恢复-" + v.ID[:8] + "）" + path.Ext(v.Path)
	dst, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	hash, n, err := hashCopy(dst, src)
	if err != nil {
		dst.Close()
		root.Remove(name)
		return err
	}
	if err = syncClose(dst); err != nil {
		return err
	}
	if hash != v.Hash || n != v.Size {
		root.Remove(name)
		return fmt.Errorf("备份校验失败")
	}
	return nil
}
