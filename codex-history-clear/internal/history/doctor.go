package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

var requiredThreadColumns = []string{
	"id",
	"title",
	"rollout_path",
	"created_at",
	"updated_at",
	"cwd",
	"archived",
	"first_user_message",
	"preview",
}

func validateDataModel(paths codexPaths) error {
	if err := validateStateDatabase(paths.stateDB); err != nil {
		return err
	}
	if err := validateOptionalSQLite(paths.logsDB, "logs_2.sqlite"); err != nil {
		return err
	}
	if err := validateOptionalSQLite(paths.goalsDB, "goals_1.sqlite"); err != nil {
		return err
	}
	if err := validateJSONFile(paths.globalState, ".codex-global-state.json"); err != nil {
		return err
	}
	if err := validateJSONFile(paths.globalStateBackup, ".codex-global-state.json.bak"); err != nil {
		return err
	}
	if err := validateJSONLFile(paths.sessionIndex, "session_index.jsonl"); err != nil {
		return err
	}
	if err := validateJSONLFile(paths.history, "history.jsonl"); err != nil {
		return err
	}
	return nil
}

func validateStateDatabase(path string) error {
	if !fileExists(path) {
		return fmt.Errorf("缺少状态库: %s", path)
	}
	db, err := openReadonlyDatabase(path)
	if err != nil {
		return fmt.Errorf("打开状态库失败: %w", err)
	}
	defer db.Close()

	exists, err := tableExists(db, "threads")
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("状态库缺少 threads 表")
	}
	columns, err := tableColumns(db, "threads")
	if err != nil {
		return err
	}
	for _, column := range requiredThreadColumns {
		if !contains(columns, column) {
			return fmt.Errorf("状态库缺少列: threads.%s", column)
		}
	}
	return nil
}

func validateOptionalSQLite(path string, label string) error {
	if !fileExists(path) {
		return nil
	}
	db, err := openReadonlyDatabase(path)
	if err != nil {
		return fmt.Errorf("%s 不可读: %w", label, err)
	}
	return db.Close()
}

func validateJSONFile(path string, label string) error {
	if !fileExists(path) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", label, err)
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("%s 不是有效 JSON: %w", label, err)
	}
	return nil
}

func validateJSONLFile(path string, label string) error {
	if !fileExists(path) {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", label, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var payload any
		if err := json.Unmarshal(line, &payload); err != nil {
			return fmt.Errorf("%s 包含无效 JSONL: %w", label, err)
		}
	}
	return scanner.Err()
}
