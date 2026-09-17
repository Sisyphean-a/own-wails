package broadcast

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultMinimaxEndpoint = "https://api.minimaxi.com/v1/t2a_v2"
	defaultMinimaxModel    = "speech-2.8-hd"
)

type AudioMetadata struct {
	DurationMillis  int
	AudioSampleRate int
}

type MinimaxClient struct {
	apiKey     string
	model      string
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

func NewMinimaxClient() *MinimaxClient {
	apiKey := strings.TrimSpace(os.Getenv("MINIMAX_API_KEY"))
	model := strings.TrimSpace(os.Getenv("MINIMAX_TTS_MODEL"))
	if model == "" {
		model = defaultMinimaxModel
	}

	return &MinimaxClient{
		apiKey:   apiKey,
		model:    model,
		endpoint: defaultMinimaxEndpoint,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *MinimaxClient) Synthesize(ctx context.Context, req GenerateRequest) ([]byte, AudioMetadata, error) {
	if c.apiKey == "" {
		return nil, AudioMetadata{}, fmt.Errorf("缺少环境变量 MINIMAX_API_KEY")
	}

	payload, err := c.buildPayload(req)
	if err != nil {
		return nil, AudioMetadata{}, err
	}

	httpReq, err := c.newSynthesizeRequest(ctx, payload)
	if err != nil {
		return nil, AudioMetadata{}, err
	}

	body, err := c.executeRequest(httpReq)
	if err != nil {
		return nil, AudioMetadata{}, err
	}

	return parseSynthesizeResponse(body)
}

func (c *MinimaxClient) buildPayload(req GenerateRequest) ([]byte, error) {
	requestBody := minimaxRequest{
		Model:  c.model,
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

func (c *MinimaxClient) newSynthesizeRequest(ctx context.Context, payload []byte) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("创建 TTS 请求失败: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	return httpReq, nil
}

func (c *MinimaxClient) executeRequest(httpReq *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(httpReq)
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
