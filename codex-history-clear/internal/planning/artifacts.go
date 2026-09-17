package planning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"codex-history-manager/internal/discovery"
)

const (
	deletePlanFileName      = "delete-plan.json"
	duplicateGroupsFileName = "duplicate-groups.json"
)

func readManifest(path string) ([]discovery.ManifestRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records []discovery.ManifestRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func writeArtifacts(outputDir string, groups []DuplicateGroup, plan DeletePlanDocument) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, duplicateGroupsFileName), groups); err != nil {
		return err
	}
	return writeJSON(filepath.Join(outputDir, deletePlanFileName), plan)
}

func writeJSON(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", path, err)
	}
	return nil
}
