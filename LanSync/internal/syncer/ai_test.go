package syncer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type aiTransport func(*http.Request) (*http.Response, error)

func (f aiTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Flow：替换外部 HTTP 边界，仍穿过真实请求序列化和 HTTP 服务，不访问付费 API。
func aiTestClient(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	endpoint, err := url.Parse(server.URL)
	must(t, err)
	client := newAIClient()
	client.Transport = aiTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.deepseek.com/chat/completions" {
			t.Errorf("意外的凭证发送目标：%s", r.URL)
		}
		copy := r.Clone(r.Context())
		copy.URL.Scheme, copy.URL.Host = endpoint.Scheme, endpoint.Host
		return http.DefaultTransport.RoundTrip(copy)
	})
	return client
}

func aiResponse(w http.ResponseWriter, content, finish string) {
	_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
		"message": map[string]string{"content": content}, "finish_reason": finish,
	}}})
}

func TestDeepSeekRequestAndStrategy(t *testing.T) {
	for _, action := range []string{"merge", "current", "copy", "keep_both"} {
		t.Run(action, func(t *testing.T) {
			client := aiTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer sk-test" || r.Header.Get("Content-Type") != "application/json" {
					t.Error("DeepSeek 请求认证或方法错误")
				}
				var payload struct {
					Model          string `json:"model"`
					Stream         bool   `json:"stream"`
					ResponseFormat struct {
						Type string `json:"type"`
					} `json:"response_format"`
					Messages []struct{ Role, Content string } `json:"messages"`
				}
				if json.NewDecoder(r.Body).Decode(&payload) != nil || payload.Model != defaultAIModel || payload.Stream || payload.ResponseFormat.Type != "json_object" || len(payload.Messages) != 2 {
					t.Error("请求格式错误")
				}
				var data map[string]string
				if json.Unmarshal([]byte(payload.Messages[1].Content), &data) != nil || len(data) != 3 || data["current"] != "当前\n" || data["copy"] != "副本\n" || data["instruction"] != "保留独有信息" {
					t.Error("正文或处理要求错误，或发送了额外元数据")
				}
				aiResponse(w, string(encode(map[string]any{"action": action, "summary": "中文建议", "warnings": []string{"请核对"}, "content": "合并稿\n"})), "stop")
			})
			proposal, err := requestAI(context.Background(), client, "sk-test", defaultAIModel, Comparison{Path: "secret-path.md", Local: "当前\n", Remote: "副本\n"}, "保留独有信息")
			must(t, err)
			want := map[string]string{"merge": "合并稿\n", "current": "当前\n", "copy": "副本\n", "keep_both": ""}[action]
			if proposal.Action != action || proposal.Content != want || proposal.Summary == "" {
				t.Fatalf("策略正文错误：%+v", proposal)
			}
		})
	}
}

func TestDeepSeekFailuresNeverBecomeProposals(t *testing.T) {
	for _, tc := range []struct{ name, content, finish string }{
		{"truncated", `{"action":"merge","summary":"OK","content":"partial"}`, "length"},
		{"empty", "", "stop"},
		{"invalid-json", "not JSON", "stop"},
		{"unknown-strategy", `{"action":"delete","summary":"删除"}`, "stop"},
		{"empty-merge", `{"action":"merge","summary":"OK","content":""}`, "stop"},
		{"missing-summary", `{"action":"merge","content":"正文"}`, "stop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := aiTestClient(t, func(w http.ResponseWriter, r *http.Request) { aiResponse(w, tc.content, tc.finish) })
			if _, err := requestAI(context.Background(), client, "sk-secret", defaultAIModel, Comparison{}, ""); err == nil {
				t.Fatal("不完整或非法结果被接受")
			}
		})
	}
	for _, status := range []int{401, 402, 403, 429, 500} {
		client := aiTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, "sk-secret 和用户隐私正文")
		})
		_, err := requestAI(context.Background(), client, "sk-secret", defaultAIModel, Comparison{}, "")
		if err == nil || strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "隐私正文") {
			t.Fatalf("HTTP %d 错误未隔离敏感响应：%v", status, err)
		}
	}
	client := aiTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat(" ", aiResponseLimit+1))
	})
	if _, err := requestAI(context.Background(), client, "sk-secret", defaultAIModel, Comparison{}, ""); err == nil {
		t.Fatal("超限响应被接受")
	}
}

