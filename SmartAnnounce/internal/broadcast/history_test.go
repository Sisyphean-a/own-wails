package broadcast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTextHistoryDailyDeduplicationAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "text-history.json")
	day := time.Date(2026, time.June, 18, 9, 20, 0, 0, time.Local)
	store := &TextHistoryStore{path: path, now: func() time.Time { return day }}
	first, err := store.Record("  今日促销\n欢迎光临  ")
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != "今日促销\n欢迎光临" || first.Day != "2026-06-18" || first.ID == "" {
		t.Fatalf("first entry = %+v", first)
	}
	day = day.Add(time.Hour)
	again, err := store.Record("今日促销\n欢迎光临")
	if err != nil || again != first {
		t.Fatalf("same-day duplicate = %+v, %v", again, err)
	}
	other, err := store.Record("今日营业通知")
	if err != nil || other.ID == first.ID {
		t.Fatalf("different text = %+v, %v", other, err)
	}
	day = day.AddDate(0, 0, 1)
	nextDay, err := store.Record("今日促销\n欢迎光临")
	if err != nil || nextDay.Day != "2026-06-19" || nextDay.ID == first.ID {
		t.Fatalf("next day = %+v, %v", nextDay, err)
	}
	reopened := &TextHistoryStore{path: path, now: func() time.Time { return day }}
	entries, err := reopened.List()
	if err != nil || len(entries) != 3 || entries[0].ID != nextDay.ID || entries[2].ID != first.ID {
		t.Fatalf("reloaded entries = %+v, %v", entries, err)
	}
	if err := reopened.Delete(other.ID); err != nil {
		t.Fatal(err)
	}
	entries, err = store.List()
	if err != nil || len(entries) != 2 || entries[0].ID != nextDay.ID || entries[1].ID != first.ID {
		t.Fatalf("after delete = %+v, %v", entries, err)
	}
	if err := store.Delete(other.ID); err == nil {
		t.Fatal("deleting an absent entry must fail")
	}
}

func TestTextHistoryReadAndWriteFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "text-history.json")
	store := &TextHistoryStore{path: path, now: time.Now}
	if entries, err := store.List(); err != nil || len(entries) != 0 {
		t.Fatalf("missing history = %+v, %v", entries, err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record("新文案"); err == nil || !strings.Contains(err.Error(), "解析文案历史") {
		t.Fatalf("corrupt history should block recording: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"entries":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record("新文案"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("id"); err == nil {
		t.Fatal("unreadable history must not appear empty")
	}
}
