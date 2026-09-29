package broadcast

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

const (
	defaultMinimaxModel = "speech-2.8-hd"
	proxyDirect         = "direct"
	proxyCustom         = "custom"
)

type SettingsInfo struct {
	Model         string `json:"model"`
	HasAPIKey     bool   `json:"hasApiKey"`
	ProxyMode     string `json:"proxyMode"`
	ProxyURL      string `json:"proxyUrl"`
	CustomVoiceID string `json:"customVoiceId"`
}

type SettingsUpdate struct {
	APIKey        string `json:"apiKey"`
	Model         string `json:"model"`
	ProxyMode     string `json:"proxyMode"`
	ProxyURL      string `json:"proxyUrl"`
	CustomVoiceID string `json:"customVoiceId"`
}

type minimaxSettings struct {
	APIKey        string
	Model         string
	ProxyMode     string
	ProxyURL      string
	CustomVoiceID string
}

type settingsFile struct {
	Model           string `json:"model"`
	ProtectedAPIKey []byte `json:"protectedApiKey"`
	ProxyMode       string `json:"proxyMode,omitempty"`
	ProxyURL        string `json:"proxyUrl,omitempty"`
	CustomVoiceID   string `json:"customVoiceId,omitempty"`
}

type SettingsStore struct {
	path string
	mu   sync.Mutex
}

func NewSettingsStore() (*SettingsStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("定位应用配置目录失败: %w", err)
	}
	return &SettingsStore{path: filepath.Join(dir, "SmartAnnounce", "minimax.json")}, nil
}

func (s *SettingsStore) Load() (minimaxSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *SettingsStore) Info() (SettingsInfo, error) {
	settings, err := s.Load()
	if err != nil {
		return SettingsInfo{}, err
	}
	return SettingsInfo{Model: settings.Model, HasAPIKey: settings.APIKey != "", ProxyMode: settings.ProxyMode, ProxyURL: settings.ProxyURL, CustomVoiceID: settings.CustomVoiceID}, nil
}

func (s *SettingsStore) Save(update SettingsUpdate) (SettingsInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.load()
	if err != nil {
		return SettingsInfo{}, err
	}
	model := strings.TrimSpace(update.Model)
	if model == "" {
		return SettingsInfo{}, errors.New("模型名称不能为空")
	}
	key := strings.TrimSpace(update.APIKey)
	if key == "" {
		key = current.APIKey
	}
	if key == "" {
		return SettingsInfo{}, errors.New("请填写 Minimax API Key")
	}
	mode := update.ProxyMode
	if mode == "" {
		mode = current.ProxyMode
	}
	proxyAddress := strings.TrimSpace(update.ProxyURL)
	if mode == proxyCustom && proxyAddress == "" && update.ProxyMode == "" {
		proxyAddress = current.ProxyURL
	}
	if err := validateProxy(mode, proxyAddress); err != nil {
		return SettingsInfo{}, err
	}
	if mode == proxyDirect {
		proxyAddress = ""
	}
	voiceID := strings.TrimSpace(update.CustomVoiceID)
	if err := validateCustomVoiceID(voiceID); err != nil {
		return SettingsInfo{}, err
	}

	protected, err := protectSecret([]byte(key))
	if err != nil {
		return SettingsInfo{}, fmt.Errorf("加密 API Key 失败: %w", err)
	}
	data, err := json.Marshal(settingsFile{Model: model, ProtectedAPIKey: protected, ProxyMode: mode, ProxyURL: proxyAddress, CustomVoiceID: voiceID})
	if err != nil {
		return SettingsInfo{}, fmt.Errorf("编码应用配置失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return SettingsInfo{}, fmt.Errorf("创建应用配置目录失败: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(s.path), ".minimax-*")
	if err != nil {
		return SettingsInfo{}, fmt.Errorf("创建临时配置失败: %w", err)
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return SettingsInfo{}, fmt.Errorf("限制配置访问权限失败: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return SettingsInfo{}, fmt.Errorf("写入应用配置失败: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return SettingsInfo{}, fmt.Errorf("同步应用配置失败: %w", err)
	}
	if err := file.Close(); err != nil {
		return SettingsInfo{}, fmt.Errorf("关闭应用配置失败: %w", err)
	}
	if err := os.Rename(file.Name(), s.path); err != nil {
		return SettingsInfo{}, fmt.Errorf("替换应用配置失败: %w", err)
	}
	return SettingsInfo{Model: model, HasAPIKey: true, ProxyMode: mode, ProxyURL: proxyAddress, CustomVoiceID: voiceID}, nil
}

func (s *SettingsStore) load() (minimaxSettings, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return minimaxSettings{Model: defaultMinimaxModel, ProxyMode: proxyDirect}, nil
	}
	if err != nil {
		return minimaxSettings{}, fmt.Errorf("读取应用配置失败: %w", err)
	}
	var stored settingsFile
	if err := json.Unmarshal(data, &stored); err != nil {
		return minimaxSettings{}, fmt.Errorf("解析应用配置失败: %w", err)
	}
	if strings.TrimSpace(stored.Model) == "" || len(stored.ProtectedAPIKey) == 0 {
		return minimaxSettings{}, errors.New("应用配置不完整，请检查配置文件")
	}
	if stored.ProxyMode == "" {
		stored.ProxyMode = proxyDirect
	}
	if err := validateProxy(stored.ProxyMode, stored.ProxyURL); err != nil {
		return minimaxSettings{}, err
	}
	if err := validateCustomVoiceID(stored.CustomVoiceID); err != nil {
		return minimaxSettings{}, err
	}
	key, err := unprotectSecret(stored.ProtectedAPIKey)
	if err != nil {
		return minimaxSettings{}, fmt.Errorf("解密 API Key 失败: %w", err)
	}
	return minimaxSettings{APIKey: string(key), Model: stored.Model, ProxyMode: stored.ProxyMode, ProxyURL: stored.ProxyURL, CustomVoiceID: stored.CustomVoiceID}, nil
}

func validateCustomVoiceID(id string) error {
	if len(id) > 256 || strings.IndexFunc(id, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return errors.New("自定义音色 ID 不能包含空白字符且不能超过 256 字符")
	}
	return nil
}

func validateProxy(mode, address string) error {
	switch mode {
	case proxyDirect:
		return nil
	case proxyCustom:
		parsed, err := url.Parse(address)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.Port() == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("代理地址必须是无账号密码的 http:// 或 https:// 主机:端口")
		}
		return nil
	default:
		return errors.New("代理模式无效，请重新选择")
	}
}
