// Package agentlog provides a structured operational logger for agent observability.
// It replaces the default slog handler with one that writes to stderr (and
// optionally a file). Set AGENT_LOG to a file path for persistent logging.
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

// Init replaces the default slog handler with one that writes JSON lines to
// stderr (and a file, if AGENT_LOG is set to a writable path). Call early in
// boot before any slog calls.
func Init() {
	var writers []io.Writer
	writers = append(writers, os.Stderr)

	if p := os.Getenv("AGENT_LOG"); p != "" {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err == nil {
			writers = append(writers, f)
		}
	} else {
		// Try a default path; sandbox may block it — that's fine.
		dir := config.ReasonixHomeDir()
		if dir != "" {
			defaultPath := dir + "/agent.log"
			f, err := os.OpenFile(defaultPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
			if err == nil {
				writers = append(writers, f)
			}
		}
	}

	var w io.Writer
	switch len(writers) {
	case 1:
		w = writers[0]
	default:
		w = io.MultiWriter(writers[0], writers[1])
	}

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
}
