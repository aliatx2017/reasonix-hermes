// Package agentlog provides a structured operational logger for agent observability.
// It replaces the default slog handler with one that writes JSON lines to a file
// (when AGENT_LOG is set or the default agent.log path is writable). If no file is
// available, output is discarded — it never writes to stderr/stdout to avoid TUI bleed.
//
// # Log rotation
//
// On Init, if the target log file exceeds cfg.MaxSizeMB (default 10), it is rotated:
// agent.log → agent.log.1, agent.log.1 → agent.log.2, … up to cfg.MaxBackups
// (default 5). The oldest backup (agent.log.N) is deleted. Rotation is self-contained
// — no external cron or logrotate needed.
//
// # Event contract
//
// Every event below MUST be logged exactly as specified. If code changes and a
// field is no longer populated, update the spec first — don't silently drop it.
//
//	agent.api_call      — every provider API round-trip (success or failure)
//	    model:       string   provider name
//	    in:          int      prompt tokens
//	    out:         int      completion tokens
//	    total:       int      total tokens
//	    cache_hit:   int      cache hit tokens (0 if provider doesn't report)
//	    cache_miss:  int      cache miss tokens
//	    latency_ms:  int64    wall-clock milliseconds (stream start to end)
//	    err:         string   (only on failure) provider error message
//	    cost:        float64  (only when pricing is configured) estimated USD cost
//
//	agent.tool_exec     — every tool execution (never skipped, even on error)
//	    tool:        string   tool name
//	    duration_ms: int64    wall-clock milliseconds
//	    result_bytes:int      output byte count
//	    success:     bool     false when errMsg is non-empty
//	    err:         string   (only on failure) short error reason
//	    truncated:   bool     (only when true) output was truncated
//
//	agent.compact       — session compaction (long-running sessions)
//	    ratio:       float64  current session size / context window
//	    messages:    int      messages before compaction
//	    kept:        int      messages kept after compaction
//
//	agent.turn          — turn boundary (one per user message)
//	    turn:        int      turn number for this session
//	    steps:       int      tool-call rounds this turn
//
//	boot.model           — model resolution
//	    ref:         string   configured model reference
//	    provider:    string   resolved provider kind
//
//	boot.mcp             — MCP server startup
//	    eager:       int      eager-start servers
//	    lazy:        int      lazy-start servers
//	    background:  int      background servers
//	    tools:       int      total tools registered
//
//	boot.config          — agent configuration
//	    max_steps:   int      configured max steps (0 = unbounded)
//	    temperature: float64  sampling temperature
//
//	boot.learner         — learner status
//	    enabled:     bool     whether learning is active
//
// # Field naming
//
// Use snake_case. Duration fields: latency_ms for API calls (network round-trip),
// duration_ms for tool execution (local work). All sizes in bytes. All times in
// milliseconds (int64).
//
// # Anti-patterns
//
// Never log to stderr or stdout — that bleeds JSON into the CLI TUI. Never skip
// a tool_exec or api_call — these are the primary telemetry for debugging agent
// behavior at 2am.
package agentlog

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"reasonix/internal/config"
)

const (
	defaultMaxSizeMB  = 10
	defaultMaxBackups = 5
)

// Init replaces the default slog handler with one that writes JSON lines to a
// file (if AGENT_LOG is set or the default path is writable). It never writes to
// stderr — that would bleed JSON into the CLI TUI. If no file can be opened,
// slog output is discarded (a no-op discard handler) to keep the TUI clean.
//
// Before opening, Init rotates the log file if it exceeds the configured size
// limit (see [agentlog] config section). Rotation renames agent.log → agent.log.1,
// shifts existing numbered backups, and deletes the oldest.
func Init(cfg config.AgentLogConfig) {
	if !cfg.Enabled && cfg != (config.AgentLogConfig{}) {
		slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})))
		return
	}

	var logPath string
	if p := os.Getenv("AGENT_LOG"); p != "" {
		logPath = p
	} else {
		dir := config.ReasonixHomeDir()
		if dir != "" {
			logPath = dir + "/agent.log"
		}
	}

	maxSize := cfg.MaxSizeMB
	if maxSize <= 0 {
		maxSize = defaultMaxSizeMB
	}
	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = defaultMaxBackups
	}

	// Rotate before opening — only when a path was resolved and the file exists
	// and exceeds the threshold.
	if logPath != "" {
		rotateLog(logPath, int64(maxSize)*1024*1024, maxBackups)
	}

	var w io.Writer
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err == nil {
			w = f
		}
	}
	if w == nil {
		// No writable log destination — discard all slog output so it
		// doesn't bleed into the TUI or desktop console.
		w = io.Discard
	}

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
}

// rotateLog renames logPath → logPath.1, .1 → .2, … up to maxBackups, deleting
// the oldest. If the file doesn't exist or is under maxBytes, this is a no-op.
func rotateLog(logPath string, maxBytes int64, maxBackups int) {
	info, err := os.Stat(logPath)
	if err != nil {
		return // file doesn't exist — nothing to rotate
	}
	if info.Size() < maxBytes {
		return // under threshold — keep appending
	}

	// Remove the oldest backup (logPath.N).
	oldest := logPath + "." + strconv.Itoa(maxBackups)
	os.Remove(oldest)

	// Shift existing backups: logPath.(N-1) → logPath.N, …, logPath.1 → logPath.2.
	for i := maxBackups - 1; i >= 1; i-- {
		old := logPath + "." + strconv.Itoa(i)
		new := logPath + "." + strconv.Itoa(i+1)
		_ = os.Rename(old, new)
	}

	// Rotate the current file: logPath → logPath.1.
	backup := logPath + "." + strconv.Itoa(1)
	if err := os.Rename(logPath, backup); err != nil {
		// If rename fails (e.g. cross-device), log via the old handler
		// (stderr, before we replaced it) and keep appending.
		fmt.Fprintf(os.Stderr, "agentlog: rotate %s → %s: %v\n", filepath.Base(logPath), filepath.Base(backup), err)
	}
}
