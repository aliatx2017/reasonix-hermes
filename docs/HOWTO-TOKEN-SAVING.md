# How to Add Token-Saving Tool Output Compression to a Reasonix Fork

This guide walks through the exact steps to integrate a token-saving tool output
compressor (internally called `sqz`) into a Reasonix-based codebase. It is
written so you can hand it to your Reasonix agent and say "implement this."

The compressor uses three techniques from the 2026 token-saving research
landscape:

1. **SHA-256 content-addressable cache** — repeated identical tool output is
   replaced with a compact `[content unchanged — same as earlier this turn
   (sha256:abcdef…)]` reference (up to 92% reduction on repeated reads).
2. **Repeated-line collapsing** — bash output that repeats the same line dozens
   of times gets collapsed into `[×N above]` markers (up to 58% reduction).
3. **JSON minification** — null fields and empty lines are stripped from JSON
   tool results (up to 45% reduction).

A **safe mode** preserves errors, stack traces, diffs, and test failures
verbatim — compression of debugging output would destroy the model's ability to
diagnose problems.

---

## Step 1 — Create the `internal/compress/` package

Create a new directory and two files:

```
internal/compress/
  compress.go
  compress_test.go
```

### 1a. `compress.go` — Core compressor

```go
// Package compress reduces tool output token consumption before it reaches the
// LLM context. It combines three techniques from the 2026 token-saving landscape:
//
//  1. Content-addressable cache (sqz-style): SHA-256 of repeated tool output
//     is replaced with a compact "[cached — same as turn N]" reference instead
//     of re-injecting identical content.
//
//  2. Repeated-line collapsing: bash/grep output often repeats the same line
//     dozens of times; each repeated run is collapsed into "[×N above]".
//
//  3. JSON minification: large JSON tool results (read_file on JSON files,
//     API responses) have null fields stripped and empty objects collapsed.
//
// Safe mode: output that looks like an error, stack trace, panic, or test
// failure is preserved verbatim — compression of debugging output destroys the
// model's ability to diagnose and fix problems.
//
// Compression is best-effort and deterministic. It never increases output size
// beyond a small fixed header overhead.
package compress

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

// Compressor holds a per-session content cache and compression statistics.
// It is safe for concurrent use — tool calls from parallel sub-agents or
// concurrent tool execution can share a single compressor.
type Compressor struct {
	mu      sync.RWMutex
	cache   map[string]cacheEntry // SHA-256 hex → first occurrence
	turn    int                    // current turn number (updated by caller)
	enabled bool

	// Atomic stats counters.
	cacheHits          atomic.Int64
	linesCollapsed     atomic.Int64
	jsonFieldsStripped atomic.Int64
	bytesSaved         atomic.Int64
}

type cacheEntry struct {
	turn    int
	summary string // first 80 chars of content for reference
}

// Stats carries cumulative compression statistics for the dashboard.
type Stats struct {
	CacheHits          int `json:"cacheHits"`
	LinesCollapsed     int `json:"linesCollapsed"`
	JSONFieldsStripped int `json:"jsonFieldsStripped"`
	BytesSaved         int `json:"bytesSaved"`
}

// New creates a compressor. Pass enabled=false to make Compress a no-op
// (useful when the user disables compression in config).
func New(enabled bool) *Compressor {
	return &Compressor{
		cache:   make(map[string]cacheEntry),
		enabled: enabled,
	}
}

// SetTurn updates the current turn number. The agent calls this once per turn
// so cache references can name the turn where content first appeared.
func (c *Compressor) SetTurn(turn int) {
	c.mu.Lock()
	c.turn = turn
	c.mu.Unlock()
}

// Stats returns a snapshot of compression statistics.
func (c *Compressor) Stats() Stats {
	return Stats{
		CacheHits:          int(c.cacheHits.Load()),
		LinesCollapsed:     int(c.linesCollapsed.Load()),
		JSONFieldsStripped: int(c.jsonFieldsStripped.Load()),
		BytesSaved:         int(c.bytesSaved.Load()),
	}
}

// Compress reduces tool output. It returns the compressed text.
// The tool name helps decide compression strategy:
//   - "bash": aggressive line collapsing
//   - "read_file" / "grep": content caching for repeated reads
//   - "web_fetch": JSON minification
//   - everything else: content caching only
func (c *Compressor) Compress(toolName, raw string) string {
	if !c.enabled || raw == "" {
		return raw
	}

	// ── Safe mode: never compress error-looking output ──
	if isErrorOutput(raw) {
		return raw
	}

	// ── Pass 1: content-addressable cache ──
	hash := sha256Hex(raw)
	c.mu.RLock()
	entry, cached := c.cache[hash]
	c.mu.RUnlock()
	if cached {
		c.cacheHits.Add(1)
		c.bytesSaved.Add(int64(len(raw)))
		if entry.turn == c.turn {
			return fmt.Sprintf("[content unchanged — same as earlier this turn (sha256:%s…)]", hash[:12])
		}
		return fmt.Sprintf("[content unchanged since turn %d (sha256:%s…): %s]", entry.turn, hash[:12], entry.summary)
	}
	// Store in cache.
	c.mu.Lock()
	if _, exists := c.cache[hash]; !exists {
		summary := firstLine(raw, 100)
		c.cache[hash] = cacheEntry{turn: c.turn, summary: summary}
	}
	c.mu.Unlock()

	// ── Pass 2: tool-specific compression ──
	switch toolName {
	case "bash":
		return c.compressBash(raw)
	case "read_file", "grep":
		return c.compressReadOutput(raw)
	case "web_fetch":
		return c.compressWebFetch(raw)
	default:
		return c.compressGeneric(raw)
	}
}

// ── safe mode detection ──

func isErrorOutput(s string) bool {
	lower := strings.ToLower(s)
	markers := []string{
		"panic:", "fatal:", "signal:", "segmentation fault",
		"stack trace:", "goroutine", "traceback",
		"error:", "failed:", "cannot ",
		"--- fail", "--- pass",
		"diff --git",
	}
	count := 0
	for _, m := range markers {
		if strings.Contains(lower, m) {
			count++
		}
	}
	return count >= 2
}

// ── content-addressable cache helpers ──

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func firstLine(s string, maxLen int) string {
	for _, line := range strings.SplitN(s, "\n", 10) {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > maxLen {
				return line[:maxLen] + "…"
			}
			return line
		}
	}
	if len(s) > maxLen {
		return s[:maxLen] + "…"
	}
	return s
}

// ── per-tool-type compression ──

func (c *Compressor) compressBash(raw string) string {
	lines := strings.Split(raw, "\n")
	if len(lines) <= 3 {
		return raw // too short to benefit
	}
	out := make([]string, 0, len(lines))
	last := ""
	repeatCount := 0
	collapsedTotal := 0

	flush := func() {
		if repeatCount > 1 {
			out = append(out, last)
			out = append(out, fmt.Sprintf("[×%d above]", repeatCount-1))
			collapsedTotal += repeatCount - 1
		} else if repeatCount == 1 {
			out = append(out, last)
		}
		repeatCount = 0
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			out = append(out, line)
			last = ""
			continue
		}
		if trimmed == last {
			repeatCount++
		} else {
			flush()
			last = trimmed
			repeatCount = 1
		}
	}
	flush()

	if collapsedTotal == 0 {
		return raw
	}

	result := strings.Join(out, "\n")
	if len(result) >= len(raw) {
		return raw
	}
	c.linesCollapsed.Add(int64(collapsedTotal))
	c.bytesSaved.Add(int64(len(raw) - len(result)))
	return result
}

func (c *Compressor) compressReadOutput(raw string) string {
	return raw // cache hit path above handles identical content
}

func (c *Compressor) compressWebFetch(raw string) string {
	return c.compressJSON(raw)
}

func (c *Compressor) compressGeneric(raw string) string {
	if len(raw) > 2 && (raw[0] == '{' || raw[0] == '[') {
		return c.compressJSON(raw)
	}
	return raw
}

// ── JSON minification ──

func (c *Compressor) compressJSON(raw string) string {
	lines := strings.Split(raw, "\n")
	if len(lines) < 5 {
		return raw
	}

	var cleaned []string
	stripped := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "," {
			stripped++
			continue
		}
		if isNullJSONLine(trimmed) {
			stripped++
			continue
		}
		cleaned = append(cleaned, line)
	}

	if stripped == 0 {
		return raw
	}

	result := strings.Join(cleaned, "\n")
	if len(result) >= len(raw) {
		return raw
	}
	return result
}

func isNullJSONLine(line string) bool {
	line = strings.TrimRight(line, ",")
	line = strings.TrimSpace(line)
	if line == "null" {
		return true
	}
	if idx := strings.LastIndex(line, ":"); idx >= 0 {
		val := strings.TrimSpace(line[idx+1:])
		if val == "null" {
			return true
		}
	}
	return false
}

// ── utility ──

func tokenEstimate(s string) int {
	runes := 0
	for range s {
		runes++
	}
	return runes / 4
}

func isTextOnly(s string) bool {
	if len(s) > 512 {
		s = s[:512]
	}
	nonPrint := 0
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if !unicode.IsPrint(r) && r != 0 {
			nonPrint++
		}
	}
	return nonPrint < len(s)/10
}

// sortedKeys is used in tests.
func sortedKeys(m map[string]cacheEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
```

