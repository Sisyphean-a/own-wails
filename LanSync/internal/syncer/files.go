package syncer

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	gitindex "github.com/go-git/go-git/v5/plumbing/format/index"
)

func validPath(p string) error {
	if p == "" || p == "." || p != path.Clean(p) || strings.ContainsAny(p, "\\:\x00") || strings.HasPrefix(p, "/") {
		return fmt.Errorf("无效相对路径")
	}
	for _, s := range strings.Split(p, "/") {
		if s == ".." || strings.TrimRight(s, " .") != s || strings.EqualFold(s, ".git") || strings.HasPrefix(s, ".lansync-") {
			return fmt.Errorf("受保护路径")
		}
		stem := strings.ToUpper(strings.SplitN(s, ".", 2)[0])
		if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || (len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '1' && stem[3] <= '9') {
			return fmt.Errorf("不支持的跨平台文件名")
		}
	}
	return nil
}
func checkNoLinks(root *os.Root, p string) error {
	if err := validPath(p); err != nil {
		return err
	}
	parts := strings.Split(p, "/")
	for i := range parts {
		f, err := root.Lstat(strings.Join(parts[:i+1], "/"))
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("不处理符号链接：%s", p)
		}
	}
	return nil
}
func hashReader(r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	return hex.EncodeToString(h.Sum(nil)), n, err
}
func fileInfo(root *os.Root, p string) (File, error) {
	if err := checkNoLinks(root, p); err != nil {
		return File{}, err
	}
	f, err := root.Open(p)
	if os.IsNotExist(err) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return File{}, err
	}
	if !before.Mode().IsRegular() {
		return File{}, fmt.Errorf("不是普通文件：%s", p)
	}
	hash, size, err := hashReader(f)
	if err != nil {
		return File{}, err
	}
	after, err := f.Stat()
	if err != nil {
		return File{}, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || size != after.Size() {
		return File{}, fmt.Errorf("文件正在变化：%s", p)
	}
	return File{Hash: hash, Size: size, Modified: after.ModTime().UnixNano()}, nil
}
func readContent(rootPath, p, expected string, limit int64) ([]byte, error) {
	root, err := openRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if err = checkNoLinks(root, p); err != nil {
		return nil, err
	}
	f, err := root.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("Markdown 超过 %d MiB，保留版本而不在线合并", MarkdownLimit>>20)
	}
	if fingerprint(b) != expected {
		return nil, fmt.Errorf("文件已变化，请重试")
	}
	return b, nil
}

type ignoreRules struct {
	enabled  bool
	repo     string
	patterns []gitignore.Pattern
	tracked  map[string]bool
}

