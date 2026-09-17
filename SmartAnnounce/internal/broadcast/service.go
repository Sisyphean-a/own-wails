package broadcast

import (
	"context"
	"encoding/base64"
	"fmt"
)

type Synthesizer interface {
	Synthesize(ctx context.Context, req GenerateRequest) ([]byte, AudioMetadata, error)
}

type AudioStorage interface {
	SaveAudio(text string, audioData []byte) (SavedFile, error)
	ExportAudio(sourcePath string, targetPath string) (SavedFile, error)
}

type Service struct {
	client  Synthesizer
	storage AudioStorage
}

func NewDefaultService() (*Service, error) {
	storage, err := NewStorage()
	if err != nil {
		return nil, err
	}

	return NewService(NewMinimaxClient(), storage)
}

func NewService(client Synthesizer, storage AudioStorage) (*Service, error) {
	if client == nil {
		return nil, fmt.Errorf("TTS 客户端不能为空")
	}

	if storage == nil {
		return nil, fmt.Errorf("音频存储不能为空")
	}

	return &Service{
		client:  client,
		storage: storage,
	}, nil
}

func (s *Service) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	if err := req.Validate(); err != nil {
		return GenerateResult{}, err
	}

	audioBytes, metadata, err := s.client.Synthesize(ctx, req)
	if err != nil {
		return GenerateResult{}, err
	}

	savedFile, err := s.storage.SaveAudio(req.NormalizedText(), audioBytes)
	if err != nil {
		return GenerateResult{}, err
	}

	return GenerateResult{
		FileName:        savedFile.Name,
		FilePath:        savedFile.Path,
		AudioDataURI:    buildAudioDataURI(audioBytes),
		VoiceID:         req.VoiceID,
		CharacterCount:  len([]rune(req.NormalizedText())),
		DurationMillis:  metadata.DurationMillis,
		AudioSampleRate: metadata.AudioSampleRate,
	}, nil
}

func (s *Service) Export(req ExportRequest, targetPath string) (ExportResult, error) {
	if err := req.Validate(); err != nil {
		return ExportResult{}, err
	}

	savedFile, err := s.storage.ExportAudio(req.SourcePath, targetPath)
	if err != nil {
		return ExportResult{}, err
	}

	return ExportResult{FilePath: savedFile.Path}, nil
}

func buildAudioDataURI(audioBytes []byte) string {
	encoded := base64.StdEncoding.EncodeToString(audioBytes)
	return "data:audio/mpeg;base64," + encoded
}
