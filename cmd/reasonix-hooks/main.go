// Reasonix Hook Runner — native Go replacement for retain-hook.sh and
// reflect-hook.sh. Reads the hook payload from stdin (JSON from the hook
// Runner), then POSTs retain/reflect requests to the Hindsight memory server.
//
// Usage:
//
//	reasonix-hooks retain   # PreToolUse hook → sends tool context to memory
//	reasonix-hooks reflect  # Stop hook → triggers session reflection
//
// Environment:
//
//	HINDSIGHT_URL   — memory server URL (default: http://localhost:8080)
//	HINDSIGHT_KEY   — Bearer token if MEMORY_API_KEY is set on the server
//	HINDSIGHT_TIMEOUT — HTTP timeout in seconds (default: 5)
//
// Exit codes: 0 on success, 0 on non-fatal errors (never blocks the agent).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"reasonix/internal/netclient"
)

// hookPayload mirrors the structure passed by the upstream hook.Runner.
type hookPayload struct {
	Event         string          `json:"event"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolResult    string          `json:"tool_result"`
	SessionID     string          `json:"session_id"`
	LastAssistant string          `json:"last_assistant"`
	Turn          int             `json:"turn"`
}

// noiseTools are tools whose invocation we skip retaining.
var noiseTools = map[string]bool{
	"read_file":  true,
	"write_file": true,
	"edit_file":  true,
	"bash":       true,
	"search":     true,
	"glob":       true,
	"":           true,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: reasonix-hooks <retain|reflect>")
		os.Exit(0)
	}
	action := os.Args[1]

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[reasonix-hooks] read stdin: %v\n", err)
		os.Exit(0)
	}

	var payload hookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		fmt.Fprintf(os.Stderr, "[reasonix-hooks] parse payload: %v\n", err)
		os.Exit(0)
	}

	url := env("HINDSIGHT_URL", "http://localhost:8080")
	key := os.Getenv("HINDSIGHT_KEY")
	timeout := envDuration("HINDSIGHT_TIMEOUT", 5*time.Second)

	switch action {
	case "retain":
		doRetain(url, key, timeout, payload)
	case "reflect":
		doReflect(url, key, timeout, payload)
	default:
		fmt.Fprintf(os.Stderr, "[reasonix-hooks] unknown action: %s\n", action)
		os.Exit(0)
	}
}

func doRetain(url, key string, timeout time.Duration, p hookPayload) {
	tool := p.ToolName
	if noiseTools[tool] {
		return // skip noise
	}

	content := tool
	if len(p.ToolInput) > 0 {
		content = fmt.Sprintf("%s: %s", tool, string(p.ToolInput))
	}

	req := jsonrpcRequest("hindsight_retain", map[string]any{
		"content": content,
		"tags":    []string{"tool_use", tool},
	})

	body, _ := json.Marshal(req)
	if err := postJSON(url, key, timeout, body); err != nil {
		fmt.Fprintf(os.Stderr, "[reasonix-hooks] retain: %v\n", err)
	}
}

func doReflect(url, key string, timeout time.Duration, p hookPayload) {
	session := p.SessionID
	if session == "" {
		session = "latest"
	}

	req := jsonrpcRequest("hindsight_reflect", map[string]any{
		"session_id": session,
		"query":      "session summary",
	})

	body, _ := json.Marshal(req)
	if err := postJSON(url, key, timeout, body); err != nil {
		fmt.Fprintf(os.Stderr, "[reasonix-hooks] reflect: %v\n", err)
	}
}

// jsonrpcRequest builds a JSON-RPC 2.0 tools/call request.
func jsonrpcRequest(toolName string, args map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	}
}

func postJSON(url, key string, timeout time.Duration, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(url, "/")+"/mcp", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := netclient.DefaultClient().Do(req)
	if err != nil {
		return fmt.Errorf("post to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if s := os.Getenv(key); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return def
}
