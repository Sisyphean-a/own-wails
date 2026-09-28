package broadcast

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMinimaxBuildPayloadUsesRequestVolume(t *testing.T) {
	client := &MinimaxClient{}

	payload, err := client.buildPayload(GenerateRequest{
		Text:    "今日特价鸡蛋",
		VoiceID: "female-shaonv",
		Speed:   1.15,
		Volume:  0.65,
	}, defaultMinimaxModel)
	if err != nil {
		t.Fatalf("buildPayload() error = %v", err)
	}

	var request minimaxRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if request.VoiceSetting.Vol != 0.65 {
		t.Fatalf("voice_setting.vol = %v, want %v", request.VoiceSetting.Vol, 0.65)
	}
}

func TestMinimaxClientProxyIsAppControlled(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows DPAPI only")
	}
	store := &SettingsStore{path: filepath.Join(t.TempDir(), "minimax.json")}
	proxyHits := atomic.Int32{}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits.Add(1)
		if r.URL.Host != "tts.example.invalid" {
			t.Errorf("proxy target = %q", r.URL.Host)
		}
		_, _ = w.Write([]byte(`{"base_resp":{"status_code":0},"data":{"audio":"494433"}}`))
	}))
	defer proxy.Close()
	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("HTTPS_PROXY", proxy.URL)
	if _, err := store.Save(SettingsUpdate{APIKey: "saved-key", Model: defaultMinimaxModel, ProxyMode: proxyCustom, ProxyURL: proxy.URL}); err != nil {
		t.Fatal(err)
	}
	client := NewMinimaxClient(store)
	client.endpoint = "http://tts.example.invalid/v1/t2a_v2"
	req := GenerateRequest{Text: "测试", VoiceID: "female-chengshu", Speed: 1, Volume: 1}
	if _, _, err := client.Synthesize(context.Background(), req); err != nil || proxyHits.Load() != 1 {
		t.Fatalf("custom proxy call: err=%v, hits=%d", err, proxyHits.Load())
	}
	originHits := atomic.Int32{}
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originHits.Add(1)
		_, _ = w.Write([]byte(`{"base_resp":{"status_code":0},"data":{"audio":"494433"}}`))
	}))
	defer origin.Close()
	if _, err := store.Save(SettingsUpdate{Model: defaultMinimaxModel, ProxyMode: proxyDirect}); err != nil {
		t.Fatal(err)
	}
	client.endpoint = origin.URL
	if _, _, err := client.Synthesize(context.Background(), req); err != nil || proxyHits.Load() != 1 || originHits.Load() != 1 {
		t.Fatalf("direct call under proxy env: err=%v, proxy hits=%d, origin hits=%d", err, proxyHits.Load(), originHits.Load())
	}
	unavailableProxy := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unavailableURL := unavailableProxy.URL
	unavailableProxy.Close()
	if _, err := store.Save(SettingsUpdate{Model: defaultMinimaxModel, ProxyMode: proxyCustom, ProxyURL: unavailableURL}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Synthesize(context.Background(), req); err == nil || originHits.Load() != 1 {
		t.Fatalf("unavailable proxy must fail without direct fallback: err=%v, origin hits=%d", err, originHits.Load())
	}
}

func TestMinimaxClientUsesSavedSettingsInsteadOfEnvironment(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows DPAPI only")
	}
	store := &SettingsStore{path: filepath.Join(t.TempDir(), "minimax.json")}
	t.Setenv("MINIMAX_API_KEY", "ignored-environment-key")
	t.Setenv("MINIMAX_TTS_MODEL", "ignored-environment-model")
	client := NewMinimaxClient(store)
	_, _, err := client.Synthesize(context.Background(), GenerateRequest{Text: "测试", VoiceID: "female-shaonv", Speed: 1, Volume: 1})
	if err == nil || !strings.Contains(err.Error(), "请先在设置中配置") {
		t.Fatalf("missing saved settings error = %v", err)
	}
	if _, err := store.Save(SettingsUpdate{APIKey: "saved-key", Model: "speech-2.8-turbo"}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer saved-key" {
			t.Errorf("authorization does not use saved key: %q", got)
		}
		var request minimaxRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Model != "speech-2.8-turbo" {
			t.Errorf("model = %q, err = %v", request.Model, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"base_resp":{"status_code":0},"data":{"audio":"494433"}}`))
	}))
	defer server.Close()
	client.endpoint = server.URL
	if _, _, err := client.Synthesize(context.Background(), GenerateRequest{Text: "测试", VoiceID: "female-shaonv", Speed: 1, Volume: 1}); err != nil {
		t.Fatalf("Synthesize() error = %v", err)
	}
	if _, err := os.Stat(store.path); err != nil {
		t.Fatal(err)
	}
}
