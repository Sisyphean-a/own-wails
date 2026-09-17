package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type manifestContext struct {
	indexedPathByNormalizedPath map[string]sessionIndexEntry
	indexedPathBySessionUID     map[string]string
}

type sessionIndexEntry struct {
	SessionUID string `json:"id"`
	Path       string `json:"path"`
}

type recordMetadata struct {
	SessionUID    *string
	ThreadUID     *string
	CanonicalPath string
	CwdRaw        *string
	CwdNorm       string
	Evidence      []string
}

type rolloutEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type rolloutSessionMeta struct {
	SessionUID string `json:"id"`
	Cwd        string `json:"cwd"`
	Originator string `json:"originator"`
}

func loadManifestContext(items []DiscoveryItem) (manifestContext, error) {
	context := manifestContext{
		indexedPathByNormalizedPath: map[string]sessionIndexEntry{},
		indexedPathBySessionUID:     map[string]string{},
	}
	for _, item := range items {
		if item.Kind != "session_index_jsonl" {
			continue
		}
		if err := context.loadSessionIndex(item.Path); err != nil {
			return manifestContext{}, err
		}
	}
	return context, nil
}

func (context *manifestContext) loadSessionIndex(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var entry sessionIndexEntry
		err := decoder.Decode(&entry)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("解析 session_index 失败: %s: %w", path, err)
		}
		cleanPath := cleanWindowsPath(entry.Path)
		normalizedPath := normalizeComparablePath(cleanPath)
		if normalizedPath == "" {
			continue
		}
		context.indexedPathByNormalizedPath[normalizedPath] = sessionIndexEntry{
			SessionUID: entry.SessionUID,
			Path:       cleanPath,
		}
		if entry.SessionUID != "" {
			context.indexedPathBySessionUID[entry.SessionUID] = cleanPath
		}
	}
}

func buildRecordMetadata(
	item DiscoveryItem,
	context manifestContext,
	sourcePath string,
	realPath string,
) (recordMetadata, error) {
	metadata := recordMetadata{
		CanonicalPath: canonicalPathFor(sourcePath, realPath),
	}
	if item.Kind != "rollout_jsonl" && item.Kind != "archived_rollout_jsonl" {
		return metadata, nil
	}
	sourcePathKey := normalizeComparablePath(sourcePath)
	if entry, ok := context.indexedPathByNormalizedPath[sourcePathKey]; ok {
		metadata.Evidence = append(metadata.Evidence, "cli-visible")
		if metadata.SessionUID == nil && entry.SessionUID != "" {
			metadata.SessionUID = stringPointer(entry.SessionUID)
		}
	}
	rolloutMetadata, err := readRolloutSessionMeta(item.Path)
	if err != nil {
		metadata.Evidence = append(metadata.Evidence, rolloutMetadataErrorEvidence(err))
		if metadata.SessionUID != nil {
			if indexedPath, ok := context.indexedPathBySessionUID[*metadata.SessionUID]; ok {
				metadata.CanonicalPath = cleanWindowsPath(indexedPath)
			}
		}
		return metadata, nil
	}
	if rolloutMetadata.SessionUID != "" {
		metadata.SessionUID = stringPointer(rolloutMetadata.SessionUID)
	}
	if rolloutMetadata.Cwd != "" {
		metadata.CwdRaw = stringPointer(rolloutMetadata.Cwd)
		metadata.CwdNorm = normalizeProjectPath(rolloutMetadata.Cwd)
	}
	if metadata.SessionUID != nil {
		if indexedPath, ok := context.indexedPathBySessionUID[*metadata.SessionUID]; ok {
			metadata.CanonicalPath = cleanWindowsPath(indexedPath)
		}
	}
	return metadata, nil
}

func rolloutMetadataErrorEvidence(err error) string {
	if errors.Is(err, io.EOF) {
		return "rollout-metadata-missing"
	}
	return "rollout-metadata-invalid"
}

func readRolloutSessionMeta(path string) (rolloutSessionMeta, error) {
	file, err := os.Open(path)
	if err != nil {
		return rolloutSessionMeta{}, err
	}
	defer file.Close()

	var envelope rolloutEnvelope
	if err := json.NewDecoder(file).Decode(&envelope); err != nil {
		return rolloutSessionMeta{}, fmt.Errorf("解析 rollout 元数据失败: %s: %w", path, err)
	}
	if envelope.Type != "session_meta" || len(envelope.Payload) == 0 {
		return rolloutSessionMeta{}, nil
	}
	var metadata rolloutSessionMeta
	if err := json.Unmarshal(envelope.Payload, &metadata); err != nil {
		return rolloutSessionMeta{}, fmt.Errorf("解析 session_meta payload 失败: %s: %w", path, err)
	}
	return metadata, nil
}

func canonicalPathFor(sourcePath string, realPath string) string {
	if realPath != "" {
		return realPath
	}
	return sourcePath
}

func normalizeProjectPath(path string) string {
	return normalizeComparablePath(path)
}

func normalizeComparablePath(path string) string {
	cleaned := cleanWindowsPath(path)
	if cleaned == "" {
		return ""
	}
	return strings.ToLower(cleaned)
}

func cleanWindowsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if converted, ok := convertWSLMountPath(path); ok {
		path = converted
	}
	path = strings.ReplaceAll(path, "/", `\`)
	return filepath.Clean(path)
}

func convertWSLMountPath(path string) (string, bool) {
	if len(path) < 6 || !strings.EqualFold(path[:5], "/mnt/") {
		return "", false
	}
	driveLetter := path[5]
	if !isASCIILetter(driveLetter) {
		return "", false
	}
	if len(path) > 6 && path[6] != '/' {
		return "", false
	}
	remainder := ""
	if len(path) > 7 {
		remainder = strings.ReplaceAll(path[7:], "/", `\`)
	}
	if remainder == "" {
		return fmt.Sprintf("%c:\\", driveLetter), true
	}
	return fmt.Sprintf("%c:\\%s", driveLetter, remainder), true
}

func isASCIILetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func stringPointer(value string) *string {
	copyValue := value
	return &copyValue
}

func mergeEvidence(groups ...[]string) []string {
	merged := []string{}
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, token := range group {
			if token == "" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			merged = append(merged, token)
		}
	}
	return merged
}
