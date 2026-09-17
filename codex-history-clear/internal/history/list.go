package history

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	defaultListLimit   = 80
	unlimitedListLimit = int(^uint(0) >> 1)
)

type threadRow struct {
	ID               string
	Title            string
	Source           string
	ModelProvider    string
	ThreadSource     sql.NullString
	RolloutPath      string
	CreatedAt        int64
	UpdatedAt        int64
	CreatedAtMS      sql.NullInt64
	UpdatedAtMS      sql.NullInt64
	CWD              string
	Archived         int64
	FirstUserMessage sql.NullString
	Preview          sql.NullString
}

func listThreads(paths codexPaths, request ListRequest) ([]ThreadSummary, int, []ScanWarning, error) {
	catalog, err := buildSessionCatalog(paths)
	if err != nil {
		return nil, 0, nil, err
	}
	items := make([]ThreadSummary, 0, len(catalog.entries))
	for _, entry := range catalog.entries {
		if matchesListRequest(entry.summary, request) {
			items = append(items, entry.summary)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt == items[j].UpdatedAt {
			return items[i].ID > items[j].ID
		}
		return timestampAfter(items[i].UpdatedAt, items[j].UpdatedAt)
	})
	total := len(items)
	if limit := effectiveLimit(request.Limit); len(items) > limit {
		items = items[:limit]
	}
	return items, total, catalog.warnings, nil
}

func resolveTargets(paths codexPaths, threadIDs []string) ([]ThreadSummary, error) {
	if len(threadIDs) == 0 {
		return nil, fmt.Errorf("至少选择一个会话")
	}
	catalog, err := buildSessionCatalog(paths)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	targets := make([]ThreadSummary, 0, len(threadIDs))
	for _, threadID := range threadIDs {
		target, err := catalog.resolve(strings.TrimSpace(threadID))
		if err != nil {
			return nil, err
		}
		if _, ok := seen[target.ID]; ok {
			continue
		}
		seen[target.ID] = struct{}{}
		targets = append(targets, target)
	}
	return targets, nil
}

func matchesListRequest(thread ThreadSummary, request ListRequest) bool {
	if !request.All && thread.Archived != request.Archived {
		return false
	}
	if request.All && request.Archived && !thread.Archived {
		return false
	}
	if request.CWD != "" && !strings.Contains(strings.ToLower(thread.CWD), strings.ToLower(request.CWD)) {
		return false
	}
	return matchesGrep(thread, request.Grep)
}

func scanThreadRow(rows *sql.Rows) (threadRow, error) {
	var row threadRow
	err := rows.Scan(
		&row.ID,
		&row.Title,
		&row.Source,
		&row.ModelProvider,
		&row.ThreadSource,
		&row.RolloutPath,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.CreatedAtMS,
		&row.UpdatedAtMS,
		&row.CWD,
		&row.Archived,
		&row.FirstUserMessage,
		&row.Preview,
	)
	return row, err
}

func mapThreadRow(row threadRow, sessionIndex map[string]sessionIndexEntry, snapshots []string) ThreadSummary {
	title := row.Title
	sourceTitle := row.Title
	if entry, ok := sessionIndex[row.ID]; ok && entry.ThreadName != "" {
		title = entry.ThreadName
	}
	return ThreadSummary{
		ID:               row.ID,
		Title:            title,
		SourceTitle:      sourceTitle,
		Source:           row.Source,
		ModelProvider:    row.ModelProvider,
		ThreadSource:     nullableString(row.ThreadSource),
		RolloutPath:      row.RolloutPath,
		RolloutPaths:     nonEmptyStrings(row.RolloutPath),
		Registered:       true,
		CreatedAt:        formatUnix(row.CreatedAtMS, row.CreatedAt),
		UpdatedAt:        formatUnix(row.UpdatedAtMS, row.UpdatedAt),
		CWD:              strings.TrimPrefix(row.CWD, `\\?\`),
		Archived:         row.Archived == 1,
		SizeBytes:        estimateThreadSize(row.RolloutPath, snapshots),
		FirstUserMessage: row.FirstUserMessage.String,
		Preview:          row.Preview.String,
	}
}

func estimateThreadSize(rolloutPath string, snapshots []string) int64 {
	size := fileSize(rolloutPath)
	for _, snapshot := range snapshots {
		size += fileSize(snapshot)
	}
	return size
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func matchesGrep(thread ThreadSummary, grep string) bool {
	if strings.TrimSpace(grep) == "" {
		return true
	}
	needle := strings.ToLower(grep)
	haystack := strings.ToLower(strings.Join([]string{
		thread.Title,
		thread.SourceTitle,
		thread.FirstUserMessage,
		thread.Preview,
	}, "\n"))
	return strings.Contains(haystack, needle)
}

func effectiveLimit(value int) int {
	if value < 0 {
		return unlimitedListLimit
	}
	if value > 0 {
		return value
	}
	return defaultListLimit
}

func formatUnix(ms sql.NullInt64, seconds int64) string {
	if ms.Valid {
		return time.UnixMilli(ms.Int64).UTC().Format(time.RFC3339)
	}
	return time.Unix(seconds, 0).UTC().Format(time.RFC3339)
}

func nullableString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}
