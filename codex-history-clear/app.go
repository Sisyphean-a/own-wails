package main

import (
	"context"

	"codex-history-manager/internal/discovery"
	"codex-history-manager/internal/history"
	"codex-history-manager/internal/planning"
)

type App struct {
	ctx       context.Context
	discovery *discovery.Service
	history   *history.Service
	planning  *planning.Service
}

func NewApp() *App {
	return &App{
		discovery: discovery.NewService(),
		history:   history.NewService(),
		planning:  planning.NewService(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