func aiConflict(t *testing.T) (*Engine, *Engine, string, string, string, Comparison) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("密钥持久化使用 Windows DPAPI")
	}
	a, b := paired(t)
	ra, rb := t.TempDir(), t.TempDir()
	write(t, ra, "文档.md", "共同内容\n")
	id := taskPair(t, a, b, ra, rb)
	must(t, a.Sync(id))
	must(t, a.Pause(id, true))
	write(t, ra, "文档.md", "甲的修改\n")
	write(t, rb, "文档.md", "乙的修改\n")
	must(t, a.Pause(id, false))
	must(t, a.Sync(id))
	comparison, err := a.Compare(id, "文档.md")
	must(t, err)
	return a, b, id, ra, rb, comparison
}

func TestAIProposalDoesNotWriteAndRequiresFreshApproval(t *testing.T) {
	a, b, id, ra, rb, c := aiConflict(t)
	must(t, a.SaveAISettings("sk-local-secret", defaultAIModel, false))
	a.aiClient = aiTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		aiResponse(w, `{"action":"merge","summary":"整合两份内容","warnings":["核对措辞"],"content":"甲的修改\n乙的修改\n"}`, "stop")
	})
	before, err := a.Versions(id)
	must(t, err)
	proposal, err := a.AnalyzeConflict(id, c.Path, c.LocalHash, c.RemoteHash, "保留双方内容")
	must(t, err)
	if read(t, ra, c.Path) != c.Local || read(t, rb, c.Path) != c.Local {
		t.Fatal("分析阶段修改了文件")
	}
	after, err := a.Versions(id)
	must(t, err)
	if len(before) != len(after) || proposal.LocalHash != c.LocalHash || proposal.RemoteHash != c.RemoteHash {
		t.Fatal("分析阶段写入备份或未绑定原版本")
	}
	must(t, a.Resolve(id, c.Path, proposal.LocalHash, proposal.RemoteHash, proposal.Content, false))
	if read(t, ra, c.Path) != proposal.Content || read(t, rb, c.Path) != proposal.Content {
		t.Fatal("用户确认后未同步建议稿")
	}
	versions, err := b.Versions(id)
	must(t, err)
	found := false
	for _, v := range versions {
		if v.Path == c.Path && v.Hash == c.LocalHash {
			found = true
		}
	}
	if !found {
		t.Fatal("采用 AI 稿前未备份旧文件")
	}
	if err := a.Resolve(id, c.Path, c.LocalHash, c.RemoteHash, "过期建议", false); err == nil {
		t.Fatal("过期结果被再次应用")
	}
}

func TestAIRefusesStaleInputAndChangesDuringAnalysis(t *testing.T) {
	a, _, id, ra, _, c := aiConflict(t)
	must(t, a.SaveAISettings("sk-local", defaultAIModel, false))
	var called atomic.Bool
	a.aiClient = aiTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		write(t, ra, c.Path, "分析期间的新修改\n")
		aiResponse(w, `{"action":"merge","summary":"整合","content":"旧建议\n"}`, "stop")
	})
	if _, err := a.AnalyzeConflict(id, c.Path, "stale", c.RemoteHash, ""); err == nil || called.Load() {
		t.Fatal("过期输入仍发送到 AI")
	}
	if _, err := a.AnalyzeConflict(id, c.Path, c.LocalHash, c.RemoteHash, ""); err == nil || !called.Load() {
		t.Fatal("分析期间变化没有失效建议")
	}
	if read(t, ra, c.Path) != "分析期间的新修改\n" {
		t.Fatal("分析覆盖了新修改")
	}
}

