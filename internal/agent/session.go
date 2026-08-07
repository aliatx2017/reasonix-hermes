// Package agent wires a Provider, a tool Registry, and a Session into the
// harness loop that drives a coding task to completion.
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"reasonix/internal/fileutil"
	"reasonix/internal/provider"
)

// Session holds the conversation history for one task. The run loop (one turn at
// a time) is the only writer, but a frontend can read History/Save from another
// goroutine while a turn appends, so mu guards Messages. Direct Messages reads on
// the run-loop goroutine stay lock-free (serial with its own writes); cross-
// goroutine access goes through Snapshot.
//
// Aggregate session statistics (TokensIn, TokensOut, TurnCount, CacheHit,
// CacheMiss, Cost, Currency) are persisted as a sidecar <path>.sessionstats
// file so a session started in the CLI shows accurate stats when resumed in
// the desktop frontend.
type Session struct {
	mu             sync.RWMutex
	Messages       []provider.Message
	rewriteVersion int // bumped each time the log is rewritten (compact/fold)

	// Aggregate session statistics, persisted as a sidecar .meta file.
	// Set via SetMeta before Save; loaded via LoadMeta after LoadSession.
	// These are NOT reset on compaction (compaction reuses the same session
	// and the aggregate is per-session, not per-message).
	TokensIn  int     `json:"tokensIn"`
	TokensOut int     `json:"tokensOut"`
	TurnCount int     `json:"turnCount"`
	CacheHit  int     `json:"cacheHit"`
	CacheMiss int     `json:"cacheMiss"`
	Cost      float64 `json:"cost"`
	Currency  string  `json:"currency"`

	// normalizedDirty is set when LoadSession repaired the history on the way in
	// (empty tool-call names, dangling calls, truncated args, …). The repair
	// already lives in Messages, so the next Save persists it automatically as
	// part of the usual full rewrite; the flag exists for observability and to
	// let callers opt out of work that a dirty session would make redundant.
	normalizedDirty bool
}

// SessionMeta is the on-disk format for the .meta sidecar file.
type SessionMeta struct {
	TokensIn  int     `json:"tokensIn"`
	TokensOut int     `json:"tokensOut"`
	TurnCount int     `json:"turnCount"`
	CacheHit  int     `json:"cacheHit"`
	CacheMiss int     `json:"cacheMiss"`
	Cost      float64 `json:"cost"`
	Currency  string  `json:"currency"`
	SavedAt   string  `json:"savedAt"`
}

// NewSession initializes a session with an optional system prompt.
func NewSession(system string) *Session {
	s := &Session{}
	if system != "" {
		s.Messages = append(s.Messages, provider.Message{Role: provider.RoleSystem, Content: system})
	}
	return s
}

// Add appends a message.
func (s *Session) Add(m provider.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, m)
}

// Replace swaps the whole message log — used by compaction, which rewrites the
// middle of the history.
func (s *Session) Replace(msgs []provider.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = msgs
}

// Snapshot returns a copy of the messages, safe to read from another goroutine
// while a turn appends. Frontends (History, Save) use it instead of touching the
// live slice.
func (s *Session) Snapshot() []provider.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]provider.Message(nil), s.Messages...)
}

// RewriteVersion returns the current rewrite version.
func (s *Session) RewriteVersion() int { return s.rewriteVersion }

// IncrementRewrite bumps the rewrite version by 1.
func (s *Session) IncrementRewrite() { s.rewriteVersion++ }

// HasContent returns true when the session carries at least one user,
// assistant, or tool message — i.e. more than just a system prompt. An
// "empty" conversation that has never been used should not be persisted.
func (s *Session) HasContent() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.Messages {
		if m.Role != provider.RoleSystem {
			return true
		}
	}
	return false
}

// SetMeta copies aggregate session statistics from the caller into the Session,
// typically from Agent atomics before saving. Call once before SaveMeta to
// record the current cumulative totals.
func (s *Session) SetMeta(tokensIn, tokensOut, turnCount, cacheHit, cacheMiss int, cost float64, currency string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TokensIn = tokensIn
	s.TokensOut = tokensOut
	s.TurnCount = turnCount
	s.CacheHit = cacheHit
	s.CacheMiss = cacheMiss
	s.Cost = cost
	s.Currency = currency
}

// statsPath returns the sidecar aggregate-stats file path for a session JSONL,
// distinct from the .meta file used for branch navigation.
func statsPath(sessionPath string) string {
	return sessionPath + ".sessionstats"
}

// SaveMeta writes the session's aggregate statistics as a JSON sidecar at
// <path>.sessionstats. This is called alongside Save() so a session started in
// the CLI shows accurate cumulative stats when resumed in another frontend.
func (s *Session) SaveMeta(path string) error {
	if path == "" {
		return fmt.Errorf("empty session path for meta")
	}
	s.mu.RLock()
	meta := SessionMeta{
		TokensIn:  s.TokensIn,
		TokensOut: s.TokensOut,
		TurnCount: s.TurnCount,
		CacheHit:  s.CacheHit,
		CacheMiss: s.CacheMiss,
		Cost:      s.Cost,
		Currency:  s.Currency,
		SavedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create meta dir: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".session.*.meta.tmp")
	if err != nil {
		return fmt.Errorf("create meta tmp: %w", err)
	}
	tmpPath := tmp.Name()

	if err := json.NewEncoder(tmp).Encode(meta); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("encode meta: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return fileutil.ReplaceFile(tmpPath, statsPath(path))
}

// LoadMeta reads the sidecar .sessionstats file for a session. When the stats
// file doesn't exist (legacy session), it returns a zero-value SessionMeta with
// no error — callers merge what they can.
func LoadMeta(path string) (SessionMeta, error) {
	f, err := os.Open(statsPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return SessionMeta{}, nil
		}
		return SessionMeta{}, err
	}
	defer f.Close()

	var meta SessionMeta
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		return SessionMeta{}, fmt.Errorf("decode stats %s: %w", statsPath(path), err)
	}
	return meta, nil
}
