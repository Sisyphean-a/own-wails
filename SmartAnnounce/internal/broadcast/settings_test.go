package broadcast

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSettingsStoreRoundTripAndKeyNotExposed(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows DPAPI only")
	}
	store := &SettingsStore{path: filepath.Join(t.TempDir(), "minimax.json")}
	initial, err := store.Info()
	if err != nil || initial.Model != defaultMinimaxModel || initial.HasAPIKey || initial.ProxyMode != proxyDirect {
		t.Fatalf("initial settings: %+v, %v", initial, err)
	}
	if _, err := store.Save(SettingsUpdate{Model: "speech-2.8-hd"}); err == nil {
		t.Fatal("saving without an API Key must fail")
	}
	secret := "sensitive-test-secret-123"
	info, err := store.Save(SettingsUpdate{APIKey: secret, Model: "speech-2.8-hd"})
	if err != nil || !info.HasAPIKey || strings.Contains(info.Model, secret) {
		t.Fatalf("save: %+v, %v", info, err)
	}
	onDisk, err := os.ReadFile(store.path)
	if err != nil || strings.Contains(string(onDisk), secret) {
		t.Fatalf("secret stored as plaintext or read failed: %v", err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.APIKey != secret {
		t.Fatalf("load failed: %v, key matches: %v", err, loaded.APIKey == secret)
	}
	if _, err := store.Save(SettingsUpdate{Model: "speech-2.8-turbo"}); err != nil {
		t.Fatalf("save model preserving key: %v", err)
	}
	loaded, err = store.Load()
	if err != nil || loaded.APIKey != secret || loaded.Model != "speech-2.8-turbo" {
		t.Fatalf("updated config: model=%q, key matches=%v, err=%v", loaded.Model, loaded.APIKey == secret, err)
	}
	if _, err := store.Save(SettingsUpdate{APIKey: "new-secret", Model: "speech-2.8-hd"}); err != nil {
		t.Fatalf("replace key: %v", err)
	}
	loaded, err = store.Load()
	if err != nil || loaded.APIKey != "new-secret" {
		t.Fatalf("replaced key: matches=%v, err=%v", loaded.APIKey == "new-secret", err)
	}
	info, err = store.Save(SettingsUpdate{Model: defaultMinimaxModel, ProxyMode: proxyCustom, ProxyURL: "http://127.0.0.1:7890"})
	if err != nil || info.ProxyMode != proxyCustom || info.ProxyURL != "http://127.0.0.1:7890" {
		t.Fatalf("custom proxy: %+v, %v", info, err)
	}
	if _, err := store.Save(SettingsUpdate{Model: defaultMinimaxModel, ProxyMode: proxyCustom, ProxyURL: "http://user:pass@127.0.0.1:7890"}); err == nil {
		t.Fatal("proxy credentials must not be saved in plaintext")
	}
	loaded, err = store.Load()
	if err != nil || loaded.ProxyURL != "http://127.0.0.1:7890" {
		t.Fatalf("invalid proxy update replaced previous settings: %v", err)
	}
	info, err = store.Save(SettingsUpdate{Model: defaultMinimaxModel, ProxyMode: proxyDirect})
	if err != nil || info.ProxyURL != "" || info.ProxyMode != proxyDirect {
		t.Fatalf("return to direct: %+v, %v", info, err)
	}
}

func TestSettingsStoreLoadsPreProxyConfigAsDirect(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows DPAPI only")
	}
	store := &SettingsStore{path: filepath.Join(t.TempDir(), "minimax.json")}
	if _, err := store.Save(SettingsUpdate{APIKey: "prior-key", Model: defaultMinimaxModel}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	var previous map[string]any
	if err := json.Unmarshal(data, &previous); err != nil {
		t.Fatal(err)
	}
	delete(previous, "proxyMode")
	data, _ = json.Marshal(previous)
	if err := os.WriteFile(store.path, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.APIKey != "prior-key" || loaded.ProxyMode != proxyDirect {
		t.Fatalf("legacy config: mode=%q, key matches=%v, err=%v", loaded.ProxyMode, loaded.APIKey == "prior-key", err)
	}
}

func TestSettingsStoreRejectsCorruptFileWithoutReset(t *testing.T) {
	store := &SettingsStore{path: filepath.Join(t.TempDir(), "minimax.json")}
	if err := os.WriteFile(store.path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Info(); err == nil {
		t.Fatal("corrupt settings must be reported")
	}
	if _, err := store.Save(SettingsUpdate{APIKey: "key", Model: defaultMinimaxModel}); err == nil {
		t.Fatal("saving over corrupt settings must not discard the old file")
	}
	data, err := os.ReadFile(store.path)
	if err != nil || string(data) != "broken" {
		t.Fatalf("corrupt settings overwritten: %v", err)
	}
}
