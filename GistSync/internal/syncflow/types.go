package syncflow

import (
	"context"
	"errors"

	"GistSync/internal/settings"
)

const (
	manifestFileName = "sync_manifest.json"
	manifestVersion  = 2
	restoreOriginal  = "original"
	restoreRooted    = "rooted"
)

var (
	ErrProfileNotFound  = errors.New("profile not found")
	ErrSnapshotNotFound = errors.New("snapshot not found")
	ErrEmptyPassword    = errors.New("master password is required")
	ErrEmptyRestoreRoot = errors.New("restore root is required when mode is rooted")
)

type CloudGateway interface {
	EnsureManifestGist(ctx context.Context) (string, error)
	FindManifestGist(ctx context.Context) (string, error)
	UpsertFile(ctx context.Context, req UpsertFileRequest) error
	GetFileContent(ctx context.Context, req FileRequest) (string, error)
}

type UpsertFileRequest struct {
	GistID   string
	FileName string
	Content  string
}

type FileRequest struct {
	GistID   string
	FileName string
}

type UploadProfileRequest struct {
	Profile         settings.Profile
	MasterPassword  string
	SelectedItemIDs []string
}

type UploadProfileResult struct {
	SnapshotID string `json:"snapshotId"`
	Uploaded   int    `json:"uploaded"`
}

type SnapshotMeta struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
}

type MasterPasswordValidation struct {
	Valid       bool `json:"valid"`
	HasSnapshot bool `json:"hasSnapshot"`
}

type ApplyConflict struct {
	ItemID       string     `json:"itemId"`
	TargetPath   string     `json:"targetPath"`
	DiffPreview  string     `json:"diffPreview"`
	DiffStatus   string     `json:"diffStatus"`
	DiffLines    []DiffLine `json:"diffLines"`
	AddedLines   int        `json:"addedLines"`
	RemovedLines int        `json:"removedLines"`
}

// DiffLine 是导出的结构化 diff 行，供前端按语义着色渲染。
type DiffLine struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type ApplySnapshotRequest struct {
	ProfileID        string   `json:"profileId"`
	SnapshotID       string   `json:"snapshotId"`
	MasterPassword   string   `json:"masterPassword"`
	RestoreMode      string   `json:"restoreMode"`
	RestoreRoot      string   `json:"restoreRoot"`
	SelectedItemIDs  []string `json:"selectedItemIds"`
	OverwriteItemIDs []string `json:"overwriteItemIds"`
}

type ApplyItemResult struct {
	ItemID     string `json:"itemId"`
	TargetPath string `json:"targetPath"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
}

type ApplySnapshotResult struct {
	Applied int               `json:"applied"`
	Skipped int               `json:"skipped"`
	Items   []ApplyItemResult `json:"items"`
}

type Service struct {
	cloud    CloudGateway
	observer MetricsObserver
}

type manifest struct {
	Version   int                `json:"version"`
	Profiles  []manifestProfile  `json:"profiles"`
	Snapshots []manifestSnapshot `json:"snapshots"`
}

type manifestProfile struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	RestoreMode string                `json:"restoreMode"`
	RestoreRoot string                `json:"restoreRoot"`
	Items       []manifestProfileItem `json:"items"`
}

type manifestProfileItem struct {
	ID                 string `json:"id"`
	SourcePathTemplate string `json:"sourcePathTemplate"`
	RelativePath       string `json:"relativePath"`
	Enabled            bool   `json:"enabled"`
}

type manifestSnapshot struct {
	ID        string                 `json:"id"`
	ProfileID string                 `json:"profileId"`
	CreatedAt string                 `json:"createdAt"`
	Items     []manifestSnapshotItem `json:"items"`
}

type manifestSnapshotItem struct {
	ItemID             string `json:"itemId"`
	SourcePathTemplate string `json:"sourcePathTemplate"`
	RelativePath       string `json:"relativePath"`
	BlobFile           string `json:"blobFile"`
}

func NewService(cloud CloudGateway) *Service {
	return &Service{
		cloud:    cloud,
		observer: newMetricsObserver(),
	}
}
