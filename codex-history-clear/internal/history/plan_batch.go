package history

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type planTableSpec struct {
	table  string
	column string
	store  string
	action string
	detail string
}

type planBatchContext struct {
	groups  map[string][]PlanStore
	ids     map[string]struct{}
	metrics *historyScanMetrics
}

type metadataPlanSpec struct {
	path   string
	store  string
	action string
	detail string
}

func planStoresBatch(paths codexPaths, targets []ThreadSummary) (map[string][]PlanStore, error) {
	ids := idsFromThreads(targets)
	groups := make(map[string][]PlanStore, len(ids))
	for id := range ids {
		groups[id] = []PlanStore{}
	}
	context := &planBatchContext{groups: groups, ids: ids, metrics: paths.scanMetrics}
	if err := context.appendDatabasePlans(paths.stateDB, statePlanSpecs()); err != nil {
		return nil, err
	}
	if err := context.appendDatabasePlans(paths.logsDB, []planTableSpec{{"logs", "thread_id", "logs_db.logs", "delete_rows", "delete matching rows"}}); err != nil {
		return nil, err
	}
	if err := context.appendDatabasePlans(paths.goalsDB, []planTableSpec{{"thread_goals", "thread_id", "goals_db.thread_goals", "delete_rows", "delete matching rows"}}); err != nil {
		return nil, err
	}
	if err := context.appendMetadataPlans(paths); err != nil {
		return nil, err
	}
	return groups, nil
}

func statePlanSpecs() []planTableSpec {
	return []planTableSpec{
		{"threads", "id", "state_db.threads", "delete_rows", "delete target thread row"},
		{"thread_dynamic_tools", "thread_id", "state_db.thread_dynamic_tools", "delete_rows", "delete matching rows"},
		{"thread_spawn_edges", "parent_thread_id", "state_db.thread_spawn_edges.parent", "delete_rows", "delete matching rows"},
		{"thread_spawn_edges", "child_thread_id", "state_db.thread_spawn_edges.child", "delete_rows", "delete matching rows"},
		{"agent_job_items", "assigned_thread_id", "state_db.agent_job_items", "rewrite_rows", "clear assigned_thread_id references"},
	}
}

func (context *planBatchContext) appendDatabasePlans(path string, specs []planTableSpec) error {
	if !fileExists(path) {
		for id := range context.ids {
			for _, spec := range specs {
				context.groups[id] = append(context.groups[id], PlanStore{Store: spec.store, Path: path, Action: "inspect", Detail: "store is missing"})
			}
		}
		return nil
	}
	db, err := openReadonlyDatabase(path)
	if err != nil {
		return err
	}
	defer db.Close()
	context.metrics.recordPlanDatabaseOpen()
	for _, spec := range specs {
		counts, err := countTableByIDs(db, spec, context.ids)
		if err != nil {
			return err
		}
		for id := range context.ids {
			context.groups[id] = append(context.groups[id], PlanStore{Store: spec.store, Path: path, Action: spec.action, Detail: spec.detail, Count: counts[id], Exists: true})
		}
	}
	return nil
}

func countTableByIDs(db *sql.DB, spec planTableSpec, ids map[string]struct{}) (map[string]int64, error) {
	counts := map[string]int64{}
	exists, err := tableExists(db, spec.table)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("数据库缺少表 %s", spec.table)
	}
	columns, err := tableColumns(db, spec.table)
	if err != nil {
		return nil, err
	}
	if !contains(columns, spec.column) {
		return nil, fmt.Errorf("数据库表 %s 缺少列 %s", spec.table, spec.column)
	}
	placeholders, args := inClause(ids)
	query := fmt.Sprintf("select %s, count(*) from %s where %s in (%s) group by %s", quoteIdentifier(spec.column), quoteIdentifier(spec.table), quoteIdentifier(spec.column), placeholders, quoteIdentifier(spec.column))
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var count int64
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		counts[id] = count
	}
	return counts, rows.Err()
}

func (context *planBatchContext) appendMetadataPlans(paths codexPaths) error {
	for _, spec := range []struct {
		path   string
		store  string
		decode func([]byte) (string, error)
	}{{paths.sessionIndex, "session_index", decodeSessionIndexID}, {paths.history, "history_jsonl", decodeHistorySessionID}} {
		if fileExists(spec.path) {
			context.metrics.recordPlanJSONLScan()
		}
		counts, err := countJSONLByID(spec.path, context.ids, spec.decode)
		if err != nil {
			return err
		}
		context.appendCountedPlans(metadataPlanSpec{spec.path, spec.store, "rewrite_jsonl", "remove matching entries"}, counts)
	}
	for _, spec := range []struct{ path, store string }{{paths.globalState, "global_state"}, {paths.globalStateBackup, "global_state_backup"}} {
		if fileExists(spec.path) {
			context.metrics.recordPlanJSONScan()
		}
		counts, err := countGlobalStateByID(spec.path, context.ids)
		if err != nil {
			return err
		}
		context.appendCountedPlans(metadataPlanSpec{spec.path, spec.store, "rewrite_json", "remove known thread references"}, counts)
	}
	return nil
}

func (context *planBatchContext) appendCountedPlans(spec metadataPlanSpec, counts map[string]int64) {
	for id := range context.ids {
		context.groups[id] = append(context.groups[id], PlanStore{
			Store: spec.store, Path: spec.path, Action: spec.action, Detail: spec.detail,
			Count: counts[id], Exists: fileExists(spec.path),
		})
	}
}

func countJSONLByID(path string, ids map[string]struct{}, decode func([]byte) (string, error)) (map[string]int64, error) {
	counts := map[string]int64{}
	if !fileExists(path) {
		return counts, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		id, err := decode(line)
		if err != nil {
			return nil, err
		}
		if _, ok := ids[id]; ok {
			counts[id]++
		}
	}
	return counts, scanner.Err()
}

func countGlobalStateByID(path string, ids map[string]struct{}) (map[string]int64, error) {
	counts := map[string]int64{}
	if !fileExists(path) {
		return counts, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	countThreadReferences(value, ids, counts)
	return counts, nil
}

func countThreadReferences(value any, ids map[string]struct{}, counts map[string]int64) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				if _, hit := ids[text]; hit {
					counts[text]++
					continue
				}
			}
			countThreadReferences(item, ids, counts)
		}
	case map[string]any:
		for key, item := range typed {
			matched := false
			for id := range ids {
				if containsSubstring(key, id) {
					counts[id]++
					matched = true
				}
			}
			if !matched {
				countThreadReferences(item, ids, counts)
			}
		}
	}
}

func idsFromThreads(targets []ThreadSummary) map[string]struct{} {
	ids := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		ids[target.ID] = struct{}{}
	}
	return ids
}
