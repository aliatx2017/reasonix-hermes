package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestSQLiteStorage_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}
	defer s.Close()

	entries := []MemoryEntry{
		{
			ID:          "e1",
			SessionID:   "sess1",
			Content:     "Go is fast",
			Tags:        []string{"go", "performance"},
			CreatedAt:   time.Now().UTC().Truncate(time.Second),
			AccessCount: 3,
			TTL:         24 * time.Hour,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second),
			Importance:  0.8,
			Vector:      map[string]float64{"go": 1.0, "fast": 1.0},
			DenseVector: []float64{0.1, 0.2, 0.3},
		},
		{
			ID:         "e2",
			SessionID:  "sess2",
			Content:    "Python is easy",
			Tags:       []string{"python"},
			CreatedAt:  time.Now().UTC().Truncate(time.Second),
			Importance: 0.5,
		},
	}

	if err := s.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded %d entries, want 2", len(loaded))
	}

	byID := make(map[string]MemoryEntry)
	for _, e := range loaded {
		byID[e.ID] = e
	}

	e1 := byID["e1"]
	if e1.Content != "Go is fast" {
		t.Errorf("e1 content = %q", e1.Content)
	}
	if len(e1.Tags) != 2 || e1.Tags[0] != "go" {
		t.Errorf("e1 tags = %v", e1.Tags)
	}
	if e1.AccessCount != 3 {
		t.Errorf("e1 access_count = %d, want 3", e1.AccessCount)
	}
	if len(e1.DenseVector) != 3 {
		t.Errorf("e1 dense_vector len = %d, want 3", len(e1.DenseVector))
	}
}

func TestSQLiteStorage_Upsert(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}
	defer s.Close()

	entry := MemoryEntry{
		ID: "u1", Content: "first", CreatedAt: time.Now().UTC().Truncate(time.Second), Importance: 0.3,
	}
	if err := s.Save([]MemoryEntry{entry}); err != nil {
		t.Fatalf("Save initial: %v", err)
	}

	entry.Content = "updated"
	entry.Importance = 0.9
	if err := s.Save([]MemoryEntry{entry}); err != nil {
		t.Fatalf("Save upsert: %v", err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry after upsert, got %d", len(loaded))
	}
	if loaded[0].Content != "updated" {
		t.Errorf("upserted content = %q, want 'updated'", loaded[0].Content)
	}
}

func TestSQLiteStorage_Search(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}
	defer s.Close()

	entries := []MemoryEntry{
		{ID: "s1", SessionID: "sess", Content: "Go concurrency with goroutines", Tags: []string{"go"}, CreatedAt: time.Now().UTC().Truncate(time.Second), Importance: 0.7},
		{ID: "s2", SessionID: "sess", Content: "Python async IO", Tags: []string{"python"}, CreatedAt: time.Now().UTC().Truncate(time.Second), Importance: 0.5},
		{ID: "s3", SessionID: "other", Content: "Go benchmarks", Tags: []string{"go"}, CreatedAt: time.Now().UTC().Truncate(time.Second), Importance: 0.6},
	}
	if err := s.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Search by content keyword
	results, err := s.Search("goroutines", "", nil, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].ID != "s1" {
		t.Errorf("Search 'goroutines' = %v, want [s1]", idsOf(results))
	}

	// Filter by session ID
	results, err = s.Search("", "sess", nil, 10)
	if err != nil {
		t.Fatalf("Search session: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search session = %d results, want 2", len(results))
	}

	// Filter by tag
	results, err = s.Search("", "", []string{"python"}, 10)
	if err != nil {
		t.Fatalf("Search tag: %v", err)
	}
	if len(results) != 1 || results[0].ID != "s2" {
		t.Errorf("Search tag python = %v, want [s2]", idsOf(results))
	}

	// No filter = all results up to limit
	results, err = s.Search("", "", nil, 10)
	if err != nil {
		t.Fatalf("Search all: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Search all = %d results, want 3", len(results))
	}
}

