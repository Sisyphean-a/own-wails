package broadcast

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TextHistoryEntry struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Day       string `json:"day"`
	CreatedAt string `json:"createdAt"`
}

type textHistoryFile struct {
	Entries []TextHistoryEntry `json:"entries"`
}

type TextHistoryStore struct {
	path string
	mu   sync.Mutex
	now  func() time.Time
}

func NewTextHistoryStore() (*TextHistoryStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("定位文案历史目录失败: %w", err)
	}
	return &TextHistoryStore{path: filepath.Join(dir, "SmartAnnounce", "text-history.json"), now: time.Now}, nil
}

func (s *TextHistoryStore) List() ([]TextHistoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *TextHistoryStore) Record(text string) (TextHistoryEntry, error) {
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) > maxTextLength {
		return TextHistoryEntry{}, errors.New("文案不能为空且不能超过 2000 个字符")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return TextHistoryEntry{}, err
	}
	now := s.now().Local()
	day := now.Format("2006-01-02")
	for _, entry := range entries {
		if entry.Day == day && entry.Text == text {
			return entry, nil
		}
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return TextHistoryEntry{}, fmt.Errorf("创建文案历史编号失败: %w", err)
	}
	entry := TextHistoryEntry{ID: hex.EncodeToString(idBytes), Text: text, Day: day, CreatedAt: now.Format(time.RFC3339Nano)}
	entries = append([]TextHistoryEntry{entry}, entries...)
	if err := s.save(entries); err != nil {
		return TextHistoryEntry{}, err
	}
	return entry, nil
}

func (s *TextHistoryStore) Delete(id string) error {
	if id == "" {
		return errors.New("请选择要删除的文案")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return err
	}
	for i, entry := range entries {
		if entry.ID == id {
			return s.save(append(entries[:i:i], entries[i+1:]...))
		}
	}
	return errors.New("文案记录不存在，请刷新历史后重试")
}

func (s *TextHistoryStore) load() ([]TextHistoryEntry, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []TextHistoryEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取文案历史失败: %w", err)
	}
	var stored textHistoryFile
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("解析文案历史失败: %w", err)
	}
	if stored.Entries == nil {
		return nil, errors.New("文案历史格式无效，请检查历史文件")
	}
	for _, entry := range stored.Entries {
		if entry.ID == "" || entry.Text == "" || entry.Day == "" || entry.CreatedAt == "" {
			return nil, errors.New("文案历史记录不完整，请检查历史文件")
		}
	}
	return stored.Entries, nil
}

func (s *TextHistoryStore) save(entries []TextHistoryEntry) error {
	data, err := json.Marshal(textHistoryFile{Entries: entries})
	if err != nil {
		return fmt.Errorf("编码文案历史失败: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("创建文案历史目录失败: %w", err)
	}
	file, err := os.CreateTemp(dir, ".text-history-*")
	if err != nil {
		return fmt.Errorf("创建临时文案历史失败: %w", err)
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return fmt.Errorf("限制文案历史访问权限失败: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("写入文案历史失败: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("同步文案历史失败: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭文案历史失败: %w", err)
	}
	if err := os.Rename(file.Name(), s.path); err != nil {
		return fmt.Errorf("替换文案历史失败: %w", err)
	}
	return nil
}
