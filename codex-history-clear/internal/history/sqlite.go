package history

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

func openDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func openReadonlyDatabase(path string) (*sql.DB, error) {
	db, err := openDatabase(path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA query_only = ON"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func tableExists(db *sql.DB, tableName string) (bool, error) {
	row := db.QueryRow("select 1 from sqlite_master where type = 'table' and name = ?", tableName)
	var ok int
	if err := row.Scan(&ok); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return ok == 1, nil
}

func tableColumns(db *sql.DB, tableName string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf("pragma table_info(%s)", quoteIdentifier(tableName)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := []string{}
	for rows.Next() {
		var cid int
		var name string
		var kind sql.NullString
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

func countWhere(db *sql.DB, tableName string, columnName string, value string) (int64, error) {
	exists, err := tableExists(db, tableName)
	if err != nil || !exists {
		return 0, err
	}
	columns, err := tableColumns(db, tableName)
	if err != nil {
		return 0, err
	}
	if !contains(columns, columnName) {
		return 0, nil
	}
	query := fmt.Sprintf(
		"select count(*) from %s where %s = ?",
		quoteIdentifier(tableName),
		quoteIdentifier(columnName),
	)
	var count int64
	if err := db.QueryRow(query, value).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func checkpointWal(path string) error {
	if !fileExists(path) {
		return nil
	}
	db, err := openDatabase(path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
