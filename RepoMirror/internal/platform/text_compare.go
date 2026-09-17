package platform

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var textComparisonExtensions = map[string]struct{}{
	".cjs":  {},
	".css":  {},
	".env":  {},
	".go":   {},
	".html": {},
	".js":   {},
	".json": {},
	".jsx":  {},
	".md":   {},
	".mjs":  {},
	".scss": {},
	".sh":   {},
	".sql":  {},
	".svg":  {},
	".toml": {},
	".ts":   {},
	".tsx":  {},
	".txt":  {},
	".vue":  {},
	".xml":  {},
	".yaml": {},
	".yml":  {},
}

var textComparisonBaseNames = map[string]struct{}{
	".editorconfig":      {},
	".env.example":       {},
	".env.local":         {},
	".gitattributes":     {},
	".gitignore":         {},
	"Dockerfile":         {},
	"Jenkinsfile":        {},
	"LICENSE":            {},
	"README":             {},
	"README.md":          {},
	"tsconfig.base.json": {},
}

func compareFiles(left string, right string, leftSize int64, rightSize int64) (bool, error) {
	equal, err := compareFileContents(left, right)
	if err != nil || equal {
		return equal, err
	}
	if !shouldIgnoreLineEndingDiff(left) || !shouldIgnoreLineEndingDiff(right) {
		return false, nil
	}
	return compareFileContentsIgnoringLineEndings(left, right, leftSize, rightSize)
}

func compareFileContents(left string, right string) (bool, error) {
	leftFile, err := os.Open(left)
	if err != nil {
		return false, err
	}
	defer leftFile.Close()

	rightFile, err := os.Open(right)
	if err != nil {
		return false, err
	}
	defer rightFile.Close()

	return compareReaders(leftFile, rightFile)
}

func compareReaders(left io.Reader, right io.Reader) (bool, error) {
	leftBuffer := borrowFileBuffer()
	rightBuffer := borrowFileBuffer()
	defer releaseFileBuffer(leftBuffer)
	defer releaseFileBuffer(rightBuffer)

	for {
		leftRead, leftErr := left.Read(leftBuffer)
		rightRead, rightErr := right.Read(rightBuffer)

		if leftRead != rightRead || !bytes.Equal(leftBuffer[:leftRead], rightBuffer[:rightRead]) {
			return false, nil
		}
		if leftErr == io.EOF || rightErr == io.EOF {
			return leftErr == io.EOF && rightErr == io.EOF, nil
		}
		if leftErr != nil {
			return false, leftErr
		}
		if rightErr != nil {
			return false, rightErr
		}
	}
}

func compareFileContentsIgnoringLineEndings(left string, right string, leftSize int64, rightSize int64) (bool, error) {
	if max(leftSize, rightSize) > 8*1024*1024 {
		return false, nil
	}
	leftBytes, err := os.ReadFile(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := os.ReadFile(right)
	if err != nil {
		return false, err
	}
	if bytes.IndexByte(leftBytes, 0) >= 0 || bytes.IndexByte(rightBytes, 0) >= 0 {
		return false, nil
	}
	return bytes.Equal(normalizeLineEndings(leftBytes), normalizeLineEndings(rightBytes)), nil
}

func normalizeLineEndings(content []byte) []byte {
	if bytes.IndexByte(content, '\r') < 0 {
		return content
	}
	normalized := make([]byte, 0, len(content))
	for index := 0; index < len(content); index++ {
		if content[index] != '\r' {
			normalized = append(normalized, content[index])
			continue
		}
		normalized = append(normalized, '\n')
		if index+1 < len(content) && content[index+1] == '\n' {
			index++
		}
	}
	return normalized
}

func shouldIgnoreLineEndingDiff(path string) bool {
	baseName := filepath.Base(path)
	if _, ok := textComparisonBaseNames[baseName]; ok {
		return true
	}
	if strings.HasPrefix(baseName, ".env.") {
		return true
	}
	_, ok := textComparisonExtensions[strings.ToLower(filepath.Ext(baseName))]
	return ok
}
