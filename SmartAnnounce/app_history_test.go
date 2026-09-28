package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"SmartAnnounce/internal/broadcast"
)

type historyTestSynth struct {
	calls   *int
	succeed *bool
}

func (s historyTestSynth) Synthesize(_ context.Context, _ broadcast.GenerateRequest) ([]byte, broadcast.AudioMetadata, error) {
	*s.calls++
	if *s.succeed {
		return []byte("mp3"), broadcast.AudioMetadata{}, nil
	}
	return nil, broadcast.AudioMetadata{}, errors.New("TTS 请求失败")
}

type unusedAudioStore struct{}

func (unusedAudioStore) SaveAudio(string, []byte) (broadcast.SavedFile, error) {
	return broadcast.SavedFile{}, nil
}
func (unusedAudioStore) ExportAudio(string, string) (broadcast.SavedFile, error) {
	return broadcast.SavedFile{}, nil
}

func TestGenerateBroadcastRecordsTextRegardlessOfAudioFailure(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("APPDATA", configDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	history, err := broadcast.NewTextHistoryStore()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	succeed := false
	service, err := broadcast.NewService(historyTestSynth{calls: &calls, succeed: &succeed}, unusedAudioStore{})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{ctx: context.Background(), broadcastService: service, history: history}
	req := broadcast.GenerateRequest{Text: "  今日特价  ", VoiceID: "female-shaonv", Speed: 1, Volume: 1}
	for i := 0; i < 2; i++ {
		if _, err := app.GenerateBroadcast(req); err == nil || err.Error() != "TTS 请求失败" {
			t.Fatalf("generate %d = %v", i, err)
		}
	}
	entries, err := app.ListTextHistory()
	if err != nil || len(entries) != 1 || entries[0].Text != "今日特价" || calls != 2 {
		t.Fatalf("history = %+v, calls = %d, err = %v", entries, calls, err)
	}
	succeed = true
	req.VoiceID = "male-qn-jingying"
	if _, err := app.GenerateBroadcast(req); err != nil {
		t.Fatalf("audio success = %v", err)
	}
	entries, err = app.ListTextHistory()
	if err != nil || len(entries) != 1 || calls != 3 {
		t.Fatalf("different voice and successful TTS should not duplicate text: %+v, calls=%d, %v", entries, calls, err)
	}
	if err := app.DeleteTextHistory(entries[0].ID); err != nil {
		t.Fatal(err)
	}
	entries, err = app.ListTextHistory()
	if err != nil || len(entries) != 0 {
		t.Fatalf("after delete = %+v, %v", entries, err)
	}
	if _, err := app.GenerateBroadcast(broadcast.GenerateRequest{Text: " "}); err == nil || calls != 3 {
		t.Fatalf("invalid input should not call TTS: %v, calls=%d", err, calls)
	}
	path := filepath.Join(configDir, "SmartAnnounce", "text-history.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := app.GenerateBroadcast(req); err == nil || calls != 3 {
		t.Fatalf("history error must block TTS: %v, calls=%d", err, calls)
	}
}
