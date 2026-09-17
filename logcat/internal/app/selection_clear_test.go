package app

import (
	"context"
	"sync"
	"testing"

	"github.com/xiakn/logcat/internal/adb"
	"github.com/xiakn/logcat/internal/session"
)

type persistentSessionStarter struct {
	mu       sync.Mutex
	contexts []context.Context
	configs  []session.Config
	events   []chan session.Event
}

func (s *persistentSessionStarter) Start(
	ctx context.Context,
	cfg session.Config,
) (session.Handle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := make(chan session.Event)
	s.contexts = append(s.contexts, ctx)
	s.configs = append(s.configs, cfg)
	s.events = append(s.events, events)
	return session.NewHandle(events), nil
}

func (s *persistentSessionStarter) latestEvents() chan<- session.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events[len(s.events)-1]
}

func (s *persistentSessionStarter) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, events := range s.events {
		close(events)
	}
}

func TestControllerClearPackagePreservesLogsAndRunningState(t *testing.T) {
	starter := &persistentSessionStarter{}
	t.Cleanup(starter.close)
	controller := newPackageSelectionController(t, starter)
	controller.ResumeKeep()

	if err := controller.SelectPackage(context.Background(), "com.demo.host"); err != nil {
		t.Fatalf("SelectPackage returned error: %v", err)
	}
	controller.pushEntry(*makeEntry("bridge ready"))
	controller.pushEntry(*makeEntry("plain log"))
	controller.SetFilterDraft("bridge")
	if err := controller.ApplyFilterDraft(); err != nil {
		t.Fatalf("ApplyFilterDraft returned error: %v", err)
	}

	if err := controller.SelectPackage(context.Background(), ""); err != nil {
		t.Fatalf("clearing package returned error: %v", err)
	}

	model := controller.Model()
	if model.TotalLogs != 2 || len(model.VisibleLogs) != 1 {
		t.Fatalf("expected preserved filtered logs, total=%d visible=%d", model.TotalLogs, len(model.VisibleLogs))
	}
	if model.Pause.Active {
		t.Fatal("expected package clear to keep streaming active")
	}
	if !controller.UISnapshot(10).SessionActive {
		t.Fatal("expected device-level session to remain active")
	}

	controller.SetFilterDraft("")
	if err := controller.ApplyFilterDraft(); err != nil {
		t.Fatalf("clearing filter returned error: %v", err)
	}
	if visible := len(controller.Model().VisibleLogs); visible != 2 {
		t.Fatalf("expected both historical logs after filter clear, got %d", visible)
	}
}

func TestControllerSavedFilterStartsBeforeTargetApp(t *testing.T) {
	starter := &persistentSessionStarter{}
	t.Cleanup(starter.close)
	controller := newPackageSelectionController(t, starter)
	controller.ReplaceSavedFilters([]SavedFilter{{
		ID:          "startup",
		Name:        "Startup",
		PackageName: "com.demo.stopped",
		Query:       "startup",
	}}, "")

	if err := controller.ApplySavedFilter(context.Background(), "startup"); err != nil {
		t.Fatalf("ApplySavedFilter returned error: %v", err)
	}
	controller.ResumeKeep()

	starter.mu.Lock()
	config := starter.configs[len(starter.configs)-1]
	starter.mu.Unlock()
	if config.PackageName != "" || len(config.AllowedPIDs) != 0 {
		t.Fatalf("expected device-level startup session, got %#v", config)
	}
	starter.latestEvents() <- session.Event{Entry: makeEntry("startup critical")}
	waitFor(t, func() bool {
		return len(controller.Model().VisibleLogs) == 1
	})
	if !controller.UISnapshot(10).SessionActive {
		t.Fatal("expected startup session to remain active")
	}
}

func TestControllerClearPackagePreservesPausedBuffer(t *testing.T) {
	starter := &persistentSessionStarter{}
	t.Cleanup(starter.close)
	controller := newPackageSelectionController(t, starter)
	controller.ResumeKeep()
	if err := controller.SelectPackage(context.Background(), "com.demo.host"); err != nil {
		t.Fatalf("SelectPackage returned error: %v", err)
	}
	controller.pushEntry(*makeEntry("visible before pause"))
	controller.Pause()
	controller.pushEntry(*makeEntry("buffered during pause"))

	if err := controller.SelectPackage(context.Background(), ""); err != nil {
		t.Fatalf("clearing package returned error: %v", err)
	}
	model := controller.Model()
	if !model.Pause.Active || model.Pause.BufferedCount != 1 {
		t.Fatalf("expected paused buffer preserved, got %#v", model.Pause)
	}
	if model.Status != "Paused，缓存 1 条新日志" || model.TotalLogs != 1 {
		t.Fatalf("unexpected paused state: status=%q total=%d", model.Status, model.TotalLogs)
	}

	controller.ResumeKeep()
	model = controller.Model()
	if model.Pause.Active || model.TotalLogs != 2 || len(model.VisibleLogs) != 2 {
		t.Fatalf("expected buffered log restored on resume, got pause=%#v total=%d visible=%d", model.Pause, model.TotalLogs, len(model.VisibleLogs))
	}
}

func newPackageSelectionController(
	t *testing.T,
	starter SessionStarter,
) *Controller {
	t.Helper()
	controller := NewController(stubDeviceService{
		install: adb.Install{Path: "adb", Version: "1.0.41"},
		devices: []adb.DeviceInfo{{ID: "device-1", Model: "Pixel_7", Status: "device"}},
		packagesByScope: map[adb.PackageScope][]adb.PackageInfo{
			adb.PackageScopeAll: {{Name: "com.demo.host"}},
		},
		processesByPackage: map[string][]adb.ProcessInfo{
			"com.demo.host": {{PID: 111, Name: "com.demo.host"}},
		},
	}, starter)
	if err := controller.Load(context.Background()); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if err := controller.SelectDevice(context.Background(), "device-1"); err != nil {
		t.Fatalf("SelectDevice returned error: %v", err)
	}
	return controller
}
