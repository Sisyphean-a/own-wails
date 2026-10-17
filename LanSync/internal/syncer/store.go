package syncer

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "state.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; PRAGMA busy_timeout=5000; CREATE TABLE IF NOT EXISTS state (key TEXT PRIMARY KEY, value BLOB NOT NULL)"); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) put(key string, v any) error {
	_, err := s.db.Exec("INSERT INTO state(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, encode(v))
	return err
}
func (s *Store) get(key string, v any) (bool, error) {
	var b []byte
	err := s.db.QueryRow("SELECT value FROM state WHERE key=?", key).Scan(&b)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(b, v)
}
func (s *Store) list(prefix string) (map[string]json.RawMessage, error) {
	rows, err := s.db.Query("SELECT key,value FROM state WHERE substr(key,1,?)=?", len(prefix), prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var k string
		var b []byte
		if err = rows.Scan(&k, &b); err != nil {
			return nil, err
		}
		out[k] = b
	}
	return out, rows.Err()
}

// Rule：日常核对只加载共同版本的元信息，发生 Markdown 冲突时才读取正文。
func (s *Store) baselineMetadata(taskID string) (map[string]File, error) {
	prefix := "base/" + taskID + "/"
	rows, err := s.db.Query("SELECT key,json_extract(CAST(value AS TEXT),'$.file') FROM state WHERE substr(key,1,?)=?", len(prefix), prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]File{}
	for rows.Next() {
		var key string
		var data []byte
		if err = rows.Scan(&key, &data); err != nil {
			return nil, err
		}
		var f File
		if err = json.Unmarshal(data, &f); err != nil {
			return nil, err
		}
		out[key] = f
	}
	return out, rows.Err()
}
func (s *Store) remove(key string) error {
	_, err := s.db.Exec("DELETE FROM state WHERE key=?", key)
	return err
}
