package history

import (
	"path/filepath"

	"codex-history-manager/internal/codexhome"
)

type codexPaths struct {
	codexHome           string
	stateDB             string
	logsDB              string
	goalsDB             string
	sessionIndex        string
	history             string
	globalState         string
	globalStateBackup   string
	shellSnapshotsDir   string
	sessionsDir         string
	archivedSessionsDir string
	scanMetrics         *historyScanMetrics
}

func (s *Service) resolvePaths() (codexPaths, error) {
	codexHome, err := codexhome.Resolve(s.codexHomeOverride, s.userHomeDir, "CODEX_HOME")
	if err != nil {
		return codexPaths{}, err
	}
	return codexPaths{
		codexHome:           codexHome,
		stateDB:             filepath.Join(codexHome, "state_5.sqlite"),
		logsDB:              filepath.Join(codexHome, "logs_2.sqlite"),
		goalsDB:             filepath.Join(codexHome, "goals_1.sqlite"),
		sessionIndex:        filepath.Join(codexHome, "session_index.jsonl"),
		history:             filepath.Join(codexHome, "history.jsonl"),
		globalState:         filepath.Join(codexHome, ".codex-global-state.json"),
		globalStateBackup:   filepath.Join(codexHome, ".codex-global-state.json.bak"),
		shellSnapshotsDir:   filepath.Join(codexHome, "shell-snapshots"),
		sessionsDir:         filepath.Join(codexHome, "sessions"),
		archivedSessionsDir: filepath.Join(codexHome, "archived_sessions"),
		scanMetrics:         s.scanMetrics,
	}, nil
}
