package history

import (
	"database/sql"
	"os"
	"strings"
)

func verifyDeletion(paths codexPaths, targets []PlanTarget) (VerificationResult, error) {
	targetIDs := idsFromTargets(targets)
	findings, err := verifyDatabaseStores(paths, targetIDs)
	if err != nil {
		return VerificationResult{}, err
	}
	metadataFindings, err := verifyMetadataStores(paths, targetIDs)
	if err != nil {
		return VerificationResult{}, err
	}
	findings = append(findings, metadataFindings...)
	findings = append(findings, verifyRolloutStores(targets)...)
	return VerificationResult{
		Status:              verificationStatus(findings),
		Summary:             verificationSummary(findings),
		Success:             len(findings) == 0,
		RemainingReferences: findings,
	}, nil
}

func verifyDatabaseStores(paths codexPaths, ids map[string]struct{}) ([]VerificationFinding, error) {
	findings := []VerificationFinding{}
	for _, spec := range []struct {
		path  string
		store string
	}{
		{path: paths.stateDB, store: "state_db"},
		{path: paths.logsDB, store: "logs_db"},
		{path: paths.goalsDB, store: "goals_db"},
	} {
		next, err := verifySQLite(spec.path, spec.store, ids)
		if err != nil {
			return nil, err
		}
		findings = append(findings, next...)
	}
	return findings, nil
}

func verifyMetadataStores(paths codexPaths, ids map[string]struct{}) ([]VerificationFinding, error) {
	findings := []VerificationFinding{}
	for _, check := range []struct {
		run func() ([]VerificationFinding, error)
	}{
		{run: func() ([]VerificationFinding, error) {
			return verifyJSONL(paths.sessionIndex, "session_index", ids, decodeSessionIndexID)
		}},
		{run: func() ([]VerificationFinding, error) {
			return verifyJSONL(paths.history, "history_jsonl", ids, decodeHistorySessionID)
		}},
		{run: func() ([]VerificationFinding, error) {
			return verifyGlobalState(paths.globalState, "global_state", ids)
		}},
		{run: func() ([]VerificationFinding, error) {
			return verifyGlobalState(paths.globalStateBackup, "global_state_backup", ids)
		}},
	} {
		next, err := check.run()
		if err != nil {
			return nil, err
		}
		findings = append(findings, next...)
	}
	return findings, nil
}

func verifyRolloutStores(targets []PlanTarget) []VerificationFinding {
	findings := []VerificationFinding{}
	for _, target := range targets {
		for _, path := range rolloutPathsFromPlanTarget(target) {
			if !fileExists(path) {
				continue
			}
			findings = append(findings, VerificationFinding{
				Store:  "rollout_jsonl",
				Path:   path,
				Detail: "rollout file still exists",
			})
		}
	}
	return findings
}

func verificationStatus(findings []VerificationFinding) string {
	if len(findings) == 0 {
		return "pass"
	}
	return "warn"
}

func verificationSummary(findings []VerificationFinding) string {
	if len(findings) == 0 {
		return "执行后未发现残留引用"
	}
	return "执行后仍有残留引用"
}

func verifySQLite(path string, store string, ids map[string]struct{}) ([]VerificationFinding, error) {
	if !fileExists(path) {
		return nil, nil
	}
	db, err := openReadonlyDatabase(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("select name from sqlite_master where type = 'table' order by name")
	if err != nil {
		return nil, err
	}
	tableNames := []string{}
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, tableName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	findings := []VerificationFinding{}
	for _, tableName := range tableNames {
		columns, err := tableColumns(db, tableName)
		if err != nil {
			return nil, err
		}
		for _, column := range columns {
			if column != "id" && !strings.Contains(column, "thread_id") {
				continue
			}
			if next, err := collectSQLiteHits(db, path, store, tableName, column, ids); err != nil {
				return nil, err
			} else {
				findings = append(findings, next...)
			}
		}
	}
	return findings, nil
}

func collectSQLiteHits(db *sql.DB, path string, store string, table string, column string, ids map[string]struct{}) ([]VerificationFinding, error) {
	findings := []VerificationFinding{}
	for id := range ids {
		count, err := countWhere(db, table, column, id)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			continue
		}
		findings = append(findings, VerificationFinding{
			Store:  store + "." + table,
			Path:   path,
			Detail: column + " still contains target thread id",
		})
	}
	return findings, nil
}

func verifyJSONL(path string, store string, ids map[string]struct{}, decode func([]byte) (string, error)) ([]VerificationFinding, error) {
	count, err := countJSONLMatches(path, ids, decode)
	if err != nil || count == 0 {
		return nil, err
	}
	return []VerificationFinding{{Store: store, Path: path, Detail: "JSONL still contains target thread id"}}, nil
}

func verifyGlobalState(path string, store string, ids map[string]struct{}) ([]VerificationFinding, error) {
	count, err := countGlobalStateMatches(path, ids)
	if err != nil || count == 0 {
		return nil, err
	}
	return []VerificationFinding{{Store: store, Path: path, Detail: "JSON state still contains target thread reference"}}, nil
}

func idsFromTargets(targets []PlanTarget) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, target := range targets {
		ids[target.Thread.ID] = struct{}{}
	}
	return ids
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
