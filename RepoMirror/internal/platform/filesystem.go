package platform

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type FileSystem interface {
	ListRegularFiles(root string) ([]string, error)
	Exists(path string) (bool, error)
	CompareFile(left string, right string) (FileComparison, error)
	CompareFileFromRoots(leftRoot string, rightRoot string, relPath string) (FileComparison, error)
	FilesEqual(left string, right string) (bool, error)
	FileSize(path string) (int64, error)
	FileSizeFromRoot(root string, relPath string) (int64, error)
	CopyFile(sourcePath string, targetPath string) error
	CopyFileFromRoots(sourceRoot string, targetRoot string, relPath string) error
	EnsureDirectory(path string) error
	Remove(path string) error
	RemoveFromRoot(root string, relPath string) error
	RemoveEmptyParents(root string, start string) error
	RemoveEmptyParentsFromRoot(root string, relPath string) error
}

type FileComparison struct {
	Equal    bool
	LeftSize int64
}

type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (fsys *OSFileSystem) ListRegularFiles(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && strings.EqualFold(entry.Name(), ".git") {
			return filepath.SkipDir
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, err
}

func (fsys *OSFileSystem) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (fsys *OSFileSystem) CompareFile(left string, right string) (FileComparison, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return FileComparison{}, err
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		return FileComparison{}, err
	}
	comparison := FileComparison{LeftSize: leftInfo.Size()}
	comparison.Equal, err = compareFiles(left, right, comparison.LeftSize, rightInfo.Size())
	return comparison, err
}

func (fsys *OSFileSystem) FilesEqual(left string, right string) (bool, error) {
	comparison, err := fsys.CompareFile(left, right)
	if err != nil {
		return false, err
	}
	return comparison.Equal, nil
}

func (fsys *OSFileSystem) FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (fsys *OSFileSystem) FileSizeFromRoot(root string, relPath string) (int64, error) {
	buffer := borrowPathBuffer(len(root) + len(relPath) + 1)
	defer releasePathBuffer(buffer)

	path := buildRootedPath(buffer, root, relPath)
	info, err := os.Stat(bytesToStringView(path))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (fsys *OSFileSystem) CopyFile(sourcePath string, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	if err := fsys.EnsureDirectory(filepath.Dir(targetPath)); err != nil {
		return err
	}
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode().Perm())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	buffer := borrowFileBuffer()
	defer releaseFileBuffer(buffer)

	_, err = io.CopyBuffer(targetFile, sourceFile, buffer)
	return err
}

func (fsys *OSFileSystem) CopyFileFromRoots(sourceRoot string, targetRoot string, relPath string) error {
	sourceBuffer := borrowPathBuffer(len(sourceRoot) + len(relPath) + 1)
	targetBuffer := borrowPathBuffer(len(targetRoot) + len(relPath) + 1)
	defer releasePathBuffer(sourceBuffer)
	defer releasePathBuffer(targetBuffer)

	sourcePath := bytesToStringView(buildRootedPath(sourceBuffer, sourceRoot, relPath))
	targetPath := bytesToStringView(buildRootedPath(targetBuffer, targetRoot, relPath))
	return fsys.CopyFile(sourcePath, targetPath)
}

func (fsys *OSFileSystem) EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (fsys *OSFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (fsys *OSFileSystem) RemoveFromRoot(root string, relPath string) error {
	buffer := borrowPathBuffer(len(root) + len(relPath) + 1)
	defer releasePathBuffer(buffer)

	path := buildRootedPath(buffer, root, relPath)
	return fsys.Remove(bytesToStringView(path))
}

func (fsys *OSFileSystem) CompareFileFromRoots(leftRoot string, rightRoot string, relPath string) (FileComparison, error) {
	leftBuffer := borrowPathBuffer(len(leftRoot) + len(relPath) + 1)
	rightBuffer := borrowPathBuffer(len(rightRoot) + len(relPath) + 1)
	defer releasePathBuffer(leftBuffer)
	defer releasePathBuffer(rightBuffer)

	leftPath := bytesToStringView(buildRootedPath(leftBuffer, leftRoot, relPath))
	rightPath := bytesToStringView(buildRootedPath(rightBuffer, rightRoot, relPath))

	leftInfo, err := os.Stat(leftPath)
	if err != nil {
		return FileComparison{}, err
	}
	rightInfo, err := os.Stat(rightPath)
	if err != nil {
		return FileComparison{}, err
	}
	comparison := FileComparison{LeftSize: leftInfo.Size()}
	comparison.Equal, err = compareFiles(leftPath, rightPath, comparison.LeftSize, rightInfo.Size())
	return comparison, err
}

func (fsys *OSFileSystem) RemoveEmptyParents(root string, start string) error {
	cleanRoot := filepath.Clean(root)
	current := parentDirectory(start)
	for current != cleanRoot && current != "." {
		err := os.Remove(current)
		switch {
		case err == nil:
			current = parentDirectory(current)
		case os.IsNotExist(err):
			current = parentDirectory(current)
		case isDirectoryNotEmptyError(err):
			return nil
		default:
			return err
		}
	}
	return nil
}

func parentDirectory(path string) string {
	end := len(path) - 1
	for end > 0 && os.IsPathSeparator(path[end]) {
		end--
	}
	for end >= 0 {
		if os.IsPathSeparator(path[end]) {
			break
		}
		end--
	}
	if end <= 0 {
		if len(path) != 0 && os.IsPathSeparator(path[0]) {
			return path[:1]
		}
		return "."
	}
	for end > 0 && os.IsPathSeparator(path[end-1]) {
		end--
	}
	if end == 0 {
		return path[:1]
	}
	return path[:end]
}

func (fsys *OSFileSystem) RemoveEmptyParentsFromRoot(root string, relPath string) error {
	buffer := borrowPathBuffer(len(root) + len(relPath) + 1)
	defer releasePathBuffer(buffer)

	start := buildRootedPath(buffer, root, relPath)
	return fsys.RemoveEmptyParents(root, bytesToStringView(start))
}

func isDirectoryNotEmptyError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.Errno(145))
}
