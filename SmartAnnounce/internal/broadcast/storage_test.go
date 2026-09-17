package broadcast

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSanitizeFileStem(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "replace invalid characters",
			text: `今日特价:苹果/鸡蛋`,
			want: "今日特价_苹果_鸡蛋",
		},
		{
			name: "trim blank content",
			text: "   ",
			want: "broadcast",
		},
		{
			name: "truncate long content",
			text: "这是一个超长的文件名用于验证截断逻辑是否生效",
			want: "这是一个超长的文件名用于验证截断逻辑",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFileStem(tt.text)
			if got != tt.want {
				t.Fatalf("sanitizeFileStem() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildAudioFileName(t *testing.T) {
	fileName := buildAudioFileName("今日特价苹果")
	if !strings.HasPrefix(fileName, "今日特价苹果_") {
		t.Fatalf("unexpected file name prefix: %s", fileName)
	}

	pattern := regexp.MustCompile(`^今日特价苹果_\d{8}_\d{6}_\d{9}\.mp3$`)
	if !pattern.MatchString(fileName) {
		t.Fatalf("unexpected file name format: %s", fileName)
	}

	if !strings.HasSuffix(fileName, ".mp3") {
		t.Fatalf("unexpected file name suffix: %s", fileName)
	}
}

func TestEnsureMP3Extension(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "append extension", text: "broadcast", want: "broadcast.mp3"},
		{name: "keep lowercase extension", text: "broadcast.mp3", want: "broadcast.mp3"},
		{name: "keep uppercase extension", text: "broadcast.MP3", want: "broadcast.MP3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureMP3Extension(tt.text)
			if got != tt.want {
				t.Fatalf("ensureMP3Extension() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExportAudio(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.mp3")
	targetPath := filepath.Join(tempDir, "exported")
	audioData := []byte("fake-mp3-data")

	if err := os.WriteFile(sourcePath, audioData, 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	storage := &Storage{baseDir: tempDir}
	exported, err := storage.ExportAudio(sourcePath, targetPath)
	if err != nil {
		t.Fatalf("ExportAudio() error = %v", err)
	}

	if exported.Path != targetPath+".mp3" {
		t.Fatalf("ExportAudio() path = %q, want %q", exported.Path, targetPath+".mp3")
	}

	got, err := os.ReadFile(exported.Path)
	if err != nil {
		t.Fatalf("read exported file: %v", err)
	}

	if string(got) != string(audioData) {
		t.Fatalf("exported content = %q, want %q", string(got), string(audioData))
	}
}

func TestExportAudioEmptyTargetPath(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.mp3")

	if err := os.WriteFile(sourcePath, []byte("fake-mp3-data"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	storage := &Storage{baseDir: tempDir}
	if _, err := storage.ExportAudio(sourcePath, "   "); err == nil {
		t.Fatal("ExportAudio() expected error for empty target path")
	}
}

func TestExportAudioRejectsSourceOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.mp3")
	audioData := []byte("fake-mp3-data")

	if err := os.WriteFile(sourcePath, audioData, 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	storage := &Storage{baseDir: tempDir}
	if _, err := storage.ExportAudio(sourcePath, sourcePath); err == nil {
		t.Fatal("ExportAudio() expected error when target matches source")
	}

	got, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source file: %v", err)
	}

	if string(got) != string(audioData) {
		t.Fatalf("source content = %q, want %q", string(got), string(audioData))
	}
}

func TestExportAudioRejectsExistingTargetFile(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.mp3")
	targetPath := filepath.Join(tempDir, "existing.mp3")
	sourceData := []byte("fake-mp3-data")
	targetData := []byte("existing-audio")

	if err := os.WriteFile(sourcePath, sourceData, 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	if err := os.WriteFile(targetPath, targetData, 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	storage := &Storage{baseDir: tempDir}
	if _, err := storage.ExportAudio(sourcePath, targetPath); err == nil {
		t.Fatal("ExportAudio() expected error when target file already exists")
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target file: %v", err)
	}

	if string(got) != string(targetData) {
		t.Fatalf("target content = %q, want %q", string(got), string(targetData))
	}
}

func TestSaveAudioRejectsExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	fileName := "今日特价苹果_20260424_123456_123456789.mp3"
	filePath := filepath.Join(tempDir, fileName)
	originalData := []byte("original-audio")

	if err := os.WriteFile(filePath, originalData, 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	storage := &Storage{
		baseDir: tempDir,
		buildNameFn: func(string) string {
			return fileName
		},
	}

	if _, err := storage.SaveAudio("今日特价苹果", []byte("new-audio")); err == nil {
		t.Fatal("SaveAudio() expected error for existing file")
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}

	if string(got) != string(originalData) {
		t.Fatalf("existing content = %q, want %q", string(got), string(originalData))
	}
}
