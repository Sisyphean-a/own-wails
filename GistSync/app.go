package main

import (
	"context"
	"fmt"

	"GistSync/internal/appsvc"
	"GistSync/internal/settings"
	"GistSync/internal/syncflow"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
	svc *appsvc.Service
}

func NewApp() *App {
	store, err := settings.NewDefaultStore()
	if err != nil {
		panic(err)
	}
	return &App{svc: appsvc.NewService(store)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) LoadSettingsV2() (settings.Data, error) {
	return a.svc.LoadSettings(a.ctx)
}

func (a *App) SaveSettingsV2(data settings.Data) error {
	return a.svc.SaveSettings(a.ctx, data)
}

func (a *App) PullProfilesFromCloud() (int, error) {
	return a.svc.PullProfilesFromCloud(a.ctx)
}

func (a *App) CreateProfile(name string) (settings.Profile, error) {
	return a.svc.CreateProfile(a.ctx, name)
}

func (a *App) DeleteProfile(profileID string) error {
	return a.svc.DeleteProfile(a.ctx, profileID)
}

func (a *App) SetActiveProfile(profileID string) error {
	return a.svc.SetActiveProfile(a.ctx, profileID)
}

func (a *App) ChooseFilesForProfile(profileID string) ([]string, error) {
	selected, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择要加入配置的文件（可多选）",
	})
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return []string{}, nil
	}
	if err = a.AddFilesToProfile(profileID, selected); err != nil {
		return nil, err
	}
	return selected, nil
}

func (a *App) AddFilesToProfile(profileID string, paths []string) error {
	return a.svc.AddFilesToProfile(a.ctx, profileID, paths)
}

func (a *App) RemoveProfileItems(profileID string, itemIDs []string) error {
	return a.svc.RemoveProfileItems(a.ctx, profileID, itemIDs)
}

func (a *App) SetProfileItemsEnabled(profileID string, itemIDs []string, enabled bool) error {
	return a.svc.SetProfileItemsEnabled(a.ctx, profileID, itemIDs, enabled)
}

func (a *App) UploadProfile(profileID string, selectedItemIDs []string) (syncflow.UploadProfileResult, error) {
	return a.svc.UploadProfile(a.ctx, profileID, selectedItemIDs)
}

func (a *App) ListSnapshots(profileID string) ([]syncflow.SnapshotMeta, error) {
	return a.svc.ListSnapshots(a.ctx, profileID)
}

func (a *App) ValidateMasterPassword(token string, password string) (syncflow.MasterPasswordValidation, error) {
	return a.svc.ValidateMasterPassword(a.ctx, token, password)
}

func (a *App) PreviewApplyConflicts(req syncflow.ApplySnapshotRequest) ([]syncflow.ApplyConflict, error) {
	return a.svc.PreviewApplyConflicts(a.ctx, req)
}

func (a *App) ApplySnapshot(req syncflow.ApplySnapshotRequest) (syncflow.ApplySnapshotResult, error) {
	return a.svc.ApplySnapshot(a.ctx, req)
}

func (a *App) QuickUpload(req appsvc.QuickUploadRequest) (appsvc.QuickOperationResult, error) {
	return a.svc.QuickUpload(a.ctx, req)
}

func (a *App) QuickDownload(req appsvc.QuickDownloadRequest) (appsvc.QuickOperationResult, error) {
	return a.svc.QuickDownload(a.ctx, req)
}

func (a *App) UploadSync() (string, error) {
	return a.svc.UploadSync(a.ctx)
}

func (a *App) DownloadSync(overwrite bool) (string, error) {
	return a.svc.DownloadSync(a.ctx, overwrite)
}

func (a *App) ChooseSyncFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择要同步的文件",
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}
