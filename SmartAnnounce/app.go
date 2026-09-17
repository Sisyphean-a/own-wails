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
}

func NewApp() (*App, error) {
	service, err := broadcast.NewDefaultService()
	if err != nil {
		return nil, err
	}

	return &App{
		broadcastService: service,
	}, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GenerateBroadcast(req broadcast.GenerateRequest) (broadcast.GenerateResult, error) {
	if a.ctx == nil {
		return broadcast.GenerateResult{}, errors.New("应用尚未完成初始化")
	}

	if a.broadcastService == nil {
		return broadcast.GenerateResult{}, errors.New("播报服务尚未完成初始化")
	}

	return a.broadcastService.Generate(a.ctx, req)
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
