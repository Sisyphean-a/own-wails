package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type sessionIndexEntry struct {
	ThreadName string
	UpdatedAt  string
}

type sessionIndexRow struct {
	ID         string `json:"id"`
	ThreadName string `json:"thread_name"`
	UpdatedAt  string `json:"updated_at"`
}

type historyRow struct {
	SessionID string `json:"session_id"`
}

func readSessionIndex(path string) (map[string]sessionIndexEntry, error) {
	entries := map[string]sessionIndexEntry{}
	if !fileExists(path) {
		return entries, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row sessionIndexRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, err
		}
		if row.ID == "" {
			continue
		}
		entries[row.ID] = sessionIndexEntry{
			ThreadName: row.ThreadName,
			UpdatedAt:  row.UpdatedAt,
		}
	}
	return entries, scanner.Err()
}

func countJSONLMatches(path string, ids map[string]struct{}, decode func([]byte) (string, error)) (int64, error) {
	if !fileExists(path) {
		return 0, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var count int64
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		id, err := decode(line)
		if err != nil {
			return 0, err
		}
		if _, ok := ids[id]; ok {
			count++
		}
	}
	return count, scanner.Err()
}

func rewriteJSONL(path string, ids map[string]struct{}, decode func([]byte) (string, error)) (bool, int64, error) {
	if !fileExists(path) {
		return false, 0, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return false, 0, err
	}
	defer file.Close()

	var removed int64
	var kept [][]byte
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		id, err := decode(line)
		if err != nil {
			return false, 0, err
		}
		if _, ok := ids[id]; ok {
			removed++
			continue
		}
		kept = append(kept, append([]byte(nil), line...))
	}
	if err := scanner.Err(); err != nil {
		return false, 0, err
	}
	if removed == 0 {
		return false, 0, nil
	}
	if err := file.Close(); err != nil {
		return false, 0, err
	}
	output := []byte{}
	for _, line := range kept {
		output = append(output, line...)
		output = append(output, '\n')
	}
	return true, removed, writeAtomic(path, output)
}

func countSessionIndexMatches(path string, ids map[string]struct{}) (int64, error) {
	return countJSONLMatches(path, ids, decodeSessionIndexID)
}

func rewriteSessionIndex(path string, ids map[string]struct{}) (bool, int64, error) {
	return rewriteJSONL(path, ids, decodeSessionIndexID)
}

func countHistoryMatches(path string, ids map[string]struct{}) (int64, error) {
	return countJSONLMatches(path, ids, decodeHistorySessionID)
}

func rewriteHistory(path string, ids map[string]struct{}) (bool, int64, error) {
	return rewriteJSONL(path, ids, decodeHistorySessionID)
}

func decodeSessionIndexID(line []byte) (string, error) {
	var row sessionIndexRow
	if err := json.Unmarshal(line, &row); err != nil {
		return "", err
	}
	return row.ID, nil
}

func decodeHistorySessionID(line []byte) (string, error) {
	var row historyRow
	if err := json.Unmarshal(line, &row); err != nil {
		return "", err
	}
	return row.SessionID, nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, 0o644); err != nil {
		return fmt.Errorf("设置临时文件权限失败: %w", err)
	}
	return os.Rename(tempPath, path)
}
