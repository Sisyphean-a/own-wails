package planning

import (
	"fmt"
	"path/filepath"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) BuildDeletePlan(manifestPath string) (Result, error) {
	absManifestPath, err := filepath.Abs(filepath.Clean(manifestPath))
	if err != nil {
		return Result{}, fmt.Errorf("解析 manifest 路径失败: %w", err)
	}
	records, err := readManifest(absManifestPath)
	if err != nil {
		return Result{}, fmt.Errorf("读取 manifest 失败: %w", err)
	}
	runID := filepath.Base(filepath.Dir(absManifestPath))
	groups := buildDuplicateGroups(records, runID)
	document := buildDeletePlanDocument(runID, groups)
	outputDir := filepath.Dir(absManifestPath)
	if err := writeArtifacts(outputDir, groups, document); err != nil {
		return Result{}, err
	}
	return Result{
		RunID:               runID,
		ManifestPath:        absManifestPath,
		DuplicateGroupsPath: filepath.Join(outputDir, duplicateGroupsFileName),
		DeletePlanPath:      filepath.Join(outputDir, deletePlanFileName),
		Summary:             buildSummary(groups),
		Groups:              groups,
		Items:               document.Items,
		Warnings:            document.Warnings,
	}, nil
}

func buildDeletePlanDocument(runID string, groups []DuplicateGroup) DeletePlanDocument {
	items := make([]DeletePlanItem, 0, countCandidates(groups))
	warnings := make([]string, 0, len(groups))
	for _, group := range groups {
		if group.Warning != "" {
			warnings = append(warnings, group.Warning)
		}
		for _, candidate := range group.Candidates {
			items = append(items, DeletePlanItem{
				DuplicateGroup: group.DuplicateGroup,
				SessionUID:     candidate.SessionUID,
				SourcePath:     candidate.SourcePath,
				PreferredPath:  group.PreferredPath,
				Action:         candidate.Action,
				ReasonCode:     candidate.ReasonCode,
				Reason:         candidate.Reason,
				RequiresCLI:    candidate.RequiresCLI,
				ReviewNeeded:   candidate.ReviewNeeded,
				QuarantinePath: candidate.QuarantinePath,
				Warnings:       cloneStrings(candidate.Warnings),
			})
		}
	}
	return DeletePlanDocument{
		RunID:    runID,
		Approved: false,
		Items:    items,
		Warnings: warnings,
	}
}

func buildSummary(groups []DuplicateGroup) Summary {
	summary := Summary{
		GroupCount:     len(groups),
		CandidateCount: countCandidates(groups),
		PlannedCount:   countPlannedItems(groups),
	}
	for _, group := range groups {
		if group.ReviewNeeded {
			summary.ReviewCount++
		}
	}
	return summary
}

func countCandidates(groups []DuplicateGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.Candidates)
	}
	return total
}

func countPlannedItems(groups []DuplicateGroup) int {
	total := 0
	for _, group := range groups {
		for _, candidate := range group.Candidates {
			if candidate.Action != "" {
				total++
			}
		}
	}
	return total
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}
