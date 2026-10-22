package gitops

import (
	"bytes"
	"strings"
)

const verboseIgnoreFieldCount = 4

// Rule: check-ignore 统一带 -z，输出路径保持原样，避免 core.quotepath 对非 ASCII 路径加引号和八进制转义。
func parseIgnoredPaths(output []byte) map[string]string {
	ignored := make(map[string]string, estimatedNullFieldCount(output)/verboseIgnoreFieldCount)
	ruleLabels := make(map[string]string, 4)
	for {
		var fields [verboseIgnoreFieldCount][]byte
		rest, ok := cutNullFields(output, &fields)
		if !ok {
			return ignored
		}
		output = rest
		relPath := normalizeSlashPathRaw(fields[3])
		if relPath == "" {
			continue
		}
		ignored[relPath] = cachedIgnoredRuleLabel(fields[2], ruleLabels)
	}
}

func cutNullFields(record []byte, fields *[verboseIgnoreFieldCount][]byte) ([]byte, bool) {
	rest := record
	for index := range fields {
		end := bytes.IndexByte(rest, 0)
		if end < 0 {
			return nil, false
		}
		fields[index] = rest[:end]
		rest = rest[end+1:]
	}
	return rest, true
}

func parseIgnoredPathSet(output []byte) map[string]struct{} {
	ignored := make(map[string]struct{}, estimatedNullFieldCount(output))
	for len(output) > 0 {
		end := bytes.IndexByte(output, 0)
		if end < 0 {
			appendIgnoredPath(ignored, output)
			break
		}
		appendIgnoredPath(ignored, output[:end])
		output = output[end+1:]
	}
	return ignored
}

func appendIgnoredPath(ignored map[string]struct{}, rawPath []byte) {
	relPath := normalizeSlashPathRaw(rawPath)
	if relPath != "" {
		ignored[relPath] = struct{}{}
	}
}

func estimatedNullFieldCount(output []byte) int {
	if len(output) == 0 {
		return 0
	}
	return bytes.Count(output, []byte{0}) + 1
}

func cachedIgnoredRuleLabel(pattern []byte, ruleLabels map[string]string) string {
	key := strings.TrimSpace(bytesToStringView(pattern))
	if rule, ok := ruleLabels[key]; ok {
		return rule
	}
	rule := ignoredRuleLabel(key)
	ruleLabels[key] = rule
	return rule
}

func ignoredRuleLabel(pattern string) string {
	lower := strings.ToLower(strings.TrimSpace(pattern))
	switch {
	case strings.Contains(lower, ".env"):
		return "env-protected"
	case strings.Contains(lower, ".yaml"), strings.Contains(lower, ".yml"), strings.Contains(lower, "config"):
		return "cfg-protected"
	case strings.Contains(lower, "secret"), strings.Contains(lower, "key"):
		return "secret-protected"
	default:
		return "ignore-protected"
	}
}