func patternsFile(filename string, domain []string) ([]gitignore.Pattern, error) {
	if st, err := os.Lstat(filename); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("忽略规则不能是符号链接：%s", filename)
	}
	f, err := os.Open(filename)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []gitignore.Pattern
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") {
			out = append(out, gitignore.ParsePattern(line, domain))
		}
	}
	return out, s.Err()
}
func loadIgnore(root string) (*ignoreRules, error) {
	r := &ignoreRules{repo: root, tracked: map[string]bool{}}
	gitDir := ""
	for d := root; ; d = filepath.Dir(d) {
		st, err := os.Lstat(filepath.Join(d, ".git"))
		if err == nil {
			r.repo = d
			gitDir = filepath.Join(d, ".git")
			if !st.IsDir() {
				b, e := os.ReadFile(gitDir)
				if e != nil {
					return nil, e
				}
				if !strings.HasPrefix(string(b), "gitdir: ") {
					return nil, fmt.Errorf("无效 Git 工作树")
				}
				gitDir = strings.TrimSpace(strings.TrimPrefix(string(b), "gitdir: "))
				if !filepath.IsAbs(gitDir) {
					gitDir = filepath.Join(d, gitDir)
				}
			}
			break
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	if gitDir == "" {
		return r, nil
	}
	r.enabled = true
	commonDir := gitDir
	if data, er := os.ReadFile(filepath.Join(gitDir, "commondir")); er == nil {
		commonDir = strings.TrimSpace(string(data))
		if !filepath.IsAbs(commonDir) {
			commonDir = filepath.Join(gitDir, commonDir)
		}
	} else if !os.IsNotExist(er) {
		return nil, er
	}
	global, err := gitignore.LoadGlobalPatterns(osfs.New(""))
	if err != nil {
		return nil, wrap("读取 Git 全局忽略规则", err)
	}
	r.patterns = append(r.patterns, global...)
	if gitDir != "" {
		if f, e := os.Open(filepath.Join(gitDir, "index")); e == nil {
			idx := &gitindex.Index{}
			e = gitindex.NewDecoder(f).Decode(idx)
			f.Close()
			if e != nil {
				return nil, wrap("读取 Git 索引", e)
			}
			for _, entry := range idx.Entries {
				r.tracked[entry.Name] = true
			}
		} else if !os.IsNotExist(e) {
			return nil, e
		}
		if f, e := os.Open(filepath.Join(commonDir, "config")); e == nil {
			cfg := config.New()
			e = config.NewDecoder(f).Decode(cfg)
			f.Close()
			if e != nil {
				return nil, e
			}
			if exclude := cfg.Section("core").Option("excludesfile"); exclude != "" {
				if strings.HasPrefix(exclude, "~/") {
					home, e := os.UserHomeDir()
					if e != nil {
						return nil, e
					}
					exclude = filepath.Join(home, exclude[2:])
				}
				if !filepath.IsAbs(exclude) {
					exclude = filepath.Join(r.repo, exclude)
				}
				ps, e := patternsFile(exclude, nil)
				if e != nil {
					return nil, e
				}
				r.patterns = append(r.patterns, ps...)
			}
		} else if !os.IsNotExist(e) {
			return nil, e
		}
		ps, e := patternsFile(filepath.Join(commonDir, "info", "exclude"), nil)
		if e != nil {
			return nil, e
		}
		r.patterns = append(r.patterns, ps...)
	}
	// Rule：选择仓库子目录时，仍继承仓库根到该目录之间的忽略规则。
	for d := r.repo; ; {
		rel, _ := filepath.Rel(r.repo, d)
		var domain []string
		if rel != "." {
			domain = strings.Split(filepath.ToSlash(rel), "/")
		}
		ps, e := patternsFile(filepath.Join(d, ".gitignore"), domain)
		if e != nil {
			return nil, e
		}
		r.patterns = append(r.patterns, ps...)
		if d == root {
			break
		}
		next, _ := filepath.Rel(d, root)
		part := strings.Split(next, string(filepath.Separator))[0]
		d = filepath.Join(d, part)
	}
	return r, nil
}
func (r *ignoreRules) ignored(abs string, isDir bool) bool {
	if !r.enabled {
		return false
	}
	rel, _ := filepath.Rel(r.repo, abs)
	rel = filepath.ToSlash(rel)
	if r.tracked[rel] {
		return false
	}
	if isDir {
		for p := range r.tracked {
			if strings.HasPrefix(p, rel+"/") {
				return false
			}
		}
	}
	return gitignore.NewMatcher(r.patterns).Match(strings.Split(rel, "/"), isDir)
}

func scan(rootPath string) (Manifest, error) {
	m := Manifest{Files: map[string]File{}, Ignored: []string{}, Problems: map[string]string{}}
	root, err := openRoot(rootPath)
	if err != nil {
		return m, err
	}
	defer root.Close()
	rules, err := loadIgnore(rootPath)
	if err != nil {
		return m, err
	}
	cases := map[string]string{}
	var walk func(string) error
	walk = func(dir string) error {
		inherited := rules
		defer func() { rules = inherited }()
		f, e := root.Open(dir)
		if e != nil {
			return e
		}
		entries, e := f.ReadDir(-1)
		f.Close()
		if e != nil {
			return e
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		nested := false
		if dir != "." {
			if _, er := root.Lstat(dir + "/.git"); er == nil {
				rules, e = loadIgnore(filepath.Join(rootPath, filepath.FromSlash(dir)))
				if e != nil {
					return e
				}
				nested = true
			} else if !os.IsNotExist(er) {
				return er
			}
		}
		if dir != "." && rules.enabled && !nested {
			rel, _ := filepath.Rel(rules.repo, filepath.Join(rootPath, filepath.FromSlash(dir)))
			ps, e := patternsFile(filepath.Join(rootPath, filepath.FromSlash(dir), ".gitignore"), strings.Split(filepath.ToSlash(rel), "/"))
			if e != nil {
				return e
			}
			rules.patterns = append(rules.patterns, ps...)
		}
		for _, entry := range entries {
			p := entry.Name()
			if dir != "." {
				p = dir + "/" + p
			}
			if strings.EqualFold(entry.Name(), ".git") || strings.HasPrefix(entry.Name(), ".lansync-") {
				m.Ignored = append(m.Ignored, p+"/")
				continue
			}
			if e = validPath(p); e != nil {
				m.Problems[p] = e.Error()
				continue
			}
			key := strings.ToLower(p)
			if prev, ok := cases[key]; ok && prev != p {
				m.Problems[p] = "文件名大小写冲突"
				m.Problems[prev] = "文件名大小写冲突"
				delete(m.Files, prev)
				continue
			}
			cases[key] = p
			if entry.Type()&os.ModeSymlink != 0 {
				m.Problems[p] = "不处理符号链接"
				continue
			}
			if rules.ignored(filepath.Join(rootPath, filepath.FromSlash(p)), entry.IsDir()) {
				suffix := ""
				if entry.IsDir() {
					suffix = "/"
				}
				m.Ignored = append(m.Ignored, p+suffix)
				continue
			}
			if entry.IsDir() {
				if e = walk(p); e != nil {
					m.Problems[p] = e.Error()
				}
				continue
			}
			fi, e := fileInfo(root, p)
			if e != nil {
				m.Problems[p] = e.Error()
				continue
			}
			m.Files[p] = fi
		}
		return nil
	}
	return m, walk(".")
}
func ignoredPath(rootPath, p string) (bool, error) {
	rules, err := loadIgnore(rootPath)
	if err != nil {
		return false, err
	}
	parts := strings.Split(p, "/")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], "/")
		abs := filepath.Join(rootPath, filepath.FromSlash(parent))
		if rules.ignored(abs, true) {
			return true, nil
		}
		if _, er := os.Lstat(filepath.Join(abs, ".git")); er == nil {
			rules, er = loadIgnore(abs)
			if er != nil {
				return false, er
			}
			continue
		} else if !os.IsNotExist(er) {
			return false, er
		}
		rel, _ := filepath.Rel(rules.repo, abs)
		ps, er := patternsFile(filepath.Join(abs, ".gitignore"), strings.Split(filepath.ToSlash(rel), "/"))
		if er != nil {
			return false, er
		}
		rules.patterns = append(rules.patterns, ps...)
	}
	return rules.ignored(filepath.Join(rootPath, filepath.FromSlash(p)), false), nil
}
func excluded(m Manifest, p string) bool {
	for _, i := range m.Ignored {
		if p == i || strings.HasSuffix(i, "/") && strings.HasPrefix(p, i) {
			return true
		}
	}
	for i := range m.Problems {
		if p == i || strings.HasPrefix(p, i+"/") {
			return true
		}
	}
	return false
}
func containsPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
func validateDirectory(p string) (string, error) {
	p, err := filepath.Abs(strings.TrimSpace(p))
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(real, p) {
		return "", fmt.Errorf("目录路径不能包含符号链接")
	}
	st, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("请选择目录")
	}
	return filepath.Clean(p), nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(p)
	}
}
func syncClose(f *os.File) error {
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
func checkExpected(root *os.Root, p, expected string) error {
	info, err := fileInfo(root, p)
	if err != nil {
		return err
	}
	if info.Hash != expected {
		return fmt.Errorf("文件已变化，未覆盖：%s", p)
	}
	return nil
}
func (e *Engine) backup(root *os.Root, task Task, p string, info File, reason string) error {
	if info.Hash == "" {
		return nil
	}
	e.mu.RLock()
	dir := e.settings.BackupDir
	e.mu.RUnlock()
	if dir == "" {
		return fmt.Errorf("请先选择备份目录")
	}
	id := randomID()
	v := Version{ID: id, TaskID: task.ID, Path: p, Hash: info.Hash, Size: info.Size, Time: now(), Reason: reason, BackupDir: dir}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return wrap("备份失败，未覆盖", err)
	}
	br, err := openRoot(dir)
	if err != nil {
		return err
	}
	defer br.Close()
	target := task.ID + "/" + id
	if err = checkNoLinks(br, target); err != nil {
		return err
	}
	if err = br.MkdirAll(task.ID, 0700); err != nil {
		return err
	}
	src, err := root.Open(p)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := br.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return wrap("备份失败，未覆盖", err)
	}
	hash, n, err := hashCopy(dst, contextReader{e.ctx, src})
	if err != nil {
		dst.Close()
		return err
	}
	if err = syncClose(dst); err != nil {
		return err
	}
	if hash != info.Hash || n != info.Size {
		return fmt.Errorf("备份期间文件变化，未覆盖")
	}
	return e.store.put("version/"+task.ID+"/"+id, v)
}
func hashCopy(w io.Writer, r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(w, h), r)
	return hex.EncodeToString(h.Sum(nil)), n, err
}

