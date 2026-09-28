package broadcast

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultMinimaxEndpoint = "https://api.minimaxi.com/v1/t2a_v2"

type AudioMetadata struct {
	DurationMillis  int
	AudioSampleRate int
}

type MinimaxClient struct {
	settings   *SettingsStore
	endpoint   string
	httpClient *http.Client
}

type minimaxRequest struct {
	Model          string              `json:"model"`
	Text           string              `json:"text"`
	Stream         bool                `json:"stream"`
	VoiceSetting   minimaxVoiceSetting `json:"voice_setting"`
	AudioSetting   minimaxAudioSetting `json:"audio_setting"`
	SubtitleEnable bool                `json:"subtitle_enable"`
	LanguageBoost  string              `json:"language_boost"`
}

type minimaxVoiceSetting struct {
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
	Vol     float64 `json:"vol"`
	Pitch   int     `json:"pitch"`
}

type minimaxAudioSetting struct {
	SampleRate int    `json:"sample_rate"`
	Bitrate    int    `json:"bitrate"`
	Format     string `json:"format"`
	Channel    int    `json:"channel"`
}

type minimaxResponse struct {
	BaseResp minimaxBaseResp `json:"base_resp"`
	Data     minimaxData     `json:"data"`
}

type minimaxBaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type minimaxData struct {
	Audio     string            `json:"audio"`
	ExtraInfo minimaxExtraInfo  `json:"extra_info"`
	TraceID   string            `json:"trace_id"`
	Subtitle  []json.RawMessage `json:"subtitle_file_list"`
}

type minimaxExtraInfo struct {
	AudioLength     int `json:"audio_length"`
	AudioSampleRate int `json:"audio_sample_rate"`
	Duration        int `json:"duration"`
}

func NewMinimaxClient(settings *SettingsStore) *MinimaxClient {
	return &MinimaxClient{
		settings: settings,
		endpoint: defaultMinimaxEndpoint,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *MinimaxClient) Synthesize(ctx context.Context, req GenerateRequest) ([]byte, AudioMetadata, error) {
	if c.settings == nil {
		return nil, AudioMetadata{}, fmt.Errorf("Minimax 配置存储尚未初始化")
	}
	settings, err := c.settings.Load()
	if err != nil {
		return nil, AudioMetadata{}, err
	}
	if settings.APIKey == "" {
		return nil, AudioMetadata{}, fmt.Errorf("请先在设置中配置 Minimax API Key")
	}

	payload, err := c.buildPayload(req, settings.Model)
	if err != nil {
		return nil, AudioMetadata{}, err
	}

	httpReq, err := c.newSynthesizeRequest(ctx, payload, settings.APIKey)
	if err != nil {
		return nil, AudioMetadata{}, err
	}

	body, err := c.executeRequest(httpReq, settings)
	if err != nil {
		return nil, AudioMetadata{}, err
	}

	return parseSynthesizeResponse(body)
}

func (c *MinimaxClient) buildPayload(req GenerateRequest, model string) ([]byte, error) {
	requestBody := minimaxRequest{
		Model:  model,
		Text:   req.NormalizedText(),
		Stream: false,
		VoiceSetting: minimaxVoiceSetting{
			VoiceID: strings.TrimSpace(req.VoiceID),
			Speed:   req.Speed,
			Vol:     req.Volume,
			Pitch:   0,
		},
		AudioSetting: minimaxAudioSetting{
			SampleRate: 32000,
			Bitrate:    128000,
			Format:     "mp3",
			Channel:    1,
		},
		SubtitleEnable: false,
		LanguageBoost:  "auto",
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化 TTS 请求失败: %w", err)
	}

	return payload, nil
}

func (c *MinimaxClient) newSynthesizeRequest(ctx context.Context, payload []byte, apiKey string) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("创建 TTS 请求失败: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	return httpReq, nil
}

func (c *MinimaxClient) executeRequest(httpReq *http.Request, settings minimaxSettings) ([]byte, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if settings.ProxyMode == proxyCustom {
		proxyURL, err := url.Parse(settings.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("解析代理地址失败: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	defer transport.CloseIdleConnections()
	client := *c.httpClient
	client.Transport = transport
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用 Minimax TTS 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Minimax 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Minimax TTS 返回非成功状态 %s: %s", resp.Status, string(body))
	}

	return body, nil
}

func parseSynthesizeResponse(body []byte) ([]byte, AudioMetadata, error) {
	var parsed minimaxResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, AudioMetadata{}, fmt.Errorf("解析 Minimax 响应失败: %w", err)
	}

	if parsed.BaseResp.StatusCode != 0 {
		return nil, AudioMetadata{}, fmt.Errorf("Minimax TTS 业务失败: %s", parsed.BaseResp.StatusMsg)
	}

	audioHex := strings.TrimSpace(parsed.Data.Audio)
	if audioHex == "" {
		return nil, AudioMetadata{}, fmt.Errorf("Minimax 未返回音频数据")
	}

	audioBytes, err := hex.DecodeString(audioHex)
	if err != nil {
		return nil, AudioMetadata{}, fmt.Errorf("解码 Minimax 音频失败: %w", err)
	}

	return audioBytes, AudioMetadata{
		DurationMillis:  parsed.Data.ExtraInfo.Duration,
		AudioSampleRate: parsed.Data.ExtraInfo.AudioSampleRate,
	}, nil
}
