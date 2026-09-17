package appsvc

import "GistSync/internal/syncflow"

type QuickConflictPolicy string

const (
	QuickConflictOverwriteAll QuickConflictPolicy = "overwrite_all"
	QuickConflictManual       QuickConflictPolicy = "manual"
)

type QuickUploadRequest struct {
	ProfileID string `json:"profileId"`
}

type QuickDownloadRequest struct {
	ProfileID        string              `json:"profileId"`
	ConflictPolicy   QuickConflictPolicy `json:"conflictPolicy"`
	OverwriteItemIDs []string            `json:"overwriteItemIds"`
}

type QuickOperationSummary struct {
	Uploaded  int `json:"uploaded"`
	Applied   int `json:"applied"`
	Skipped   int `json:"skipped"`
	Conflicts int `json:"conflicts"`
	Errors    int `json:"errors"`
}

type QuickOperationItem struct {
	ItemID       string              `json:"itemId"`
	TargetPath   string              `json:"targetPath"`
	Status       string              `json:"status"`
	Reason       string              `json:"reason"`
	DiffPreview  string              `json:"diffPreview"`
	DiffStatus   string              `json:"diffStatus"`
	DiffLines    []syncflow.DiffLine `json:"diffLines"`
	AddedLines   int                 `json:"addedLines"`
	RemovedLines int                 `json:"removedLines"`
}

type QuickOperationResult struct {
	OperationID                string                `json:"operationId"`
	Action                     string                `json:"action"`
	ProfileID                  string                `json:"profileId"`
	SnapshotID                 string                `json:"snapshotId"`
	RequiresConflictResolution bool                  `json:"requiresConflictResolution"`
	Summary                    QuickOperationSummary `json:"summary"`
	Conflicts                  []QuickOperationItem  `json:"conflicts"`
	Items                      []QuickOperationItem  `json:"items"`
}
