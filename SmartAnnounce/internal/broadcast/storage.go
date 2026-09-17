package broadcast

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var invalidFileChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)

type SavedFile struct {
	Name string
	Path string
}

type Storage struct {
	baseDir      string
	buildNameFn func(string) string
}

func NewStorage() (*Storage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}

	baseDir := filepath.Join(homeDir, "Downloads", "SmartAnnounce")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建音频目录失败: %w", err)
	}

	return &Storage{
		baseDir:      baseDir,
		buildNameFn: buildAudioFileName,
	}, nil
}

func (s *Storage) SaveAudio(text string, audioData []byte) (SavedFile, error) {
	fileName := s.fileNameBuilder()(text)
	filePath := filepath.Join(s.baseDir, fileName)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return SavedFile{}, fmt.Errorf("创建音频文件失败: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(audioData); err != nil {
		return SavedFile{}, fmt.Errorf("写入音频文件失败: %w", err)
	}

	return SavedFile{
		Name: fileName,
		Path: filePath,
	}, nil
}

func (s *Storage) ExportAudio(sourcePath string, targetPath string) (SavedFile, error) {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return SavedFile{}, fmt.Errorf("打开源音频失败: %w", err)
	}
	defer sourceFile.Close()

	resolvedTargetPath := strings.TrimSpace(targetPath)
	if resolvedTargetPath == "" {
		return SavedFile{}, fmt.Errorf("导出目标路径不能为空")
	}

	resolvedTargetPath = ensureMP3Extension(resolvedTargetPath)
	if same, err := isSameFilePath(sourcePath, resolvedTargetPath); err != nil {
		return SavedFile{}, err
	} else if same {
		return SavedFile{}, fmt.Errorf("导出目标不能覆盖源音频文件")
	}

	targetFile, err := os.OpenFile(resolvedTargetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return SavedFile{}, fmt.Errorf("导出目标文件已存在，请重新选择保存位置")
		}

		return SavedFile{}, fmt.Errorf("创建导出文件失败: %w", err)
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return SavedFile{}, fmt.Errorf("写入导出文件失败: %w", err)
	}

	return SavedFile{
		Name: filepath.Base(resolvedTargetPath),
		Path: resolvedTargetPath,
	}, nil
}

func buildAudioFileName(text string) string {
	stem := sanitizeFileStem(text)
	timestamp := time.Now().Format("20060102_150405_000000000")
	return fmt.Sprintf("%s_%s.mp3", stem, timestamp)
}

func (s *Storage) fileNameBuilder() func(string) string {
	if s.buildNameFn != nil {
		return s.buildNameFn
	}

	return buildAudioFileName
}

func sanitizeFileStem(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "broadcast"
	}

	normalized := invalidFileChars.ReplaceAllString(trimmed, "_")
	normalized = strings.Join(strings.Fields(normalized), "_")
	runes := []rune(normalized)
	if len(runes) > 18 {
		normalized = string(runes[:18])
	}

	normalized = strings.Trim(normalized, "._")
	if normalized == "" {
		return "broadcast"
	}

	return normalized
}

func ensureMP3Extension(fileName string) string {
	if strings.EqualFold(filepath.Ext(fileName), ".mp3") {
		return fileName
	}

	return fileName + ".mp3"
}

func isSameFilePath(sourcePath string, targetPath string) (bool, error) {
	sourceAbs, err := filepath.Abs(sourcePath)
	if err != nil {
		return false, fmt.Errorf("解析源音频路径失败: %w", err)
	}

	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return false, fmt.Errorf("解析导出路径失败: %w", err)
	}

	sourceClean := filepath.Clean(sourceAbs)
	targetClean := filepath.Clean(targetAbs)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(sourceClean, targetClean), nil
	}

	return sourceClean == targetClean, nil
}
