// Package agentlog provides a structured operational logger for agent observability.
// It replaces the default slog handler with one that writes JSON lines to a file
// (when AGENT_LOG is set or the default agent.log path is writable). If no file is
// available, output is discarded — it never writes to stderr/stdout to avoid TUI bleed.
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
	"io"
	"log/slog"
	"os"

	"reasonix/internal/config"
)

// Init replaces the default slog handler with one that writes JSON lines to a
// file (if AGENT_LOG is set or the default path is writable). It never writes to
// stderr — that would bleed JSON into the CLI TUI. If no file can be opened,
// slog output is discarded (a no-op discard handler) to keep the TUI clean.
func Init() {
	var w io.Writer

	if p := os.Getenv("AGENT_LOG"); p != "" {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err == nil {
			w = f
		}
	}
	if w == nil {
		// Try a default path; sandbox may block it — that's fine.
		dir := config.ReasonixHomeDir()
		if dir != "" {
			defaultPath := dir + "/agent.log"
			f, err := os.OpenFile(defaultPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
			if err == nil {
				w = f
			}
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
