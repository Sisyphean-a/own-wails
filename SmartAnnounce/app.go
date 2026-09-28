package main

import (
	"context"
	"errors"
	"path/filepath"

	"SmartAnnounce/internal/broadcast"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx              context.Context
	broadcastService *broadcast.Service
	settings         *broadcast.SettingsStore
	history          *broadcast.TextHistoryStore
}

func NewApp() (*App, error) {
	settings, err := broadcast.NewSettingsStore()
	if err != nil {
		return nil, err
	}
	service, err := broadcast.NewDefaultService(settings)
	if err != nil {
		return nil, err
	}
	history, err := broadcast.NewTextHistoryStore()
	if err != nil {
		return nil, err
	}

	return &App{
		broadcastService: service,
		settings:         settings,
		history:          history,
	}, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetMinimaxSettings() (broadcast.SettingsInfo, error) {
	if a.settings == nil {
		return broadcast.SettingsInfo{}, errors.New("配置服务尚未完成初始化")
	}
	return a.settings.Info()
}

func (a *App) SaveMinimaxSettings(req broadcast.SettingsUpdate) (broadcast.SettingsInfo, error) {
	if a.settings == nil {
		return broadcast.SettingsInfo{}, errors.New("配置服务尚未完成初始化")
	}
	return a.settings.Save(req)
}

func (a *App) GenerateBroadcast(req broadcast.GenerateRequest) (broadcast.GenerateResult, error) {
	if a.ctx == nil {
		return broadcast.GenerateResult{}, errors.New("应用尚未完成初始化")
	}

	if a.broadcastService == nil {
		return broadcast.GenerateResult{}, errors.New("播报服务尚未完成初始化")
	}

	if err := req.Validate(); err != nil {
		return broadcast.GenerateResult{}, err
	}
	if a.history == nil {
		return broadcast.GenerateResult{}, errors.New("文案历史尚未完成初始化")
	}
	// Rule: 先持久化当次提交的文本，音频生成失败也不影响历史记录。
	if _, err := a.history.Record(req.NormalizedText()); err != nil {
		return broadcast.GenerateResult{}, err
	}
	return a.broadcastService.Generate(a.ctx, req)
}

func (a *App) ListTextHistory() ([]broadcast.TextHistoryEntry, error) {
	if a.history == nil {
		return nil, errors.New("文案历史尚未完成初始化")
	}
	return a.history.List()
}

func (a *App) DeleteTextHistory(id string) error {
	if a.history == nil {
		return errors.New("文案历史尚未完成初始化")
	}
	return a.history.Delete(id)
}

func (a *App) ExportBroadcast(req broadcast.ExportRequest) (broadcast.ExportResult, error) {
	if a.ctx == nil {
		return broadcast.ExportResult{}, errors.New("应用尚未完成初始化")
	}

	if a.broadcastService == nil {
		return broadcast.ExportResult{}, errors.New("播报服务尚未完成初始化")
	}

	if err := req.Validate(); err != nil {
		return broadcast.ExportResult{}, err
	}

	targetPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "导出音频文件",
		DefaultDirectory:     filepath.Dir(req.SourcePath),
		DefaultFilename:      req.DefaultFileName(),
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "MP3 音频",
				Pattern:     "*.mp3",
			},
		},
	})
	if err != nil {
		return broadcast.ExportResult{}, err
	}

	if targetPath == "" {
		return broadcast.ExportResult{Cancelled: true}, nil
	}

	return a.broadcastService.Export(req, targetPath)
}