func TestAIContentLimitAndMissingKeyDoNotSend(t *testing.T) {
	a, b, id, ra, _, c := aiConflict(t)
	var called atomic.Bool
	a.aiClient = aiTestClient(t, func(w http.ResponseWriter, r *http.Request) { called.Store(true) })
	if _, err := a.AnalyzeConflict(id, c.Path, c.LocalHash, c.RemoteHash, ""); err == nil {
		t.Fatal("未配置密钥仍发送请求")
	}
	must(t, a.SaveAISettings("sk-local", defaultAIModel, false))
	peerSettings, err := b.AISettings()
	must(t, err)
	if peerSettings.Configured {
		t.Fatal("密钥被同步到对端")
	}
	write(t, ra, c.Path, strings.Repeat("x", AIContentLimit+1))
	c, err = a.Compare(id, c.Path)
	must(t, err)
	if _, err = a.AnalyzeConflict(id, c.Path, c.LocalHash, c.RemoteHash, ""); err == nil || called.Load() {
		t.Fatal("超限正文被发送或未明确拒绝")
	}
}

func TestAIRedirectDoesNotForwardCredentials(t *testing.T) {
	var leaked atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Store(true) }))
	defer target.Close()
	client := aiTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	})
	if _, err := requestAI(context.Background(), client, "sk-local", defaultAIModel, Comparison{}, ""); err == nil || leaked.Load() {
		t.Fatal("重定向被接受或凭证被转发")
	}
}

func TestAICancelAndSingleFlight(t *testing.T) {
	a, _, id, _, _, c := aiConflict(t)
	must(t, a.SaveAISettings("sk-local", defaultAIModel, false))
	started := make(chan struct{})
	a.aiClient = aiTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
	})
	result := make(chan error, 1)
	go func() { _, err := a.AnalyzeConflict(id, c.Path, c.LocalHash, c.RemoteHash, ""); result <- err }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("AI 请求未开始")
	}
	if _, err := a.AnalyzeConflict(id, c.Path, c.LocalHash, c.RemoteHash, ""); err == nil {
		t.Fatal("并发请求未被阻止")
	}
	a.CancelAI()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("取消被伪报成功")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("取消未终止 HTTP 请求")
	}
}

func TestAIKeyPersistenceIsLocalAndEncrypted(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("密钥持久化使用 Windows DPAPI")
	}
	dir := t.TempDir()
	e, err := New(dir)
	must(t, err)
	const key = "sk-secret-not-for-snapshot"
	must(t, e.SaveAISettings(key, defaultAIModel, false))
	must(t, e.SaveAISettings("", "deepseek-v4-pro", false))
	settings, err := e.AISettings()
	must(t, err)
	if !settings.Configured || settings.Model != "deepseek-v4-pro" {
		t.Fatal("空密钥未保留凭证或模型未保存")
	}
	if strings.Contains(string(encode(e.Snapshot())), key) || strings.Contains(string(encode(settings)), key) {
		t.Fatal("快照泄露密钥")
	}
	must(t, e.Close())
	for _, name := range []string{"state.db", "state.db-wal"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if strings.Contains(string(data), key) {
			t.Fatal("数据库存在明文密钥")
		}
	}
	e, err = New(dir)
	must(t, err)
	t.Cleanup(func() { must(t, e.Close()) })
	settings, err = e.AISettings()
	must(t, err)
	if !settings.Configured {
		t.Fatal("重启丢失密钥")
	}
	must(t, e.SaveAISettings("", defaultAIModel, true))
	settings, err = e.AISettings()
	must(t, err)
	if settings.Configured {
		t.Fatal("删除密钥未生效")
	}
	if err := e.SaveAISettings("bad\nkey", defaultAIModel, false); err == nil {
		t.Fatal("接受包含换行的密钥")
	}
	if err := e.SaveAISettings("sk-key", "unknown", false); err == nil {
		t.Fatal("接受未知模型")
	}
}
