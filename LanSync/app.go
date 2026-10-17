package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xiakn/lansync/internal/platform"
	"github.com/xiakn/lansync/internal/syncer"
)

type App struct {
	ctx        context.Context
	engine     *syncer.Engine
	dataDir    string
	listen     string
	discover   bool
	startError error
	ready      chan struct{}
	quitting   atomic.Bool
	trayReady  atomic.Bool
}

func NewApp(dataDir, listen string, discover bool) *App {
	return &App{dataDir: dataDir, listen: listen, discover: discover, ready: make(chan struct{})}
}
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	defer close(a.ready)
	// Rule：仅真实启动时访问用户数据，生成 Wails 绑定不能创建身份或数据库。
	a.engine, a.startError = syncer.New(a.dataDir)
	if a.startError != nil {
		return
	}
	a.startError = a.engine.Start(a.listen, a.discover)
	if a.startError != nil {
		return
	}
	platform.StartTray(trayIcon, func() { a.trayReady.Store(true) }, func() { runtime.WindowShow(a.ctx); runtime.WindowUnminimise(a.ctx) }, func() {
		if err := a.SyncAll(); err != nil {
			runtime.EventsEmit(a.ctx, "app:error", err.Error())
		}
	}, func() { a.quitting.Store(true); runtime.Quit(a.ctx) })
}
func (a *App) service() (*syncer.Engine, error) { <-a.ready; return a.engine, a.startError }
func (a *App) secondInstance(options.SecondInstanceData) {
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}
func (a *App) beforeClose(ctx context.Context) bool {
	if !a.quitting.Load() && a.trayReady.Load() {
		runtime.WindowHide(ctx)
		return true
	}
	return false
}
func (a *App) shutdown(context.Context) {
	<-a.ready
	platform.StopTray()
	if a.engine != nil {
		if err := a.engine.Close(); err != nil {
			runtime.LogError(a.ctx, err.Error())
		}
	}
}
func (a *App) GetSnapshot() (syncer.Snapshot, error) {
	e, err := a.service()
	if err != nil {
		return syncer.Snapshot{}, err
	}
	return e.Snapshot(), nil
}
func (a *App) ChooseDirectory(title string) (string, error) {
	if _, err := a.service(); err != nil {
		return "", err
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title, CanCreateDirectories: true})
}
func (a *App) Connect(address string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.Connect(address)
}
func (a *App) ApprovePair(id string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.ApprovePair(id)
}
func (a *App) DismissPair(id string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	e.DismissPair(id)
	return nil
}
func (a *App) Invite(peerID, name, root string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.Invite(peerID, name, root)
}
func (a *App) AcceptInvite(id, root string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.AcceptInvite(id, root)
}
func (a *App) Preview(id string) (syncer.Preview, error) {
	e, err := a.service()
	if err != nil {
		return syncer.Preview{}, err
	}
	return e.Preview(id)
}
func (a *App) Begin(id string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.Begin(id)
}
func (a *App) Sync(id string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.RequestSync(id)
}
func (a *App) SyncAll() error {
	e, err := a.service()
	if err != nil {
		return err
	}
	var first error
	for _, t := range e.Snapshot().Tasks {
		if t.Started && !t.Paused {
			if err := e.RequestSync(t.ID); err != nil && first == nil {
				first = fmt.Errorf("%s：%w", t.Name, err)
			}
		}
	}
	return first
}
func (a *App) Pause(id string, paused bool) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.Pause(id, paused)
}
func (a *App) RemoveTask(id string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.RemoveTask(id)
}
func (a *App) Versions(id string) ([]syncer.Version, error) {
	e, err := a.service()
	if err != nil {
		return nil, err
	}
	return e.Versions(id)
}
func (a *App) Restore(id, versionID string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.Restore(id, versionID)
}
func (a *App) ExportVersion(id, versionID string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	dir, err := a.ChooseDirectory("恢复到文件夹")
	if err != nil || dir == "" {
		return err
	}
	return e.ExportVersion(id, versionID, dir)
}
func (a *App) Compare(id, path string) (syncer.Comparison, error) {
	e, err := a.service()
	if err != nil {
		return syncer.Comparison{}, err
	}
	return e.Compare(id, path)
}
func (a *App) Resolve(id, path, mainHash, copyHash, content string, keepBoth bool) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.Resolve(id, path, mainHash, copyHash, content, keepBoth)
}
func (a *App) ConfirmDelete(id, path, localHash, remoteHash string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	return e.ConfirmDelete(id, path, localHash, remoteHash)
}
func (a *App) SaveSettings(s syncer.Settings) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	old := e.Snapshot().Settings
	if s.AutoStart != old.AutoStart {
		if err = platform.SetAutoStart(s.AutoStart); err != nil {
			return err
		}
	}
	if err = e.SaveSettings(s); err != nil {
		if s.AutoStart != old.AutoStart {
			if rollback := platform.SetAutoStart(old.AutoStart); rollback != nil {
				return fmt.Errorf("%w；开机启动设置恢复失败：%v", err, rollback)
			}
		}
		return err
	}
	return nil
}
func (a *App) AllowLAN() error {
	if _, err := a.service(); err != nil {
		return err
	}
	return platform.AllowLAN()
}
func (a *App) OpenFolder(id string) error {
	e, err := a.service()
	if err != nil {
		return err
	}
	for _, t := range e.Snapshot().Tasks {
		if t.ID == id {
			return platform.OpenFolder(t.Root)
		}
	}
	return fmt.Errorf("同步目录不存在")
}
func (a *App) Quit() { a.quitting.Store(true); runtime.Quit(a.ctx) }
