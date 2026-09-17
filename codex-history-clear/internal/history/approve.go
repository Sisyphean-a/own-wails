package history

import "path/filepath"

func approveDeletePlan(planPath string) (ApproveResult, error) {
	document, err := loadPlanDocument(planPath)
	if err != nil {
		return ApproveResult{}, err
	}
	approvedPlanPath, err := writeApprovedPlan(filepath.Dir(planPath), document)
	if err != nil {
		return ApproveResult{}, err
	}
	return ApproveResult{
		RunID:            document.RunID,
		PlanPath:         planPath,
		ApprovedPlanPath: approvedPlanPath,
		Summary:          document.Summary,
		Targets:          document.Targets,
		Warnings:         document.Warnings,
	}, nil
}
