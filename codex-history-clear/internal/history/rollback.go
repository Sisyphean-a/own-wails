package history

import (
	"encoding/json"
	"os"
)

func rollbackExecution(journalPath string) (RollbackResult, error) {
	events, err := restoreBackups(journalPath)
	if err != nil {
		return RollbackResult{}, err
	}
	journal, err := loadRollbackJournal(journalPath)
	if err != nil {
		return RollbackResult{}, err
	}
	restoredCount := 0
	for _, entry := range journal.Entries {
		if entry.Restored {
			restoredCount++
		}
	}
	return RollbackResult{
		RunID:         journal.RunID,
		JournalPath:   journalPath,
		RestoredCount: restoredCount,
		Entries:       journal.Entries,
		Events:        events,
	}, nil
}

func restoreBackups(journalPath string) ([]JobEvent, error) {
	journal, err := loadRollbackJournal(journalPath)
	if err != nil {
		return nil, err
	}
	events := []JobEvent{}
	for index := range journal.Entries {
		entry := &journal.Entries[index]
		if err := copyFile(entry.BackupPath, entry.OriginalPath); err != nil {
			return events, err
		}
		entry.Restored = true
		events = append(events, JobEvent{
			Phase:        "rollback",
			ItemIndex:    index + 1,
			ItemTotal:    len(journal.Entries),
			Level:        "info",
			Message:      "已恢复备份文件",
			ArtifactPath: entry.OriginalPath,
		})
	}
	return events, writeJSON(journalPath, journal)
}

func loadRollbackJournal(path string) (rollbackJournal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return rollbackJournal{}, err
	}
	var journal rollbackJournal
	return journal, json.Unmarshal(data, &journal)
}
