package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func latestTranscriptTimestamp(scanner *bufio.Scanner, path string, first map[string]any) (time.Time, bool, []ScanWarning) {
	latest, found := parseRecordTimestamp(first)
	warnings := []ScanWarning{}
	for scanner.Scan() {
		recordTime, ok, parseErr := timestampFromLine(scanner.Bytes())
		if parseErr != nil {
			warnings = append(warnings, newScanWarning(path, "invalid-json", parseErr.Error()))
			continue
		}
		if ok && (!found || recordTime.After(latest)) {
			latest, found = recordTime, true
		}
	}
	if err := scanner.Err(); err != nil {
		warnings = append(warnings, newScanWarning(path, "read-error", err.Error()))
		return time.Time{}, false, warnings
	}
	return latest, found, warnings
}

func timestampFromLine(line []byte) (time.Time, bool, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return time.Time{}, false, nil
	}
	var record map[string]json.RawMessage
	if err := json.Unmarshal(line, &record); err != nil {
		return time.Time{}, false, err
	}
	value, exists := record["timestamp"]
	if !exists {
		return time.Time{}, false, nil
	}
	var timestamp string
	if err := json.Unmarshal(value, &timestamp); err != nil {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(timestamp))
	return parsed, err == nil, nil
}

func parseRecordTimestamp(record map[string]any) (time.Time, bool) {
	value := stringValue(record["timestamp"])
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, err == nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func newScanWarning(path string, code string, message string) ScanWarning {
	return ScanWarning{Path: path, Code: code, Message: message}
}

func normalizeWarnings(warnings []ScanWarning) []ScanWarning {
	seen := map[string]ScanWarning{}
	for _, warning := range warnings {
		seen[warning.Path+"\x00"+warning.Code] = warning
	}
	result := make([]ScanWarning, 0, len(seen))
	for _, warning := range seen {
		result = append(result, warning)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path == result[j].Path {
			return result[i].Code < result[j].Code
		}
		return result[i].Path < result[j].Path
	})
	return result
}

func nonEmptyStrings(values ...string) []string {
	result := []string{}
	for _, value := range values {
		if value != "" {
			result = appendUnique(result, value)
		}
	}
	return result
}

func appendUnique(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniquePhysicalPath(values []string, value string) ([]string, bool) {
	realValue, valueErr := resolvePhysicalPath(value)
	for _, current := range values {
		if current == value {
			return values, false
		}
		realCurrent, currentErr := resolvePhysicalPath(current)
		if valueErr == nil && currentErr == nil && sameFilesystemPath(realCurrent, realValue) {
			return values, false
		}
	}
	return append(values, value), true
}

func isSessionID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

func sumFileSizes(paths []string) int64 {
	var total int64
	for _, path := range paths {
		total += fileSize(path)
	}
	return total
}

func earlierTimestamp(left string, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	leftTime, leftOK := parseTimestamp(left)
	rightTime, rightOK := parseTimestamp(right)
	if leftOK && rightOK && rightTime.Before(leftTime) {
		return right
	}
	return left
}

func laterTimestamp(left string, right string) string {
	if left == "" || timestampAfter(right, left) {
		return right
	}
	return left
}

func timestampAfter(left string, right string) bool {
	leftTime, leftOK := parseTimestamp(left)
	rightTime, rightOK := parseTimestamp(right)
	if leftOK && rightOK {
		return leftTime.After(rightTime)
	}
	return left > right
}

func parseTimestamp(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, err == nil
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
