package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type artifactSet struct {
	discoveryDoc discoveryDocument
	manifest     []ManifestRecord
	unknownItems []UnknownItem
}

type discoveryDocument struct {
	RunID  string          `json:"run_id"`
	Roots  []string        `json:"roots"`
	Items  []DiscoveryItem `json:"items"`
}

func buildArtifactSet(runID string, roots []string, items []DiscoveryItem, unknownItems []UnknownItem) artifactSet {
	return artifactSet{
		discoveryDoc: discoveryDocument{
			RunID: runID,
			Roots: roots,
			Items: items,
		},
		unknownItems: append([]UnknownItem(nil), unknownItems...),
	}
}

func writeArtifacts(outputDir string, artifacts artifactSet) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "discovery.json"), artifacts.discoveryDoc); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "manifest-before.json"), artifacts.manifest); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "unknown-items.json"), artifacts.unknownItems); err != nil {
		return err
	}
	return nil
}

func newArtifactSet(
	runID string,
	roots []string,
	items []DiscoveryItem,
	unknownItems []UnknownItem,
) (artifactSet, error) {
	manifest, err := buildManifest(items)
	if err != nil {
		return artifactSet{}, err
	}
	artifacts := buildArtifactSet(runID, roots, items, unknownItems)
	artifacts.manifest = manifest
	return artifacts, nil
}

func writeJSON(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeBytes(path, data)
}

func writeBytes(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("输出文件已存在: %s", path)
		}
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}
