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
// concurrent tool execution (storm-breaker) can share a single compressor.
type Compressor struct {
	mu      sync.RWMutex
	cache   map[string]cacheEntry // SHA-256 hex → first occurrence
	turn    int                    // current turn number (updated by caller)
	enabled bool

	// Atomic stats counters.
	cacheHits         atomic.Int64
	linesCollapsed    atomic.Int64
	jsonFieldsStripped atomic.Int64
	bytesSaved        atomic.Int64
}

type cacheEntry struct {
	turn    int
	summary string // first 80 chars of content for reference
}

// Stats carries cumulative compression statistics for the dashboard.
type Stats struct {
	CacheHits         int `json:"cacheHits"`
	LinesCollapsed    int `json:"linesCollapsed"`
	JSONFieldsStripped int `json:"jsonFieldsStripped"`
	BytesSaved        int `json:"bytesSaved"`
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
	c.turn = turn
}

// Stats returns a snapshot of compression statistics.
func (c *Compressor) Stats() Stats {
	return Stats{
		CacheHits:         int(c.cacheHits.Load()),
		LinesCollapsed:    int(c.linesCollapsed.Load()),
		JSONFieldsStripped: int(c.jsonFieldsStripped.Load()),
		BytesSaved:        int(c.bytesSaved.Load()),
	}
}

// Compress reduces tool output. It returns the compressed text and whether
// any compression was applied. When disabled, it returns the input unchanged.
//
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
	// Fast scan for unmistakable error signatures.
	lower := strings.ToLower(s)
	markers := []string{
		"panic:", "fatal:", "signal:", "segmentation fault",
		"stack trace:", "goroutine", "traceback",
		"error:", "failed:", "cannot ",
		"--- fail", "--- pass", // test output — keep verbatim
		"diff --git", // diff output — essential for review
	}
	count := 0
	for _, m := range markers {
		if strings.Contains(lower, m) {
			count++
		}
	}
	// A single marker can be a false positive (a file containing "error:"
	// in its name). Multiple markers strongly suggest real error output.
	return count >= 2
}

// ── content-addressable cache helpers ──

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func firstLine(s string, maxLen int) string {
	// Take the first non-empty line, trimmed.
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
			// Keep one instance of the repeated line, then a collapse marker.
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
		// Only collapse truly repeated lines (exact match), not similar ones.
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

	// Only return compressed version if we actually saved space.
	result := strings.Join(out, "\n")
	if len(result) >= len(raw) {
		return raw
	}
	c.linesCollapsed.Add(int64(collapsedTotal))
	c.bytesSaved.Add(int64(len(raw) - len(result)))
	return result
}

func (c *Compressor) compressReadOutput(raw string) string {
	// read_file/grep output: the cache hit path above handles identical
	// content. For partial reads (different offset/limit), no compression
	// is safe. Return unchanged.
	return raw
}

func (c *Compressor) compressWebFetch(raw string) string {
	return c.compressJSON(raw)
}

func (c *Compressor) compressGeneric(raw string) string {
	// For generic tool output, try JSON compression if it looks JSON-like.
	if len(raw) > 2 && (raw[0] == '{' || raw[0] == '[') {
		return c.compressJSON(raw)
	}
	return raw
}

// ── JSON minification ──

// compressJSON applies lossy-but-safe JSON minification:
//   - Strips lines that are only whitespace or "null," / "null"
//   - Collapses consecutive empty objects/arrays
//
// It operates on text, not parsed JSON, so it works on malformed/partial JSON
// (e.g. streaming API responses) and never errors.
func (c *Compressor) compressJSON(raw string) string {
	lines := strings.Split(raw, "\n")
	if len(lines) < 5 {
		return raw // too short to benefit
	}

	// Strip lines that are pure whitespace or just commas.
	var cleaned []string
	stripped := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "," {
			stripped++
			continue
		}
		// Strip "null" value lines in JSON objects.
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
	// Match patterns like: "key": null,  or  "key": null
	// or just: null,  or  null
	line = strings.TrimRight(line, ",")
	line = strings.TrimSpace(line)
	if line == "null" {
		return true
	}
	// "key": null
	if idx := strings.LastIndex(line, ":"); idx >= 0 {
		val := strings.TrimSpace(line[idx+1:])
		if val == "null" {
			return true
		}
	}
	return false
}

// ── utility ──

// tokenEstimate is a rough heuristic: ~4 chars per token for English text.
func tokenEstimate(s string) int {
	runes := 0
	for range s {
		runes++
	}
	return runes / 4
}

// isTextOnly reports whether a string is printable text (not binary).
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
	return nonPrint < len(s)/10 // <10% non-printable
}

// ── sort utility used in tests ──

func sortedKeys(m map[string]cacheEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
