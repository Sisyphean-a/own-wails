package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type sessionCatalog struct {
	entries  map[string]sessionCatalogEntry
	warnings []ScanWarning
}

type sessionCatalogEntry struct {
	summary       ThreadSummary
	registered    bool
	snapshotBytes int64
	rolloutBytes  int64
	hasLive       bool
	hasArchived   bool
}

type transcriptMetadata struct {
	id               string
	clonedFrom       string
	originalProvider string
	modelProvider    string
	source           string
	cwd              string
	createdAt        string
	updatedAt        string
	path             string
	size             int64
	archived         bool
}

func buildSessionCatalog(paths codexPaths) (sessionCatalog, error) {
	snapshots := indexShellSnapshots(paths.shellSnapshotsDir)
	entries, err := readRegisteredCatalog(paths, snapshots)
	if err != nil {
		return sessionCatalog{}, err
	}
	transcripts, warnings, err := scanTranscriptRoots(paths)
	if err != nil {
		return sessionCatalog{}, err
	}
	for _, transcript := range transcripts {
		entry, exists := entries[transcript.id]
		if !exists && transcript.clonedFrom == "" {
			continue
		}
		if !exists {
			entry = newFileCloneEntry(transcript, sumFileSizes(snapshots[transcript.id]))
		}
		entry = mergeTranscript(entry, transcript)
		entries[transcript.id] = entry
	}
	return sessionCatalog{entries: entries, warnings: warnings}, nil
}

func readRegisteredCatalog(paths codexPaths, snapshots snapshotIndex) (map[string]sessionCatalogEntry, error) {
	index, err := readSessionIndex(paths.sessionIndex)
	if err != nil {
		return nil, err
	}
	db, err := openReadonlyDatabase(paths.stateDB)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	paths.scanMetrics.recordCatalogDatabaseOpen()
	rows, err := db.Query(`select id, title, source, model_provider, thread_source, rollout_path, created_at, updated_at, created_at_ms, updated_at_ms, cwd, archived, first_user_message, preview from threads`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := map[string]sessionCatalogEntry{}
	for rows.Next() {
		row, err := scanThreadRow(rows)
		if err != nil {
			return nil, err
		}
		summary := mapThreadRow(row, index, snapshots[row.ID])
		entries[summary.ID] = newRegisteredEntry(summary, sumFileSizes(snapshots[summary.ID]))
	}
	return entries, rows.Err()
}

func scanTranscriptRoots(paths codexPaths) ([]transcriptMetadata, []ScanWarning, error) {
	all := []transcriptMetadata{}
	warnings := []ScanWarning{}
	for _, root := range []struct {
		path     string
		archived bool
	}{{paths.sessionsDir, false}, {paths.archivedSessionsDir, true}} {
		items, nextWarnings, err := scanTranscriptRoot(root.path, root.archived, paths.scanMetrics)
		if err != nil {
			return nil, nil, err
		}
		all = append(all, items...)
		warnings = append(warnings, nextWarnings...)
	}
	return all, normalizeWarnings(warnings), nil
}

func scanTranscriptRoot(root string, archived bool, metrics *historyScanMetrics) ([]transcriptMetadata, []ScanWarning, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return []transcriptMetadata{}, []ScanWarning{}, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("读取会话目录失败 %s: %w", root, err)
	}
	items := []transcriptMetadata{}
	warnings := []ScanWarning{}
	metrics.recordTranscriptWalk()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "rollout-") || filepath.Ext(entry.Name()) != ".jsonl" {
			return nil
		}
		item, nextWarnings := readTranscriptMetadata(path, archived)
		warnings = append(warnings, nextWarnings...)
		if item.id != "" {
			items = append(items, item)
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("遍历会话目录失败 %s: %w", root, err)
	}
	return items, warnings, nil
}

func readTranscriptMetadata(path string, archived bool) (transcriptMetadata, []ScanWarning) {
	file, err := os.Open(path)
	if err != nil {
		return transcriptMetadata{}, []ScanWarning{newScanWarning(path, "invalid-json", err.Error())}
	}
	defer file.Close()
	info, _ := file.Stat()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return transcriptMetadata{}, []ScanWarning{newScanWarning(path, "read-error", err.Error())}
		}
		return transcriptMetadata{}, []ScanWarning{newScanWarning(path, "invalid-session-meta", "缺少 session_meta")}
	}
	var first map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &first); err != nil {
		return transcriptMetadata{}, []ScanWarning{newScanWarning(path, "invalid-json", err.Error())}
	}
	item, ok := transcriptFromFirstRecord(first, path, archived)
	if !ok {
		return transcriptMetadata{}, []ScanWarning{newScanWarning(path, "invalid-session-meta", "首条记录不是有效 session_meta")}
	}
	if !isSessionID(item.id) {
		return transcriptMetadata{}, []ScanWarning{newScanWarning(path, "invalid-session-id", "会话 ID 不是规范 UUID")}
	}
	if info != nil {
		item.size = info.Size()
	}
	latest, hasTimestamp, warnings := latestTranscriptTimestamp(scanner, path, first)
	if !hasTimestamp {
		if !hasWarningCode(warnings, "read-error") {
			warnings = append(warnings, newScanWarning(path, "missing-timestamp", "没有有效的顶层时间戳"))
		}
	} else {
		item.updatedAt = latest.UTC().Format(time.RFC3339Nano)
	}
	return item, warnings
}

