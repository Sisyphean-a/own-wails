package history

import (
	"encoding/json"
	"os"
)

func countGlobalStateMatches(path string, ids map[string]struct{}) (int64, error) {
	if !fileExists(path) {
		return 0, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return 0, err
	}
	_, count := pruneThreadReferences(value, ids)
	return count, nil
}

func rewriteGlobalState(path string, ids map[string]struct{}) (bool, int64, error) {
	if !fileExists(path) {
		return false, 0, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, 0, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false, 0, err
	}
	next, count := pruneThreadReferences(value, ids)
	if count == 0 {
		return false, 0, nil
	}
	output, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return false, 0, err
	}
	output = append(output, '\n')
	return true, count, writeAtomic(path, output)
}

func pruneThreadReferences(value any, ids map[string]struct{}) (any, int64) {
	switch typed := value.(type) {
	case []any:
		next := make([]any, 0, len(typed))
		var removed int64
		for _, item := range typed {
			if text, ok := item.(string); ok {
				if _, hit := ids[text]; hit {
					removed++
					continue
				}
			}
			child, count := pruneThreadReferences(item, ids)
			removed += count
			next = append(next, child)
		}
		return next, removed
	case map[string]any:
		next := make(map[string]any, len(typed))
		var removed int64
		for key, item := range typed {
			if containsTrackedID(key, ids) {
				removed++
				continue
			}
			child, count := pruneThreadReferences(item, ids)
			removed += count
			next[key] = child
		}
		return next, removed
	default:
		return value, 0
	}
}

func containsTrackedID(value string, ids map[string]struct{}) bool {
	for id := range ids {
		if id != "" && len(value) >= len(id) && containsSubstring(value, id) {
			return true
		}
	}
	return false
}

func containsSubstring(value string, needle string) bool {
	return len(needle) > 0 && len(value) >= len(needle) && stringContains(value, needle)
}

func stringContains(value string, needle string) bool {
	return len(needle) > 0 && len(value) >= len(needle) && (value == needle || len(value) > len(needle) && indexOf(value, needle) >= 0)
}

func indexOf(value string, needle string) int {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
