package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codex-history-manager/internal/discovery"
)

func loadPlanDocument(path string) (planDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return planDocument{}, err
	}
	var document planDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return planDocument{}, err
	}
	if len(document.Targets) == 0 {
		return planDocument{}, fmt.Errorf("删除计划为空: %s", path)
	}
	return document, nil
}

func writeAfterManifest(outputDir string, newDiscovery func() *discovery.Service) error {
	afterScan, err := newDiscovery().RunReadOnlyScan()
	if err != nil {
		return err
	}
	if err := copyFile(afterScan.ManifestPath, filepath.Join(outputDir, "manifest-after.json")); err != nil {
		return err
	}
	if err := copyFile(afterScan.DiscoveryPath, filepath.Join(outputDir, "discovery-after.json")); err != nil {
		return err
	}
	return copyFile(afterScan.UnknownItemsPath, filepath.Join(outputDir, "unknown-items-after.json"))
}

func emit(events *[]JobEvent, phase string, index int, total int, level string, message string, artifact string) {
	*events = append(*events, JobEvent{
		Phase:        phase,
		ItemIndex:    index,
		ItemTotal:    total,
		Level:        level,
		Message:      message,
		ArtifactPath: artifact,
	})
}

func assertTargetNotActive(target ThreadSummary) error {
	currentThreadID := strings.TrimSpace(os.Getenv("CODEX_THREAD_ID"))
	if currentThreadID != "" && currentThreadID == target.ID {
		return fmt.Errorf("拒绝删除当前活动线程: %s", target.ID)
	}
	return nil
}

func writeJSON(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
