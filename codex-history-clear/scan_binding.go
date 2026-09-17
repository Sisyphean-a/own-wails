package main

import "codex-history-manager/internal/discovery"

type ScanSummary struct {
	RootCount    int `json:"rootCount"`
	ItemCount    int `json:"itemCount"`
	UnknownCount int `json:"unknownCount"`
}

type DiscoveryItem struct {
	SourceRoot string   `json:"sourceRoot"`
	Path       string   `json:"path"`
	Kind       string   `json:"kind"`
	Size       int64    `json:"size"`
	MTimeUTC   string   `json:"mtimeUtc"`
	Attributes []string `json:"attributes"`
	LinkType   *string  `json:"linkType"`
	Target     *string  `json:"target"`
}

type ScanResult struct {
	RunID            string          `json:"runId"`
	Roots            []string        `json:"roots"`
	DiscoveryPath    string          `json:"discoveryPath"`
	ManifestPath     string          `json:"manifestPath"`
	UnknownItemsPath string          `json:"unknownItemsPath"`
	Summary          ScanSummary     `json:"summary"`
	Items            []DiscoveryItem `json:"items"`
}

func (a *App) RunReadOnlyScan() (ScanResult, error) {
	result, err := a.discovery.RunReadOnlyScan()
	if err != nil {
		return ScanResult{}, err
	}

	return mapScanResult(result), nil
}

func mapScanResult(result discovery.ScanResult) ScanResult {
	items := make([]DiscoveryItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, DiscoveryItem{
			SourceRoot: item.SourceRoot,
			Path:       item.Path,
			Kind:       item.Kind,
			Size:       item.Size,
			MTimeUTC:   item.MTimeUTC,
			Attributes: item.Attributes,
			LinkType:   item.LinkType,
			Target:     item.Target,
		})
	}

	return ScanResult{
		RunID:            result.RunID,
		Roots:            append([]string(nil), result.Roots...),
		DiscoveryPath:    result.DiscoveryPath,
		ManifestPath:     result.ManifestPath,
		UnknownItemsPath: result.UnknownItemsPath,
		Summary: ScanSummary{
			RootCount:    result.Summary.RootCount,
			ItemCount:    result.Summary.ItemCount,
			UnknownCount: result.Summary.UnknownCount,
		},
		Items: items,
	}
}
