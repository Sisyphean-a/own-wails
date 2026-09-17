package broadcast

import (
	"errors"
	"strings"
)

const maxTextLength = 2000

type GenerateRequest struct {
	Text    string  `json:"text"`
	VoiceID string  `json:"voiceId"`
	Speed   float64 `json:"speed"`
	Volume  float64 `json:"volume"`
}

type GenerateResult struct {
	FileName        string `json:"fileName"`
	FilePath        string `json:"filePath"`
	AudioDataURI    string `json:"audioDataUri"`
	VoiceID         string `json:"voiceId"`
	CharacterCount  int    `json:"characterCount"`
	DurationMillis  int    `json:"durationMillis"`
	AudioSampleRate int    `json:"audioSampleRate"`
}

type ExportRequest struct {
	SourcePath string `json:"sourcePath"`
	FileName   string `json:"fileName"`
}

type ExportResult struct {
	FilePath  string `json:"filePath"`
	Cancelled bool   `json:"cancelled"`
}

func (r GenerateRequest) NormalizedText() string {
	return strings.TrimSpace(r.Text)
}

func (r GenerateRequest) Validate() error {
	text := r.NormalizedText()
	if text == "" {
		return errors.New("播报文本不能为空")
	}

	if len([]rune(text)) > maxTextLength {
		return errors.New("播报文本长度不能超过 2000 个字符")
	}

	if strings.TrimSpace(r.VoiceID) == "" {
		return errors.New("必须选择一个音色")
	}

	if r.Speed < 0.5 || r.Speed > 2 {
		return errors.New("语速必须在 0.5 到 2.0 之间")
	}

	if r.Volume < 0 || r.Volume > 1 {
		return errors.New("生成音量必须在 0 到 1 之间")
	}

	return nil
}

func (r ExportRequest) Validate() error {
	if strings.TrimSpace(r.SourcePath) == "" {
		return errors.New("没有可导出的音频文件")
	}

	return nil
}

func (r ExportRequest) DefaultFileName() string {
	fileName := strings.TrimSpace(r.FileName)
	if fileName == "" {
		return "broadcast.mp3"
	}

	return ensureMP3Extension(fileName)
}
