// Hindsight-Reasonix Memory MCP Server — provides cross-session persistent
// memory via MCP tools: hindsight_recall, hindsight_retain, hindsight_reflect.
//
// Usage:
//
//	go run ./cmd/reasonix-memoryserver [--http] [--port 8080]
//
// Can be connected to Reasonix as an MCP plugin (stdio or HTTP).
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"reasonix/pkg/mcputil"
)

const (
	maxContentLength = 32768 // 32KB max per memory entry

	// Default TTL for new memories: 90 days.
	defaultTTL = 90 * 24 * time.Hour

	// Importance decay factor per day (exponential). Memories lose ~1% importance
	// per day when not accessed.
	importanceDecayPerDay = 0.01

	// Importance boost on each recall.
	importanceBoostOnRecall = 0.05

	// Minimum importance below which expired memories are purged on tidy.
	minImportanceToKeep = 0.0
)

// MemoryEntry is a single stored memory.
type MemoryEntry struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	Content     string    `json:"content"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastDecayAt time.Time `json:"last_decay_at"` // last time importance decay was applied
	AccessCount int       `json:"access_count"`
	TTL         time.Duration `json:"ttl_ns"`       // nanoseconds; 0 = use default
	ExpiresAt   time.Time     `json:"expires_at"`   // computed as CreatedAt + TTL
	Importance  float64        `json:"importance"`   // 0.0–1.0, bumped on recall
	Vector      map[string]float64 `json:"vector,omitempty"` // TF vector for sparse semantic search
	DenseVector []float64     `json:"dense_vector,omitempty"` // dense embedding vector from API
}

// Expired reports whether the entry has passed its expiry and should be
// excluded from recall results.
func (e *MemoryEntry) Expired() bool { return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt) }

// Storage abstracts persistence for MemoryStore.
type Storage interface {
	Load() ([]MemoryEntry, error)
	Save(entries []MemoryEntry) error
}

// MemoryStore persists memories via a pluggable Storage backend.
type MemoryStore struct {
	mu      sync.RWMutex
	storage Storage
	dir     string // directory path (for diagnostics)
	entries []MemoryEntry
	nextID  atomic.Int64
	embed   *embeddingClient // optional dense embedding API client
	embedBatch int            // max facts per embedding API call
}

// Dir returns the storage directory path.
func (ms *MemoryStore) Dir() string { return ms.dir }

// ── Vector Search ──────────────────────────────────────────────────

// vectorize tokenizes text into a term-frequency vector.
// Stops words shorter than 3 chars and common English stop words.
func vectorize(text string) map[string]float64 {
	vec := make(map[string]float64)
	fields := strings.Fields(strings.ToLower(text))
	for _, f := range fields {
		f = strings.Trim(f, ".,;:!?\"'()[]{}<>-–—")
		if len(f) < 3 || stopWords[f] {
			continue
		}
		vec[f]++
	}
	return vec
}

// cosineSimilarity returns the cosine similarity between two TF vectors (0.0–1.0).
func cosineSimilarity(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var dot, sumSqA, sumSqB float64
	for word, aVal := range a {
		sumSqA += aVal * aVal
		if bVal, ok := b[word]; ok {
			dot += aVal * bVal
		}
	}
	for _, bVal := range b {
		sumSqB += bVal * bVal
	}
	if sumSqA == 0 || sumSqB == 0 {
		return 0
	}
	return dot / (math.Sqrt(sumSqA) * math.Sqrt(sumSqB))
}

// logger is the package-level structured logger.
var logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "are": true, "but": true,
	"not": true, "you": true, "all": true, "can": true, "had": true,
	"her": true, "was": true, "one": true, "our": true, "out": true,
	"has": true, "have": true, "this": true, "that": true, "with": true,
	"from": true, "they": true, "will": true, "would": true, "been": true,
	"were": true, "their": true, "there": true, "which": true, "about": true,
	"into": true, "than": true, "then": true, "them": true, "these": true,
	"some": true, "such": true, "each": true, "over": true, "only": true,
	"other": true, "more": true, "most": true, "also": true, "very": true,
	"just": true, "being": true, "does": true, "done": true, "doing": true,
}

// ── File Storage Backend ───────────────────────────────────────────

// fileStorage persists entries as a JSON file on disk.
type fileStorage struct {
	dir string
}

func newFileStorage(dir string) *fileStorage { return &fileStorage{dir: dir} }

func (fs *fileStorage) Load() ([]MemoryEntry, error) {
	path := filepath.Join(fs.dir, "memories.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []MemoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (fs *fileStorage) Save(entries []MemoryEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(fs.dir, "memories.json.tmp")
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(fs.dir, "memories.json"))
}

// ── MemoryStore ────────────────────────────────────────────────────

func NewMemoryStore(dir string) (*MemoryStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return NewMemoryStoreWithStorage(newFileStorage(dir), dir)
}

func NewMemoryStoreWithStorage(s Storage, dir string) (*MemoryStore, error) {
	ms := &MemoryStore{storage: s, dir: dir}
	if err := ms.load(); err != nil {
		// Start fresh if load fails (e.g. no file yet).
		ms.nextID.Store(1)
	}
	return ms, nil
}

// SetEmbedder wires the optional dense embedding client and batch size.
func (ms *MemoryStore) SetEmbedder(ec *embeddingClient, batchSize int) {
	ms.embed = ec
	ms.embedBatch = batchSize
}

// SearchDense returns memories ranked by cosine similarity of their dense
// vectors to the query vector. Facts without dense vectors are skipped.
func (ms *MemoryStore) SearchDense(query, sessionID string, limit int) ([]MemoryEntry, error) {
	if ms.embed == nil {
		return nil, fmt.Errorf("dense search requires an embedding provider (set EMBEDDING_PROVIDER)")
	}
	queryVec := ms.embed.embedOne(query)
	if queryVec == nil {
		return nil, fmt.Errorf("failed to embed query")
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	type scored struct {
		entry MemoryEntry
		score float64
	}
	var candidates []scored
	for i := range ms.entries {
		e := &ms.entries[i]
		if e.Expired() {
			continue
		}
		if len(e.DenseVector) == 0 {
			continue
		}
		if sessionID != "" && e.SessionID != sessionID {
			continue
		}
		sim := denseCosine(queryVec, e.DenseVector)
		if sim > 0.3 { // minimum similarity threshold
			candidates = append(candidates, scored{entry: *e, score: sim})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	results := make([]MemoryEntry, len(candidates))
	for i, c := range candidates {
		results[i] = c.entry
	}
	return results, nil
}

func (ms *MemoryStore) load() error {
	entries, err := ms.storage.Load()
	if err != nil {
		return err
	}
	ms.entries = entries

	// Find highest ID number for monotonic counter
	var maxID int64
	for _, e := range ms.entries {
		var n int64
		fmt.Sscanf(e.ID, "mem-%d-", &n)
		if n > maxID {
			maxID = n
		}
	}
	ms.nextID.Store(maxID + 1)
	return nil
}

func (ms *MemoryStore) save() error {
	return ms.storage.Save(ms.entries)
}

// Tidy purges expired entries that fall below the minimum importance threshold,
// then persists. Safe to call periodically.
func (ms *MemoryStore) Tidy() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	filtered := ms.entries[:0]
	for i := range ms.entries {
		e := &ms.entries[i]
		if e.Expired() && e.Importance <= minImportanceToKeep {
			continue // purge
		}
		// Apply daily decay to importance, using LastDecayAt to avoid
		// double-decay and preserving CreatedAt for display.
		decayAnchor := e.LastDecayAt
		if decayAnchor.IsZero() {
			decayAnchor = e.CreatedAt
		}
		if !decayAnchor.IsZero() {
			days := now.Sub(decayAnchor).Hours() / 24
			if days > 0 {
				e.Importance = max(0, e.Importance-importanceDecayPerDay*days)
				e.LastDecayAt = now
			}
		}
		filtered = append(filtered, *e)
	}
	ms.entries = filtered
	if err := ms.save(); err != nil {
		logger.Warn("tidy save failed", "err", err)
	}
}

func (ms *MemoryStore) Retain(sessionID, content string, tags []string) (*MemoryEntry, error) {
	// Validate content
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("content must not be empty")
	}
	if len(content) > maxContentLength {
		return nil, fmt.Errorf("content exceeds maximum length of %d bytes", maxContentLength)
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	id := ms.nextID.Add(1) - 1
	now := time.Now()
	ttl := defaultTTL
	entry := MemoryEntry{
		ID:         fmt.Sprintf("mem-%d-%d", id, now.Unix()),
		SessionID:  sessionID,
		Content:    content,
		Tags:       tags,
		CreatedAt:  now,
		TTL:        ttl,
		ExpiresAt:  now.Add(ttl),
		Importance: 0.5, // start at medium importance
		Vector:     vectorize(content),
	}

	// Auto-embed with dense vectors when the embedding client is configured.
	if ms.embed != nil {
		entry.DenseVector = ms.embed.embedOne(content)
	}

	ms.entries = append(ms.entries, entry)
	if err := ms.save(); err != nil {
		// Rollback
		ms.entries = ms.entries[:len(ms.entries)-1]
		return nil, fmt.Errorf("persist memory: %w", err)
	}
	return &entry, nil
}

func (ms *MemoryStore) Recall(sessionID, query string, limit int) ([]MemoryEntry, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var results []MemoryEntry
	lower := strings.ToLower(query)

	for i := range ms.entries {
		e := &ms.entries[i]

		// Skip expired entries
		if e.Expired() {
			continue
		}

		var match bool

		if query == "" && sessionID == "" {
			// No filters — return all (non-expired)
			match = true
		} else if query == "" && sessionID != "" {
			// Session filter only
			match = e.SessionID == sessionID
		} else {
			// Query-based matching (content + tags)
			match = strings.Contains(strings.ToLower(e.Content), lower)
			if !match {
				for _, tag := range e.Tags {
					if strings.Contains(strings.ToLower(tag), lower) {
						match = true
						break
					}
				}
			}
			// If sessionID also provided, require BOTH query match AND session match
			if match && sessionID != "" {
				match = e.SessionID == sessionID
			}
		}

		if match {
			results = append(results, *e)
		}
	}

	// Sort by creation time, newest first
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	// Increment access counts and boost importance (in-memory only; persisted on next Retain/Tidy).
	if len(results) > 0 {
		matched := make(map[string]bool, len(results))
		for _, r := range results {
			matched[r.ID] = true
		}
		for i := range ms.entries {
			if matched[ms.entries[i].ID] {
				ms.entries[i].AccessCount++
				// Boost importance (capped at 1.0); frequently recalled = longer effective TTL
				ms.entries[i].Importance = min(1.0, ms.entries[i].Importance+importanceBoostOnRecall)
				// Extend expiry: each recall pushes ExpiresAt forward by boost * defaultTTL
				boost := time.Duration(importanceBoostOnRecall * float64(defaultTTL))
				ms.entries[i].ExpiresAt = ms.entries[i].ExpiresAt.Add(boost)
			}
		}
	}

	return results, nil
}

// SearchSimilar returns memories ranked by cosine similarity of their TF vectors
// to the query vector. Falls back gracefully when entries lack vectors.
func (ms *MemoryStore) SearchSimilar(query string, sessionID string, limit int) ([]MemoryEntry, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	queryVec := vectorize(query)
	if len(queryVec) == 0 {
		return nil, nil
	}

	type scored struct {
		entry MemoryEntry
		score float64
	}
	var ranked []scored

	for i := range ms.entries {
		e := &ms.entries[i]
		if e.Expired() {
			continue
		}
		if sessionID != "" && e.SessionID != sessionID {
			continue
		}
		if len(e.Vector) == 0 {
			continue // skip entries without vectors (pre-vectorization data)
		}
		sim := cosineSimilarity(queryVec, e.Vector)
		if sim > 0 {
			ranked = append(ranked, scored{entry: *e, score: sim})
		}
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}

	results := make([]MemoryEntry, len(ranked))
	for i, s := range ranked {
		results[i] = s.entry
	}
	return results, nil
}

func (ms *MemoryStore) Reflect(sessionID string) string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var sessionMemories []MemoryEntry
	for _, e := range ms.entries {
		if e.SessionID == sessionID && !e.Expired() {
			sessionMemories = append(sessionMemories, e)
		}
	}

	if len(sessionMemories) == 0 {
		return "No memories found for this session."
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("# Session Reflection: %s\n\n", sessionID))
	out.WriteString(fmt.Sprintf("%d memories retained:\n\n", len(sessionMemories)))
	var denseCount int
	for _, e := range sessionMemories {
		out.WriteString(fmt.Sprintf("- [%s] %s\n", e.CreatedAt.Format("Jan 2 15:04"), truncateStr(e.Content, 100)))
		if len(e.DenseVector) > 0 {
			denseCount++
		}
	}
	out.WriteString(fmt.Sprintf("\n---\n%d/%d have dense embeddings.\n", denseCount, len(sessionMemories)))
	return out.String()
}

func truncateStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-3]) + "..."
}

// ── MCP Tool Definitions ──────────────────────────────────────────

func memoryTools() []mcputil.Tool {
	return []mcputil.Tool{
		{
			Name:        "hindsight_retain",
			Description: "Store a new memory fact for later recall. Use after important decisions or discoveries.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Current session identifier"},
					"content":    map[string]any{"type": "string", "description": "The memory content to store (max 32KB)"},
					"tags":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional tags for categorization"},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "hindsight_recall",
			Description: "Search and retrieve memories by keyword, session, tags, or semantic similarity.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Filter by session ID (combined with query if both set)"},
					"query":      map[string]any{"type": "string", "description": "Search keyword (empty = all). Use with semantic=true for meaning-based search."},
					"limit":      map[string]any{"type": "integer", "description": "Max results (default 10)"},
					"semantic":   map[string]any{"type": "boolean", "description": "Use TF-IDF vector similarity instead of keyword matching"},
					"dense":      map[string]any{"type": "boolean", "description": "Use dense embedding similarity (requires EMBEDDING_PROVIDER configured)"},
				},
			},
		},
		{
			Name:        "hindsight_reflect",
			Description: "Reflect on all memories from a session. Summarizes what was learned and retained.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Session to reflect on"},
				},
				"required": []string{"session_id"},
			},
		},
	}
}

// ── Handler ───────────────────────────────────────────────────────

type memoryHandler struct {
	store *MemoryStore
}

func (h *memoryHandler) handle(name string, args map[string]any) (string, error) {
	switch name {
	case "hindsight_retain":
		sessionID, _ := args["session_id"].(string)
		content, _ := args["content"].(string)
		var tags []string
		if t, ok := args["tags"].([]interface{}); ok {
			for _, tag := range t {
				if ts, ok := tag.(string); ok {
					tags = append(tags, ts)
				}
			}
		}
		entry, err := h.store.Retain(sessionID, content, tags)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Memory retained: %s", entry.ID), nil

	case "hindsight_recall":
		sessionID, _ := args["session_id"].(string)
		query, _ := args["query"].(string)
		limit := 10
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}
		semantic := false
		if s, ok := args["semantic"].(bool); ok {
			semantic = s
		}
		dense := false
		if d, ok := args["dense"].(bool); ok {
			dense = d
		}
		var entries []MemoryEntry
		var err error
		if dense && query != "" && h.store.embed != nil {
			entries, err = h.store.SearchDense(query, sessionID, limit)
		} else if semantic && query != "" {
			entries, err = h.store.SearchSimilar(query, sessionID, limit)
		} else {
			entries, err = h.store.Recall(sessionID, query, limit)
		}
		if err != nil {
			return "", err
		}
		if len(entries) == 0 {
			return "No matching memories found.", nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Found %d memories:\n\n", len(entries)))
		for _, e := range entries {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.ID, e.Content))
		}
		return sb.String(), nil

	case "hindsight_reflect":
		sessionID, _ := args["session_id"].(string)
		if sessionID == "" {
			return "", fmt.Errorf("session_id is required")
		}
		return h.store.Reflect(sessionID), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// ── Main ──────────────────────────────────────────────────────────

func main() {
	storeDir := ".reasonix/hindsight-memory"
	if os.Getenv("REASONIX_PORTABLE") != "" {
		if exe, err := os.Executable(); err == nil {
			storeDir = filepath.Join(filepath.Dir(exe), ".reasonix", "hindsight-memory")
		}
	} else if home, err := os.UserHomeDir(); err == nil {
		storeDir = filepath.Join(home, ".reasonix", "hindsight-memory")
	}

	// Parse flags: --backend file|sqlite, --http [--port N]
	backend := "sqlite"
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "--backend" && i+1 < len(os.Args) {
			backend = os.Args[i+1]
		}
	}

	var store *MemoryStore
	var err error
	var ss *sqliteStorage
	switch backend {
	case "sqlite":
		var sErr error
		ss, sErr = newSQLiteStorage(storeDir)
		if sErr != nil {
			logger.Error("failed to create sqlite storage", "err", sErr)
		os.Exit(1)
		}
		store, err = NewMemoryStoreWithStorage(ss, storeDir)
	default:
		store, err = NewMemoryStore(storeDir)
	}
	if err != nil {
		logger.Error("failed to create memory store", "err", err)
		os.Exit(1)
	}
	if ss != nil {
		defer ss.Close()
	}
	store.Tidy() // clean up expired entries on startup

	// Wire optional dense embedding client (reads EMBEDDING_PROVIDER / EMBEDDING_MODEL / EMBEDDING_API_KEY).
	ec := newEmbeddingClientFromEnv()
	if ec != nil {
		store.SetEmbedder(ec, 20)
		logger.Info("dense embedding configured", "provider", ec.baseURL, "model", ec.model)
	} else {
		logger.Info("dense embedding not configured (set EMBEDDING_PROVIDER to enable)")
	}

	h := &memoryHandler{store: store}

	srv := &mcputil.Server{
		Name:    "hindsight-reasonix",
		Version: "1.1.0",
		Tools:   memoryTools(),
		Handle:  h.handle,
	}

	if len(os.Args) > 1 && os.Args[1] == "--http" {
		port := "8080"
		if len(os.Args) > 3 && os.Args[2] == "--port" {
			port = os.Args[3]
		}

		// Run HTTP server in a goroutine so we can listen for signals.
		errCh := make(chan error, 1)
		go func() { errCh <- srv.ServeHTTP("127.0.0.1:"+port, "MEMORY_API_KEY") }()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		select {
		case err := <-errCh:
			logger.Error("http server error", "err", err)
			os.Exit(1)
		case sig := <-sigCh:
			logger.Info("shutting down", "signal", sig)
		}
		return
	}

	logger.Info("starting in stdio mode (MCP)")
	if err := srv.ServeStdio(); err != nil {
		logger.Error("stdio serve error", "err", err)
		os.Exit(1)
	}
}
