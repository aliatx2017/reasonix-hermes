package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// sqliteStorage persists entries in a SQLite database with indexed search.
type sqliteStorage struct {
	db *sql.DB
}

func newSQLiteStorage(dir string) (*sqliteStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "memories.db")

	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id          TEXT PRIMARY KEY,
			session_id  TEXT NOT NULL DEFAULT '',
			content     TEXT NOT NULL,
			tags        TEXT NOT NULL DEFAULT '[]',
			created_at  TEXT NOT NULL,
			access_count INTEGER NOT NULL DEFAULT 0,
			ttl_ns       INTEGER NOT NULL DEFAULT 0,
			expires_at   TEXT NOT NULL DEFAULT '',
			importance   REAL NOT NULL DEFAULT 0.5,
			vector       TEXT NOT NULL DEFAULT '{}'
		);
		CREATE INDEX IF NOT EXISTS idx_memories_session ON memories(session_id);
		CREATE INDEX IF NOT EXISTS idx_memories_expires ON memories(expires_at);
		CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &sqliteStorage{db: db}, nil
}

func (s *sqliteStorage) Load() ([]MemoryEntry, error) {
	rows, err := s.db.Query(`SELECT id, session_id, content, tags, created_at, access_count, ttl_ns, expires_at, importance, vector FROM memories ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		var tagsJSON, createdAtStr, expiresAtStr, vectorJSON string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Content, &tagsJSON,
			&createdAtStr, &e.AccessCount, &e.TTL, &expiresAtStr, &e.Importance, &vectorJSON); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &e.Tags)
		json.Unmarshal([]byte(vectorJSON), &e.Vector)
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		e.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *sqliteStorage) Save(entries []MemoryEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear and re-insert (simple; replace with upsert for large datasets)
	if _, err := tx.Exec(`DELETE FROM memories`); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO memories (id, session_id, content, tags, created_at, access_count, ttl_ns, expires_at, importance, vector) VALUES (?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		tagsJSON, _ := json.Marshal(e.Tags)
		vectorJSON, _ := json.Marshal(e.Vector)
		createdAt := e.CreatedAt.Format(time.RFC3339)
		expiresAt := ""
		if !e.ExpiresAt.IsZero() {
			expiresAt = e.ExpiresAt.Format(time.RFC3339)
		}
		if _, err := stmt.Exec(e.ID, e.SessionID, e.Content, string(tagsJSON),
			createdAt, e.AccessCount, int64(e.TTL), expiresAt, e.Importance, string(vectorJSON)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Search performs a keyword search across content and tags using SQLite FTS-like LIKE matching.
func (s *sqliteStorage) Search(query, sessionID string, tags []string, limit int) ([]MemoryEntry, error) {
	var conditions []string
	var args []any

	if sessionID != "" {
		conditions = append(conditions, "session_id = ?")
		args = append(args, sessionID)
	}
	if query != "" {
		conditions = append(conditions, "(content LIKE ? OR tags LIKE ?)")
		like := "%" + query + "%"
		args = append(args, like, like)
	}
	for _, tag := range tags {
		conditions = append(conditions, "tags LIKE ?")
		args = append(args, "%\""+tag+"\"%")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	q := fmt.Sprintf(`SELECT id, session_id, content, tags, created_at, access_count, ttl_ns, expires_at, importance, vector FROM memories %s ORDER BY importance DESC, created_at DESC LIMIT ?`, where)
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		var tagsJSON, createdAtStr, expiresAtStr, vectorJSON string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Content, &tagsJSON,
			&createdAtStr, &e.AccessCount, &e.TTL, &expiresAtStr, &e.Importance, &vectorJSON); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &e.Tags)
		json.Unmarshal([]byte(vectorJSON), &e.Vector)
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		e.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Close releases the database connection.
func (s *sqliteStorage) Close() error {
	return s.db.Close()
}