func TestSQLiteStorage_SearchLikeWildcards(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}
	defer s.Close()

	// Entry content with SQL LIKE wildcard characters.
	entry := MemoryEntry{
		ID: "w1", Content: "50% done", Tags: []string{"progress_report"},
		CreatedAt: time.Now().UTC().Truncate(time.Second), Importance: 0.5,
	}
	if err := s.Save([]MemoryEntry{entry}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	results, err := s.Search("50%", "", nil, 10)
	if err != nil {
		t.Fatalf("Search %%: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search '50%%' = %d results, want 1", len(results))
	}

	results, err = s.Search("", "", []string{"progress_report"}, 10)
	if err != nil {
		t.Fatalf("Search _ tag: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search '_' tag = %d results, want 1", len(results))
	}
}

func TestSQLiteStorage_EmptyLoad(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}
	defer s.Close()

	entries, err := s.Load()
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestSQLiteStorage_SaveEmptySlice(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}
	defer s.Close()

	if err := s.Save(nil); err != nil {
		t.Errorf("Save nil: %v", err)
	}
	if err := s.Save([]MemoryEntry{}); err != nil {
		t.Errorf("Save empty: %v", err)
	}
}

func TestSQLiteStorage_SearchLimit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}
	defer s.Close()

	var entries []MemoryEntry
	for i := 0; i < 5; i++ {
		entries = append(entries, MemoryEntry{
			ID: fmt.Sprintf("lim%d", i), Content: "item", CreatedAt: time.Now().UTC().Truncate(time.Second), Importance: float64(i) * 0.1,
		})
	}
	if err := s.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	results, err := s.Search("", "", nil, 3)
	if err != nil {
		t.Fatalf("Search limit: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("limited search = %d results, want 3", len(results))
	}
}

func idsOf(entries []MemoryEntry) []string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return ids
}

func TestSQLiteStorage_InvalidDir(t *testing.T) {
	t.Parallel()
	// Pass a path whose parent is a file (not a dir) to trigger MkdirAll failure.
	// Create a file at /tmp/testfile, then try to use it as a parent dir.
	f, err := os.CreateTemp("", "notadir")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	_, err = newSQLiteStorage(f.Name() + "/subdir")
	if err == nil {
		t.Error("expected error when creating storage in a file path")
	}
}

func TestSQLiteStorage_LoadBadJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}

	// Insert a row with invalid JSON in tags and vector columns via raw SQL.
	_, err = s.db.Exec(`INSERT INTO memories (id, session_id, content, tags, created_at, access_count, ttl_ns, expires_at, importance, vector, dense_vector) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"bad1", "", "bad json entry", "not-json", "2024-01-01T00:00:00Z", 0, 0, "", 0.5, "not-json-either", "also-bad",
	)
	if err != nil {
		t.Fatalf("insert bad json row: %v", err)
	}
	s.Close()

	// Re-open and load — should not panic; bad JSON is gracefully handled.
	s2, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	entries, err := s2.Load()
	if err != nil {
		t.Fatalf("Load with bad JSON: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// Tags/Vector should be nil when JSON is invalid.
	if entries[0].Tags != nil {
		t.Errorf("tags should be nil for invalid JSON, got %v", entries[0].Tags)
	}
}

func TestSQLiteStorage_LoadBadTimestamp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := newSQLiteStorage(dir)
	if err != nil {
		t.Fatalf("newSQLiteStorage: %v", err)
	}
	_, err = s.db.Exec(`INSERT INTO memories (id, session_id, content, tags, created_at, access_count, ttl_ns, expires_at, importance, vector, dense_vector) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"ts1", "", "content", "[]", "not-a-time", 0, 0, "also-not-a-time", 0.5, "{}", "",
	)
	if err != nil {
		t.Fatalf("insert bad timestamp row: %v", err)
	}
	defer s.Close()
	entries, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1, got %d", len(entries))
	}
	// Bad created_at should fall back to time.Now() (not zero).
	if entries[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero for bad timestamp")
	}
	// Bad expires_at with non-empty string stays zero.
	if !entries[0].ExpiresAt.IsZero() {
		t.Errorf("ExpiresAt should be zero for bad timestamp, got %v", entries[0].ExpiresAt)
	}
}
