// Package agentlog provides a structured operational logger for agent observability.
// It replaces the default slog handler with one that writes JSON lines to a file
// (when AGENT_LOG is set or the default agent.log path is writable). If no file is
// available, output is discarded — it never writes to stderr/stdout to avoid TUI bleed.
//
// Log categories:
//
//	agent.api_call          — per-API-call telemetry (model, tokens, latency)
//	agent.tool_exec         — tool execution (name, duration, result size)
//	boot                    — startup sequence (model, MCP, config, learner)
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