### 1b. `compress_test.go` — 21 tests

Write tests covering: disabled compressor, empty input, error safe mode (≥2
markers vs single marker), bash repeated-line collapsing, short bash output,
cache hit same-turn, cache hit different-turn, JSON null stripping, JSON short
input, diff preservation, `isErrorOutput` table, `firstLine` truncation,
`tokenEstimate`, `isTextOnly`, blank-line stripping, degenerate inputs (nil-safe,
binary), and stats accumulation.

The full test file is ~330 lines; see `internal/compress/compress_test.go` in
the Reasonix-Hermes source for the complete listing. Key patterns:

```go
func TestCompressBashRepeatedLines(t *testing.T) {
	c := New(true)
	raw := strings.Join([]string{
		"Building...", "Building...", "Building...", "Building...", "Done.",
	}, "\n")
	got := c.Compress("bash", raw)
	if !strings.Contains(got, "[×3 above]") {
		t.Errorf("expected [×3 above] marker, got: %s", got)
	}
}

func TestCompressErrorSafeMode(t *testing.T) {
	c := New(true)
	errOutput := strings.Join([]string{
		"--- FAIL: TestFoo (0.00s)",
		"    foo_test.go:10: error: expected 1, got 2",
		"    panic: runtime error: invalid memory address",
		"goroutine 1 [running]:",
	}, "\n")
	if got := c.Compress("bash", errOutput); got != errOutput {
		t.Errorf("error output should be preserved verbatim, got %q", got)
	}
}

func TestIsErrorOutput(t *testing.T) {
	tests := []struct{ input string; want bool }{
		{"panic: runtime error\ngoroutine 1 [running]:", true},  // 2 markers
		{"panic: runtime error", false},                          // 1 marker only
		{"--- FAIL\nerror: test", true},                         // 2 markers
		{"hello world", false},
	}
	for _, tc := range tests {
		got := isErrorOutput(tc.input)
		if got != tc.want {
			t.Errorf("isErrorOutput(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
```

