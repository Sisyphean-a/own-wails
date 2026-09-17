package broadcast

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubSynthesizer struct {
	audio    []byte
	metadata AudioMetadata
	err      error
}

func (s stubSynthesizer) Synthesize(_ context.Context, _ GenerateRequest) ([]byte, AudioMetadata, error) {
	if s.err != nil {
		return nil, AudioMetadata{}, s.err
	}

	return s.audio, s.metadata, nil
}

type stubStorage struct {
	savedFile SavedFile
	exported  SavedFile
	saveErr   error
	exportErr error
}

func (s stubStorage) SaveAudio(_ string, _ []byte) (SavedFile, error) {
	if s.saveErr != nil {
		return SavedFile{}, s.saveErr
	}

	return s.savedFile, nil
}

func (s stubStorage) ExportAudio(_, _ string) (SavedFile, error) {
	if s.exportErr != nil {
		return SavedFile{}, s.exportErr
	}

	return s.exported, nil
}

func TestNewServiceRequiresDependencies(t *testing.T) {
	storage := stubStorage{}
	client := stubSynthesizer{}

	if _, err := NewService(nil, storage); err == nil {
		t.Fatal("NewService() expected error for nil client")
	}

	if _, err := NewService(client, nil); err == nil {
		t.Fatal("NewService() expected error for nil storage")
	}
}

func TestServiceGenerate(t *testing.T) {
	service, err := NewService(
		stubSynthesizer{
			audio: []byte("fake-mp3-data"),
			metadata: AudioMetadata{
				DurationMillis:  1500,
				AudioSampleRate: 32000,
			},
		},
		stubStorage{
			savedFile: SavedFile{
				Name: "broadcast.mp3",
				Path: "C:/Downloads/SmartAnnounce/broadcast.mp3",
			},
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, err := service.Generate(context.Background(), GenerateRequest{
		Text:    "今日特价鸡蛋",
		VoiceID: "female-shaonv",
		Speed:   1,
		Volume:  1,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.FileName != "broadcast.mp3" {
		t.Fatalf("Generate() file name = %q", result.FileName)
	}

	if !strings.HasPrefix(result.AudioDataURI, "data:audio/mpeg;base64,") {
		t.Fatalf("Generate() audio data uri = %q", result.AudioDataURI)
	}

	if result.CharacterCount != len([]rune("今日特价鸡蛋")) {
		t.Fatalf("Generate() character count = %d", result.CharacterCount)
	}
}

func TestServiceGeneratePropagatesSaveError(t *testing.T) {
	service, err := NewService(
		stubSynthesizer{audio: []byte("fake-mp3-data")},
		stubStorage{saveErr: errors.New("save failed")},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, err := service.Generate(context.Background(), GenerateRequest{
		Text:    "今日特价鸡蛋",
		VoiceID: "female-shaonv",
		Speed:   1,
		Volume:  1,
	}); err == nil {
		t.Fatal("Generate() expected save error")
	}
}

func TestServiceExport(t *testing.T) {
	service, err := NewService(
		stubSynthesizer{},
		stubStorage{
			exported: SavedFile{
				Name: "exported.mp3",
				Path: "D:/Audio/exported.mp3",
			},
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, err := service.Export(
		ExportRequest{
			SourcePath: "C:/Downloads/SmartAnnounce/source.mp3",
			FileName:   "source.mp3",
		},
		"D:/Audio/exported.mp3",
	)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if result.FilePath != "D:/Audio/exported.mp3" {
		t.Fatalf("Export() file path = %q", result.FilePath)
	}
}
