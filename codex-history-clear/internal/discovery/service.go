package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Service struct {
	now               func() time.Time
	userHomeDir       func() (string, error)
	codexHomeOverride string
}

func NewService() *Service {
	return &Service{
		now:               time.Now,
		userHomeDir:       os.UserHomeDir,
		codexHomeOverride: "",
	}
}

func (s *Service) SetCodexHomeOverride(root string) {
	s.codexHomeOverride = root
}

func (s *Service) RunReadOnlyScan() (ScanResult, error) {
	runID := buildRunID(s.now().UTC())
	roots, err := s.resolveRoots()
	if err != nil {
		return ScanResult{}, err
	}
	outputDir, err := resolveOutputDir(runID)
	if err != nil {
		return ScanResult{}, err
	}
	items, unknownItems, err := s.collectItems(roots)
	if err != nil {
		return ScanResult{}, err
	}
	artifacts, err := newArtifactSet(runID, roots, items, unknownItems)
	if err != nil {
		return ScanResult{}, err
	}
	if err := writeArtifacts(outputDir, artifacts); err != nil {
		return ScanResult{}, err
	}

	return ScanResult{
		RunID:            runID,
		Roots:            append([]string(nil), roots...),
		DiscoveryPath:    filepath.Join(outputDir, "discovery.json"),
		ManifestPath:     filepath.Join(outputDir, "manifest-before.json"),
		UnknownItemsPath: filepath.Join(outputDir, "unknown-items.json"),
		Summary:          scanSummary(roots, items, len(artifacts.unknownItems)),
		Items:            items,
	}, nil
}

func resolveOutputDir(runID string) (string, error) {
	return filepath.Abs(filepath.Clean(filepath.Join(os.TempDir(), "codex-history-manager", "runs", runID)))
}

func scanSummary(roots []string, items []DiscoveryItem, unknownCount int) ScanSummary {
	return ScanSummary{
		RootCount:    len(roots),
		ItemCount:    len(items),
		UnknownCount: unknownCount,
	}
}

func buildRunID(now time.Time) string {
	return fmt.Sprintf("%s-%09d", now.Format("20060102-150405"), now.Nanosecond())
}