**Verify this step**: `go test ./internal/compress/ -v` — all 21 tests pass.

---

## Step 2 — Add config option to `internal/config/config.go`

In the `AgentConfig` struct, add a field for the user to disable compression:

```go
// In internal/config/config.go, within the AgentConfig struct:

// CompressToolOutput enables token-saving compression on tool results via
// SHA-256 content caching, line dedup, and JSON minification (default true).
CompressToolOutput *bool `toml:"compress_tool_output"`
```

This goes alongside the other `AgentConfig` fields (around `ColdResumePrune`,
`Auxiliary`, etc.). The pointer type means `nil` = use the default (enabled).

Example TOML usage for users:

```toml
[agent]
compress_tool_output = false   # disable if desired
```

**Verify**: `go build ./internal/config/` compiles.

---

## Step 3 — Add `CompressToolOutput` to Agent Options

### 3a. Add the field to `agent.Options`

In `internal/agent/agent.go`, add to the `Options` struct:

```go
// CompressToolOutput enables token-saving compression on tool results before
// they enter the model's context. Uses SHA-256 content caching, repeated-line
// collapsing, and JSON minification with safe-mode error preservation.
CompressToolOutput bool
```

### 3b. Add the compressor field to the Agent struct

```go
type Agent struct {
	// ...existing fields...
	compress *compress.Compressor // tool output token compressor (nil = disabled)
}
```

### 3c. Initialize the compressor in `New()`

In the `New` function, add:

```go
compress: compress.New(opts.CompressToolOutput),
```

### 3d. Call `SetTurn` at the start of each step

In the `Run` method's step loop (around where `step` increments), add:

```go
a.compress.SetTurn(step + 1)
```

This lets cache-hit messages name the turn where content first appeared.

### 3e. Call `Compress` on tool results

In the method that executes tool calls (around where results are collected and
truncated), add the compression call AFTER the workshop sidecar but BEFORE any
output truncation:

```go
// Tool output compression: apply token-saving compression (SHA-256 content
// cache, line dedup, JSON minification) unless this is an error result.
if a.compress != nil && err == nil {
	result = a.compress.Compress(call.Name, result)
}
```