// Guarantee：只有校验、备份和写入前比较均成功，才替换原文件。
func (e *Engine) apply(task Task, p, expected string, info File, body io.Reader, reason string) error {
	if err := validPath(p); err != nil {
		return err
	}
	e.filesMu.Lock()
	defer e.filesMu.Unlock()
	root, err := openRoot(task.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	if err = checkNoLinks(root, p); err != nil {
		return err
	}
	ignored, err := ignoredPath(task.Root, p)
	if err != nil {
		return err
	}
	if ignored {
		return fmt.Errorf("文件已被忽略，未写入")
	}
	if err = checkExpected(root, p, expected); err != nil {
		return err
	}
	if info.Hash == "" {
		old, er := fileInfo(root, p)
		if er != nil {
			return er
		}
		if er = e.backup(root, task, p, old, reason); er != nil {
			return er
		}
		if er = checkExpected(root, p, expected); er != nil {
			return er
		}
		if expected == "" {
			return nil
		}
		return root.Remove(p)
	}
	if err = root.MkdirAll(path.Dir(p), 0755); err != nil {
		return err
	}
	temp := path.Join(path.Dir(p), ".lansync-"+randomID())
	f, err := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(temp)
	hash, n, er := hashCopy(f, io.LimitReader(contextReader{e.ctx, body}, info.Size+1))
	if er != nil {
		f.Close()
		return er
	}
	if er = syncClose(f); er != nil {
		return er
	}
	if n != info.Size || hash != info.Hash {
		return fmt.Errorf("接收文件校验失败，未覆盖")
	}
	if er = checkExpected(root, p, expected); er != nil {
		return er
	}
	old, er := fileInfo(root, p)
	if er != nil {
		return er
	}
	if er = e.backup(root, task, p, old, reason); er != nil {
		return er
	}
	if er = root.Chtimes(temp, time.Now(), time.Unix(0, info.Modified)); er != nil {
		return er
	}
	if er = checkExpected(root, p, expected); er != nil {
		return er
	}
	select {
	case <-e.ctx.Done():
		return e.ctx.Err()
	default:
	}
	return root.Rename(temp, p)
}
func (e *Engine) Versions(taskID string) ([]Version, error) {
	values, err := e.store.list("version/" + taskID + "/")
	if err != nil {
		return nil, err
	}
	out := []Version{}
	for _, b := range values {
		var v Version
		if err = decode(b, &v); err != nil {
			return nil, err
		}
		v.BackupDir = ""
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
	return out, nil
}
