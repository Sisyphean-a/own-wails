package main

import "codex-history-manager/internal/planning"

type PlanSummary struct {
	GroupCount     int `json:"groupCount"`
	CandidateCount int `json:"candidateCount"`
	ReviewCount    int `json:"reviewCount"`
	PlannedCount   int `json:"plannedCount"`
}

type GroupCandidate struct {
	SessionUID     *string  `json:"sessionUid"`
	ThreadUID      *string  `json:"threadUid"`
	StorageKind    string   `json:"storageKind"`
	SourcePath     string   `json:"sourcePath"`
	CanonicalPath  string   `json:"canonicalPath"`
	RealPath       string   `json:"realPath"`
	UpdatedAt      string   `json:"updatedAt"`
	Preferred      bool     `json:"preferred"`
	Relation       string   `json:"relation"`
	Action         string   `json:"action"`
	ReasonCode     string   `json:"reasonCode"`
	Reason         string   `json:"reason"`
	RequiresCLI    bool     `json:"requiresCli"`
	ReviewNeeded   bool     `json:"reviewNeeded"`
	QuarantinePath *string  `json:"quarantinePath"`
	Warnings       []string `json:"warnings"`
}

type DuplicateGroup struct {
	DuplicateGroup string           `json:"duplicateGroup"`
	PreferredPath  string           `json:"preferredPath"`
	ReviewNeeded   bool             `json:"reviewNeeded"`
	Warning        string           `json:"warning"`
	Candidates     []GroupCandidate `json:"candidates"`
}

type DeletePlanItem struct {
	DuplicateGroup string   `json:"duplicateGroup"`
	SessionUID     *string  `json:"sessionUid"`
	SourcePath     string   `json:"sourcePath"`
	PreferredPath  string   `json:"preferredPath"`
	Action         string   `json:"action"`
	ReasonCode     string   `json:"reasonCode"`
	Reason         string   `json:"reason"`
	RequiresCLI    bool     `json:"requiresCli"`
	ReviewNeeded   bool     `json:"reviewNeeded"`
	QuarantinePath *string  `json:"quarantinePath"`
	Warnings       []string `json:"warnings"`
}

type DeletePlanResult struct {
	RunID               string           `json:"runId"`
	ManifestPath        string           `json:"manifestPath"`
	DuplicateGroupsPath string           `json:"duplicateGroupsPath"`
	DeletePlanPath      string           `json:"deletePlanPath"`
	Summary             PlanSummary      `json:"summary"`
	Groups              []DuplicateGroup `json:"groups"`
	Items               []DeletePlanItem `json:"items"`
	Warnings            []string         `json:"warnings"`
}

func (a *App) BuildDeletePlan(manifestPath string) (DeletePlanResult, error) {
	result, err := a.planning.BuildDeletePlan(manifestPath)
	if err != nil {
		return DeletePlanResult{}, err
	}
	return mapDeletePlanResult(result), nil
}

func mapDeletePlanResult(result planning.Result) DeletePlanResult {
	groups := make([]DuplicateGroup, 0, len(result.Groups))
	for _, group := range result.Groups {
		groups = append(groups, mapDuplicateGroup(group))
	}
	items := make([]DeletePlanItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapDeletePlanItem(item))
	}
	return DeletePlanResult{
		RunID:               result.RunID,
		ManifestPath:        result.ManifestPath,
		DuplicateGroupsPath: result.DuplicateGroupsPath,
		DeletePlanPath:      result.DeletePlanPath,
		Summary: PlanSummary{
			GroupCount:     result.Summary.GroupCount,
			CandidateCount: result.Summary.CandidateCount,
			ReviewCount:    result.Summary.ReviewCount,
			PlannedCount:   result.Summary.PlannedCount,
		},
		Groups:   groups,
		Items:    items,
		Warnings: cloneStrings(result.Warnings),
	}
}

func mapDuplicateGroup(group planning.DuplicateGroup) DuplicateGroup {
	candidates := make([]GroupCandidate, 0, len(group.Candidates))
	for _, candidate := range group.Candidates {
		candidates = append(candidates, GroupCandidate{
			SessionUID:     candidate.SessionUID,
			ThreadUID:      candidate.ThreadUID,
			StorageKind:    candidate.StorageKind,
			SourcePath:     candidate.SourcePath,
			CanonicalPath:  candidate.CanonicalPath,
			RealPath:       candidate.RealPath,
			UpdatedAt:      candidate.UpdatedAt,
			Preferred:      candidate.Preferred,
			Relation:       candidate.Relation,
			Action:         candidate.Action,
			ReasonCode:     candidate.ReasonCode,
			Reason:         candidate.Reason,
			RequiresCLI:    candidate.RequiresCLI,
			ReviewNeeded:   candidate.ReviewNeeded,
			QuarantinePath: candidate.QuarantinePath,
			Warnings:       cloneStrings(candidate.Warnings),
		})
	}
	return DuplicateGroup{
		DuplicateGroup: group.DuplicateGroup,
		PreferredPath:  group.PreferredPath,
		ReviewNeeded:   group.ReviewNeeded,
		Warning:        group.Warning,
		Candidates:     candidates,
	}
}

func mapDeletePlanItem(item planning.DeletePlanItem) DeletePlanItem {
	return DeletePlanItem{
		DuplicateGroup: item.DuplicateGroup,
		SessionUID:     item.SessionUID,
		SourcePath:     item.SourcePath,
		PreferredPath:  item.PreferredPath,
		Action:         item.Action,
		ReasonCode:     item.ReasonCode,
		Reason:         item.Reason,
		RequiresCLI:    item.RequiresCLI,
		ReviewNeeded:   item.ReviewNeeded,
		QuarantinePath: item.QuarantinePath,
		Warnings:       cloneStrings(item.Warnings),
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}