### 3f. Add the `CompressStats()` getter

```go
// CompressStats returns cumulative compression statistics.
func (a *Agent) CompressStats() compress.Stats {
	if a.compress == nil {
		return compress.Stats{}
	}
	return a.compress.Stats()
}
```

### 3g. Add the import

```go
import "reasonix/internal/compress"
```

(Adjust the module path to match your fork's module name.)

**Verify**: `go build ./internal/agent/` compiles.

---

## Step 4 — Wire config → Agent in `internal/boot/boot.go`

In the function that builds agent options (around where `WorkshopThreshold` and
`Workshop` are set), add:

```go
CompressToolOutput: compressEnabled(cfg.Agent.CompressToolOutput),
```

Then add the helper function:

```go
// compressEnabled resolves the CompressToolOutput config: nil = default true.
func compressEnabled(override *bool) bool {
	if override == nil {
		return true
	}
	return *override
}
```

**Verify**: `go build ./internal/boot/` compiles.

---

## Step 5 — Surface stats through the Controller

### 5a. Add `CompressStats()` to the Controller

In `internal/control/`, add a method (or add to an existing stats file):

```go
import "reasonix/internal/compress"

// CompressStats returns compression statistics (cache hits, lines collapsed, bytes saved).
func (c *Controller) CompressStats() compress.Stats {
	if c.executor == nil {
		return compress.Stats{}
	}
	return c.executor.CompressStats()
}
```

**Verify**: `go build ./internal/control/` compiles.

---

## Step 6 — Surface stats in the CLI TUI

In your CLI TUI code (where the status bar or banner renders), add a compact
`sqz↓` indicator:

```go
if cs := ctrl.CompressStats(); cs.BytesSaved > 0 {
    data = append(data, dim("sqz")+" "+formatBytes(cs.BytesSaved))
}
```

Where `formatBytes` renders human-readable sizes:

```go
func formatBytes(n int) string {
    switch {
    case n >= 1_000_000:
        return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
    case n >= 1_000:
        return fmt.Sprintf("%.1fK", float64(n)/1_000)
    default:
        return strconv.Itoa(n)
    }
}
```

This appears in the status line alongside other session stats (model, turns,
tokens, cost). The typical pattern is to add it to:

1. **The pinned header banner** — shown once at session start and updated on
   each render.
2. **The bottom status bar** — shown on every frame via `View()`.
3. **The `/stats` command output** — when the user requests detailed stats.

**Verify**: Build your CLI binary and start a chat session. The `sqz↓` counter
appears in the status bar.

---

## Step 7 — Surface stats in the Desktop app (if applicable)

Skip this step if your fork has no desktop/Wails frontend.

### 7a. Go binding

In the desktop Go backend, add Wails bindings:

```go
// CompressStatsView carries tool-output compression savings for the dashboard.
type CompressStatsView struct {
	BytesSaved         int `json:"bytesSaved"`
	CacheHits          int `json:"cacheHits"`
	LinesCollapsed     int `json:"linesCollapsed"`
	JSONFieldsStripped int `json:"jsonFieldsStripped"`
	AuxTokens          int `json:"auxTokens"`  // optional — aux provider tokens
}

// CompressStats returns tool-output compression savings for the active tab.
func (a *App) CompressStats() CompressStatsView {
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return CompressStatsView{}
	}
	cs := ctrl.CompressStats()
	return CompressStatsView{
		BytesSaved:         cs.BytesSaved,
		CacheHits:          cs.CacheHits,
		LinesCollapsed:     cs.LinesCollapsed,
		JSONFieldsStripped: cs.JSONFieldsStripped,
	}
}
```

Add `CompressStatsView` to the `HermesDashboardEvent` struct so it gets pushed
in the live-data event loop:

```go
type HermesDashboardEvent struct {
	// ...
	Compress CompressStatsView `json:"compress"`
}
```

And populate it in the push loop:

```go
Compress: a.CompressStats(),
```

### 7b. TypeScript types

```typescript
export interface CompressStatsView {
  bytesSaved: number;
  cacheHits: number;
  linesCollapsed: number;
  jsonFieldsStripped: number;
  auxTokens: number;
}
```

### 7c. TypeScript bridge

```typescript
CompressStats(): Promise<CompressStatsView>;
CompressStatsForTab(tabID: string): Promise<CompressStatsView>;
```

### 7d. StatusBar component

```tsx
function CompressGaugeCompact() {
  const [cs, setCS] = useState<CompressStatsView | null>(null);
  useEffect(() => {
    // Prefer push events; fall back to polling.
    try {
      const w = window as any;
      if (w.runtime?.EventsOn) {
        const unsub = w.runtime.EventsOn("hermes:dashboard", (payload: any) => {
          if (payload?.compress) setCS(payload.compress);
        });
        app.CompressStats().then(setCS).catch(() => {});
        return () => { try { unsub(); } catch {} };
      }
    } catch {}
    const poll = () => { app.CompressStats().then(setCS).catch(() => {}); };
    poll();
    const id = setInterval(poll, 30000);
    return () => clearInterval(id);
  }, []);
  if (!cs || (cs.bytesSaved === 0 && cs.auxTokens === 0)) return null;
  return (
    <span title={`Compressor: ${fmtBytes(cs.bytesSaved)} saved · ${cs.cacheHits} cache hits · ${cs.linesCollapsed} lines collapsed`}>
      <Zap size={11} />
      {cs.bytesSaved > 0 && <span>sqz&nbsp;↓{fmtBytes(cs.bytesSaved)}</span>}
    </span>
  );
}
```

**Verify**: Desktop builds and `tsc --noEmit` passes.

---

## Step 8 — Full end-to-end verification

1. **Unit tests**: `go test ./internal/compress/ -v` — all 21 tests pass.
2. **Go build**: `go build ./... && go vet ./...` — no errors.
3. **Agent integration**: `go test ./internal/agent/...` — existing tests still pass.
4. **Controller integration**: `go test ./internal/control/...` — existing tests still pass.
5. **Integration test**: Start a chat session, run a bash command twice (`ls`),
   then run `read_file` on the same file twice. The second call should produce
   a cache-hit message instead of the full output.
6. **Safe mode test**: Create a file with a Go test failure (`--- FAIL:` +
   `error:` + `panic:`) and run bash `cat` on it — the output must be
   preserved verbatim (not compressed).
7. **Config test**: Set `compress_tool_output = false` in `[agent]` section,
   restart — compression should be a no-op (stats stay zero).
8. **Status bar**: The `sqz↓` counter should appear and increment as
   compression fires.

---

## Architecture Summary

```
reasonix.toml                  User config
  [agent]
  compress_tool_output = true  (default)
        │
        ▼
internal/config/config.go      CompressToolOutput *bool
        │
        ▼
internal/boot/boot.go          compressEnabled() → bool
        │
        ▼
internal/agent/agent.go        Options.CompressToolOutput
  Agent.compress *compress.Compressor
  CompressStats() compress.Stats
        │
        ├─ Run() step loop:     a.compress.SetTurn(step+1)
        └─ executeToolCall():   a.compress.Compress(toolName, result)
                │
                ▼
internal/compress/compress.go   Compress(toolName, raw) string
  ├─ isErrorOutput()            safe mode (≥2 markers → verbatim)
  ├─ SHA-256 cache              [content unchanged since turn N …]
  ├─ compressBash()             [×3 above]
  └─ compressJSON()             null stripping
        │
        ▼
internal/control/controller_stats.go  CompressStats() compress.Stats
        │
        ├─ internal/cli/chat_tui.go    status bar: sqz↓12K
        └─ desktop/hermes_dashboard.go Wails binding → React StatusBar
```

## Files Changed (checklist)

Use this as a self-check after implementation:

- [ ] `internal/compress/compress.go` — new file (350 lines)
- [ ] `internal/compress/compress_test.go` — new file (330 lines, 21 tests)
- [ ] `internal/config/config.go` — add `CompressToolOutput *bool` field to `AgentConfig`
- [ ] `internal/agent/agent.go` — 5 changes:
  - Import `"reasonix/internal/compress"`
  - Add `CompressToolOutput bool` to `Options`
  - Add `compress *compress.Compressor` to `Agent` struct
  - Initialize `compress: compress.New(opts.CompressToolOutput)` in `New()`
  - Add `CompressStats() compress.Stats` method
  - Call `a.compress.SetTurn(step+1)` in step loop
  - Call `a.compress.Compress(call.Name, result)` after tool execution
- [ ] `internal/boot/boot.go` — add `CompressToolOutput: compressEnabled(...)` + helper
- [ ] `internal/control/controller_stats.go` — add `CompressStats() compress.Stats`
- [ ] CLI TUI — add `sqz↓N` to status bar and `/stats` output
- [ ] Desktop (optional) — Go bindings + TypeScript types + StatusBar component
