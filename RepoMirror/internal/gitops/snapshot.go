package gitops

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"RepoMirror/internal/model"
)

type Snapshot struct {
	Root  string
	Files []string
	Scope []string
}

func (snapshot Snapshot) Close() error {
	return os.RemoveAll(snapshot.Root)
}

type snapshotBlob struct {
	path string
	hash string
}

func (s *Service) ExportRevision(root string, selection model.SourceSelection) (snapshot Snapshot, resultErr error) {
	hash, err := s.ResolveRevision(root, selection.Commit)
	if err != nil {
		return snapshot, err
	}
	if selection.Mode == "commit" {
		snapshot.Scope, err = s.changedPaths(root, hash)
		if err != nil {
			return snapshot, err
		}
		sort.Strings(snapshot.Scope)
	} else if selection.Mode != "branch" {
		return snapshot, fmt.Errorf("无效的版本来源: %s", selection.Mode)
	}
	output, err := s.runner.Run(root, nil, "ls-tree", "-r", "-z", "--full-tree", hash)
	if err != nil {
		return snapshot, fmt.Errorf("无法读取版本文件: %w", err)
	}
	blobs := make([]snapshotBlob, 0)
	snapshot.Files = make([]string, 0)
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, path, ok := strings.Cut(string(record), "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 3 {
			return snapshot, fmt.Errorf("Git 文件树格式无效")
		}
		if isProtectedPath(path) {
			continue
		}
		if snapshot.Scope != nil {
			if _, exists := slices.BinarySearch(snapshot.Scope, path); !exists {
				continue
			}
		}
		if err := validateSnapshotPath(path); err != nil {
			return snapshot, err
		}
		if fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
			return snapshot, fmt.Errorf("版本中包含不支持的符号链接或子模块: %s", path)
		}
		blobs = append(blobs, snapshotBlob{path: path, hash: fields[2]})
		snapshot.Files = append(snapshot.Files, path)
	}
	sort.Strings(snapshot.Files)
	snapshot.Root, err = os.MkdirTemp("", "repomirror-source-")
	if err != nil {
		return snapshot, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, snapshot.Close())
		}
	}()
	if len(blobs) == 0 {
		return snapshot, nil
	}
	var input bytes.Buffer
	for _, blob := range blobs {
		input.WriteString(blob.hash)
		input.WriteByte('\n')
	}
	reader, writer := io.Pipe()
	finished := make(chan error, 1)
	// Flow: 单个 Git 进程流式导出，避免按文件启动进程或缓存整个版本内容。
	go func() {
		err := s.runner.RunTo(root, input.Bytes(), writer, "cat-file", "--batch")
		writer.CloseWithError(err)
		finished <- err
	}()
	err = extractBlobs(bufio.NewReader(reader), snapshot.Root, blobs)
	if err == nil {
		_, err = io.Copy(io.Discard, reader)
	}
	reader.CloseWithError(err)
	runErr := <-finished
	if err != nil || runErr != nil {
		return snapshot, fmt.Errorf("无法导出版本文件: %w", errors.Join(err, runErr))
	}
	return snapshot, nil
}

func extractBlobs(reader *bufio.Reader, root string, blobs []snapshotBlob) error {
	for _, blob := range blobs {
		header, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[0] != blob.hash || fields[1] != "blob" {
			return fmt.Errorf("Git 文件内容格式无效: %s", blob.path)
		}
		size, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || size < 0 {
			return fmt.Errorf("Git 文件大小无效: %s", blob.path)
		}
		path := filepath.Join(root, filepath.FromSlash(blob.path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(file, reader, size)
		if err := errors.Join(copyErr, file.Close()); err != nil {
			return err
		}
		if separator, err := reader.ReadByte(); err != nil || separator != '\n' {
			return fmt.Errorf("Git 文件内容不完整: %s", blob.path)
		}
	}
	return nil
}

func validateSnapshotPath(path string) error {
	if !filepath.IsLocal(path) || strings.ContainsAny(path, "\\:") || filepath.ToSlash(filepath.Clean(path)) != path {
		return fmt.Errorf("版本中包含非法路径: %q", path)
	}
	for _, part := range strings.Split(path, "/") {
		if strings.TrimRight(part, ". ") != part {
			return fmt.Errorf("版本中包含 Windows 不支持的路径: %q", path)
		}
	}
	return nil
}
