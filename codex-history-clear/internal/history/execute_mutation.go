package history

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
)

func performExecution(paths codexPaths, targets []PlanTarget, events *[]JobEvent) ([]MutationResult, error) {
	ids := idsFromTargets(targets)
	results := []MutationResult{}
	emit(events, "backup", 0, len(targets), "info", "备份完成，开始改写本地存储", "")

	stateResults, err := executeStateDatabase(paths.stateDB, ids)
	if err != nil {
		return nil, err
	}
	results = append(results, stateResults...)

	logResults, err := executeSimpleDelete(paths.logsDB, "logs", "thread_id", ids, "logs_db.logs")
	if err != nil {
		return nil, err
	}
	results = append(results, logResults...)

	goalResults, err := executeSimpleDelete(paths.goalsDB, "thread_goals", "thread_id", ids, "goals_db.thread_goals")
	if err != nil {
		return nil, err
	}
	results = append(results, goalResults...)

	jsonResults, err := executeFileRewrites(paths, ids)
	if err != nil {
		return nil, err
	}
	results = append(results, jsonResults...)

	fileResults, err := executeFileDeletes(paths, targets)
	if err != nil {
		return nil, err
	}
	results = append(results, fileResults...)
	emit(events, "delete", len(targets), len(targets), "info", "本地历史删除执行完成", "")
	return results, nil
}

func executeStateDatabase(path string, ids map[string]struct{}) ([]MutationResult, error) {
	if !fileExists(path) {
		return nil, nil
	}
	db, err := openDatabase(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	results, err := deleteStateRows(tx, path, ids)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return results, tx.Commit()
}

func deleteStateRows(tx *sql.Tx, path string, ids map[string]struct{}) ([]MutationResult, error) {
	results := []MutationResult{}
	for _, spec := range []struct {
		table  string
		column string
		store  string
	}{
		{table: "threads", column: "id", store: "state_db.threads"},
		{table: "thread_dynamic_tools", column: "thread_id", store: "state_db.thread_dynamic_tools"},
		{table: "thread_spawn_edges", column: "parent_thread_id", store: "state_db.thread_spawn_edges.parent"},
		{table: "thread_spawn_edges", column: "child_thread_id", store: "state_db.thread_spawn_edges.child"},
	} {
		result, err := deleteRows(tx, path, spec.table, spec.column, ids, spec.store)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	rewriteResult, err := clearAssignedThreadIDs(tx, path, ids)
	if err != nil {
		return nil, err
	}
	return append(results, rewriteResult), nil
}

func executeSimpleDelete(path string, table string, column string, ids map[string]struct{}, store string) ([]MutationResult, error) {
	if !fileExists(path) {
		return nil, nil
	}
	db, err := openDatabase(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	result, err := deleteRows(tx, path, table, column, ids, store)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return []MutationResult{result}, tx.Commit()
}

func executeFileRewrites(paths codexPaths, ids map[string]struct{}) ([]MutationResult, error) {
	results := []MutationResult{}
	for _, rewrite := range []struct {
		store   string
		action  string
		path    string
		execute func() (bool, int64, error)
	}{
		{store: "session_index", action: "rewrite_jsonl", path: paths.sessionIndex, execute: func() (bool, int64, error) { return rewriteSessionIndex(paths.sessionIndex, ids) }},
		{store: "history_jsonl", action: "rewrite_jsonl", path: paths.history, execute: func() (bool, int64, error) { return rewriteHistory(paths.history, ids) }},
		{store: "global_state", action: "rewrite_json", path: paths.globalState, execute: func() (bool, int64, error) { return rewriteGlobalState(paths.globalState, ids) }},
		{store: "global_state_backup", action: "rewrite_json", path: paths.globalStateBackup, execute: func() (bool, int64, error) { return rewriteGlobalState(paths.globalStateBackup, ids) }},
	} {
		changed, count, err := rewrite.execute()
		if err != nil {
			return nil, err
		}
		results = append(results, MutationResult{
			Store:       rewrite.store,
			Action:      rewrite.action,
			Path:        rewrite.path,
			ChangedRows: count,
			Changed:     changed,
		})
	}
	return results, nil
}

func executeFileDeletes(paths codexPaths, targets []PlanTarget) ([]MutationResult, error) {
	results := []MutationResult{}
	for _, target := range targets {
		for _, store := range target.Stores {
			if store.Action != "delete_file" || !store.Exists || store.Path == "" {
				continue
			}
			if err := validateDeleteStore(paths, target.Thread, store); err != nil {
				return nil, err
			}
			if err := os.Remove(store.Path); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			results = append(results, MutationResult{
				Store:       store.Store,
				Action:      "delete_file",
				Path:        store.Path,
				ChangedRows: store.Count,
				Changed:     true,
			})
		}
	}
	return results, nil
}

func deleteRows(tx *sql.Tx, path string, table string, column string, ids map[string]struct{}, store string) (MutationResult, error) {
	query, args := deleteQuery(table, column, ids)
	result, err := tx.Exec(query, args...)
	if err != nil {
		return MutationResult{}, err
	}
	changed, _ := result.RowsAffected()
	return MutationResult{Store: store, Action: "delete_rows", Path: path, ChangedRows: changed, Changed: changed > 0}, nil
}

func clearAssignedThreadIDs(tx *sql.Tx, path string, ids map[string]struct{}) (MutationResult, error) {
	query, args := updateNullQuery("agent_job_items", "assigned_thread_id", ids)
	result, err := tx.Exec(query, args...)
	if err != nil {
		return MutationResult{}, err
	}
	changed, _ := result.RowsAffected()
	return MutationResult{Store: "state_db.agent_job_items", Action: "rewrite_rows", Path: path, ChangedRows: changed, Changed: changed > 0}, nil
}

func checkpointMutatedDatabases(paths codexPaths) error {
	for _, path := range []string{paths.stateDB, paths.logsDB, paths.goalsDB} {
		if err := checkpointWal(path); err != nil {
			return err
		}
	}
	return nil
}

func deleteQuery(table string, column string, ids map[string]struct{}) (string, []any) {
	placeholders, args := inClause(ids)
	query := fmt.Sprintf("delete from %s where %s in (%s)", quoteIdentifier(table), quoteIdentifier(column), placeholders)
	return query, args
}

func updateNullQuery(table string, column string, ids map[string]struct{}) (string, []any) {
	placeholders, args := inClause(ids)
	query := fmt.Sprintf("update %s set %s = null where %s in (%s)", quoteIdentifier(table), quoteIdentifier(column), quoteIdentifier(column), placeholders)
	return query, args
}

func inClause(ids map[string]struct{}) (string, []any) {
	keys := make([]string, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	args := make([]any, 0, len(keys))
	parts := make([]string, 0, len(keys))
	for _, id := range keys {
		parts = append(parts, "?")
		args = append(args, id)
	}
	return strings.Join(parts, ","), args
}