func hasWarningCode(warnings []ScanWarning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

func transcriptFromFirstRecord(record map[string]any, path string, archived bool) (transcriptMetadata, bool) {
	if stringValue(record["type"]) != "session_meta" {
		return transcriptMetadata{}, false
	}
	payload, ok := record["payload"].(map[string]any)
	if !ok {
		return transcriptMetadata{}, false
	}
	id := stringValue(payload["id"])
	if id == "" {
		id = stringValue(payload["session_id"])
	}
	if id == "" {
		return transcriptMetadata{}, false
	}
	createdAt := ""
	if created, ok := parseRecordTimestamp(record); ok {
		createdAt = created.UTC().Format(time.RFC3339Nano)
	}
	return transcriptMetadata{
		id: id, clonedFrom: strings.TrimSpace(stringValue(payload["cloned_from"])),
		originalProvider: stringValue(payload["original_provider"]), modelProvider: stringValue(payload["model_provider"]),
		source: stringValue(payload["source"]), cwd: stringValue(payload["cwd"]),
		createdAt: createdAt, path: path, archived: archived,
	}, true
}

func newRegisteredEntry(summary ThreadSummary, snapshotBytes int64) sessionCatalogEntry {
	return sessionCatalogEntry{
		summary:       summary,
		registered:    true,
		snapshotBytes: snapshotBytes,
		rolloutBytes:  sumFileSizes(summary.RolloutPaths),
		hasLive:       !summary.Archived,
		hasArchived:   summary.Archived,
	}
}

func newFileCloneEntry(item transcriptMetadata, snapshotBytes int64) sessionCatalogEntry {
	title := "克隆会话 " + shortID(item.id)
	return sessionCatalogEntry{
		summary: ThreadSummary{
			ID: item.id, Title: title, SourceTitle: title, Source: "clone-file",
			ModelProvider: item.modelProvider, CWD: item.cwd,
			CreatedAt: item.createdAt, UpdatedAt: item.updatedAt,
		},
		snapshotBytes: snapshotBytes,
	}
}

func mergeTranscript(entry sessionCatalogEntry, item transcriptMetadata) sessionCatalogEntry {
	summary := entry.summary
	rolloutPaths, added := appendUniquePhysicalPath(summary.RolloutPaths, item.path)
	if added {
		entry.rolloutBytes += item.size
	}
	summary.Registered = entry.registered
	summary.RolloutPaths = rolloutPaths
	summary.RolloutPath = rolloutPaths[0]
	summary.SizeBytes = entry.rolloutBytes + entry.snapshotBytes
	summary.CreatedAt = earlierTimestamp(summary.CreatedAt, item.createdAt)
	summary.UpdatedAt = laterTimestamp(summary.UpdatedAt, item.updatedAt)
	if item.clonedFrom != "" {
		summary.IsClone = true
		if summary.ClonedFrom == "" {
			summary.ClonedFrom = item.clonedFrom
			summary.OriginalProvider = item.originalProvider
		}
	}
	entry.hasArchived = entry.hasArchived || item.archived
	entry.hasLive = entry.hasLive || !item.archived
	summary.Archived = entry.hasArchived && !entry.hasLive
	entry.summary = summary
	return entry
}

func (catalog sessionCatalog) resolve(prefix string) (ThreadSummary, error) {
	if prefix == "" {
		return ThreadSummary{}, fmt.Errorf("会话 ID 不能为空")
	}
	matches := []ThreadSummary{}
	for id, entry := range catalog.entries {
		if strings.HasPrefix(id, prefix) {
			matches = append(matches, entry.summary)
		}
	}
	if len(matches) == 0 {
		return ThreadSummary{}, fmt.Errorf("未找到会话: %s", prefix)
	}
	if len(matches) > 1 {
		return ThreadSummary{}, fmt.Errorf("短 ID 命中 %d 条会话，请输入更长的前缀: %s", len(matches), prefix)
	}
	return matches[0], nil
}
