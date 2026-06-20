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
			vector       TEXT NOT NULL DEFAULT '{}',
			dense_vector TEXT NOT NULL DEFAULT ''
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
	rows, err := s.db.Query(`SELECT id, session_id, content, tags, created_at, access_count, ttl_ns, expires_at, importance, vector, dense_vector FROM memories ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		var tagsJSON, createdAtStr, expiresAtStr, vectorJSON, denseVectorJSON string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Content, &tagsJSON,
			&createdAtStr, &e.AccessCount, &e.TTL, &expiresAtStr, &e.Importance, &vectorJSON, &denseVectorJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tagsJSON), &e.Tags); err != nil {
			e.Tags = nil
		}
		if err := json.Unmarshal([]byte(vectorJSON), &e.Vector); err != nil {
			e.Vector = nil
		}
		if err := json.Unmarshal([]byte(denseVectorJSON), &e.DenseVector); err != nil {
			e.DenseVector = nil
		}
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			e.CreatedAt = t
		} else {
			e.CreatedAt = time.Now()
		}
		if t, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
			e.ExpiresAt = t
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *sqliteStorage) Save(entries []MemoryEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Use INSERT OR REPLACE so each entry is its own atomic upsert.
	// The id column has a PRIMARY KEY constraint — INSERT OR REPLACE
	// deletes the old row and inserts the new one in a single step,
	// avoiding the DELETE-all-then-reinsert race window.
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO memories (id, session_id, content, tags, created_at, access_count, ttl_ns, expires_at, importance, vector, dense_vector) VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		tagsJSON, err := json.Marshal(e.Tags)
		if err != nil {
			tagsJSON = []byte("[]")
		}
		vectorJSON, err := json.Marshal(e.Vector)
		if err != nil {
			vectorJSON = []byte("{}")
		}
		denseVectorJSON, err := json.Marshal(e.DenseVector)
		if err != nil {
			denseVectorJSON = []byte("[]")
		}
		createdAt := e.CreatedAt.Format(time.RFC3339)
		expiresAt := ""
		if !e.ExpiresAt.IsZero() {
			expiresAt = e.ExpiresAt.Format(time.RFC3339)
		}
		if _, err := stmt.Exec(e.ID, e.SessionID, e.Content, string(tagsJSON),
			createdAt, e.AccessCount, int64(e.TTL), expiresAt, e.Importance, string(vectorJSON), string(denseVectorJSON)); err != nil {
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
		conditions = append(conditions, "(content LIKE ? ESCAPE '\\' OR tags LIKE ? ESCAPE '\\')")
		like := "%" + escapeLikeWildcards(query) + "%"
		args = append(args, like, like)
	}
	for _, tag := range tags {
		conditions = append(conditions, "tags LIKE ? ESCAPE '\\'")
		args = append(args, "%\""+escapeLikeWildcards(tag)+"\"%")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	q := fmt.Sprintf(`SELECT id, session_id, content, tags, created_at, access_count, ttl_ns, expires_at, importance, vector, dense_vector FROM memories %s ORDER BY importance DESC, created_at DESC LIMIT ?`, where)
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		var tagsJSON, createdAtStr, expiresAtStr, vectorJSON, denseVectorJSON string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Content, &tagsJSON,
			&createdAtStr, &e.AccessCount, &e.TTL, &expiresAtStr, &e.Importance, &vectorJSON, &denseVectorJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tagsJSON), &e.Tags); err != nil {
			e.Tags = nil
		}
		if err := json.Unmarshal([]byte(vectorJSON), &e.Vector); err != nil {
			e.Vector = nil
		}
		if err := json.Unmarshal([]byte(denseVectorJSON), &e.DenseVector); err != nil {
			e.DenseVector = nil
		}
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			e.CreatedAt = t
		} else {
			e.CreatedAt = time.Now()
		}
		if t, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
			e.ExpiresAt = t
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// escapeLikeWildcards escapes '%', '_', and '\' in a string so it is treated
// as a literal value when embedded in a LIKE pattern that uses '\' as the
// ESCAPE character.
func escapeLikeWildcards(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// Close releases the database connection.
func (s *sqliteStorage) Close() error {
	return s.db.Close()
}
