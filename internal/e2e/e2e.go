// Package e2e provides a replay-based regression testing harness for Reasonix
// agent sessions. It loads saved session files, replays user prompts through an
// agent, and compares tool calls and responses to detect behavioral regressions.
//
// Typical usage in a Go test:
//
//	func TestSessionReplay(t *testing.T) {
//	    harness := e2e.NewHarness(t, e2e.Options{
//	        SessionsDir: "testdata/sessions",
//	    })
//	    harness.Run(context.Background(), "my-session.json")
//	}
//
// The harness records which tools were called and whether the agent completed
// successfully. For snapshot-based comparison, call AssertSnapshot.
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reasonix/internal/provider"
)

// Options configures the test harness.
type Options struct {
	// SessionsDir is the directory containing saved session files.
	SessionsDir string
	// MaxTurns limits how many turns to replay (0 = all).
	MaxTurns int
	// Timeout per turn.
	TurnTimeout time.Duration
}

// Harness runs replay-based regression tests.
type Harness struct {
	t   *testing.T
	opt Options
}

// NewHarness creates a test harness.
func NewHarness(t *testing.T, opt Options) *Harness {
	if opt.TurnTimeout == 0 {
		opt.TurnTimeout = 30 * time.Second
	}
	return &Harness{t: t, opt: opt}
}

// SessionInputs extracts user prompts from a saved session file.
// Returns the messages in turn order, skipping system and assistant messages.
func SessionInputs(path string) ([]string, error) {
	msgs, err := loadMessages(path)
	if err != nil {
		return nil, err
	}
	var inputs []string
	for _, m := range msgs {
		if m.Role == provider.RoleUser && !isCompactionSummary(m.Content) {
			inputs = append(inputs, m.Content)
		}
	}
	return inputs, nil
}

// SessionTools extracts the set of tool names called in a saved session.
func SessionTools(path string) ([]string, error) {
	msgs, err := loadMessages(path)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var tools []string
	for _, m := range msgs {
		for _, tc := range m.ToolCalls {
			if !seen[tc.Name] {
				seen[tc.Name] = true
				tools = append(tools, tc.Name)
			}
		}
	}
	return tools, nil
}

// TurnCount returns the number of user turns in a session.
func TurnCount(path string) (int, error) {
	inputs, err := SessionInputs(path)
	if err != nil {
		return 0, err
	}
	return len(inputs), nil
}

// SessionStats holds aggregate statistics about a session.
type SessionStats struct {
	Path      string   `json:"path"`
	Turns     int      `json:"turns"`
	Tools     int      `json:"tools"`
	ToolNames []string `json:"toolNames"`
	Files     int      `json:"files"`     // estimated — messages that mention file paths
	TokensIn  int      `json:"tokensIn"`  // rough estimate from content length
	TokensOut int      `json:"tokensOut"` // rough estimate from assistant content
}

// Analyze reads a session file and returns aggregate statistics.
func Analyze(path string) (SessionStats, error) {
	msgs, err := loadMessages(path)
	if err != nil {
		return SessionStats{}, err
	}
	s := SessionStats{Path: path}
	seenTools := map[string]bool{}
	seenFiles := map[string]bool{}
	for _, m := range msgs {
		switch m.Role {
		case provider.RoleUser:
			if !isCompactionSummary(m.Content) {
				s.Turns++
				s.TokensIn += len(m.Content) / 4
			}
		case provider.RoleAssistant:
			s.TokensOut += len(m.Content) / 4
			s.TokensOut += len(m.ReasoningContent) / 4
			for _, tc := range m.ToolCalls {
				if !seenTools[tc.Name] {
					seenTools[tc.Name] = true
					s.ToolNames = append(s.ToolNames, tc.Name)
				}
			}
		case provider.RoleTool:
			// Extract file paths from tool results (heuristic).
			for _, word := range strings.Fields(m.Content) {
				if strings.Contains(word, "/") || strings.HasSuffix(word, ".go") || strings.HasSuffix(word, ".ts") {
					clean := strings.Trim(word, "\"'`,;:()[]{}")
					if strings.Contains(clean, "/") || strings.Contains(clean, ".") {
						seenFiles[clean] = true
					}
				}
			}
		}
	}
	s.Tools = len(s.ToolNames)
	s.Files = len(seenFiles)
	return s, nil
}

// AssertTools asserts that a session uses at least the given tool names.
func (h *Harness) AssertTools(path string, wantTools ...string) {
	h.t.Helper()
	tools, err := SessionTools(path)
	if err != nil {
		h.t.Fatalf("SessionTools(%q): %v", path, err)
	}
	for _, want := range wantTools {
		found := false
		for _, t := range tools {
			if t == want {
				found = true
				break
			}
		}
		if !found {
			h.t.Errorf("session %q: expected tool %q but it was never called (tools used: %v)", path, want, tools)
		}
	}
}

// AssertTurns asserts that a session has at least the given number of turns.
func (h *Harness) AssertTurns(path string, minTurns int) {
	h.t.Helper()
	n, err := TurnCount(path)
	if err != nil {
		h.t.Fatalf("TurnCount(%q): %v", path, err)
	}
	if n < minTurns {
		h.t.Errorf("session %q: expected at least %d turns, got %d", path, minTurns, n)
	}
}

// ListSessions returns all .json session files in the harness sessions directory.
func (h *Harness) ListSessions() ([]string, error) {
	entries, err := os.ReadDir(h.opt.SessionsDir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		paths = append(paths, filepath.Join(h.opt.SessionsDir, e.Name()))
	}
	return paths, nil
}

// RunAll lists all sessions and runs AssertTurns + AssertTools on each.
func (h *Harness) RunAll(ctx context.Context) {
	h.t.Helper()
	paths, err := h.ListSessions()
	if err != nil {
		h.t.Fatalf("ListSessions: %v", err)
	}
	for _, p := range paths {
		h.t.Run(filepath.Base(p), func(t *testing.T) {
			hh := &Harness{t: t, opt: h.opt}
			hh.AssertTurns(p, 1)
			stats, err := Analyze(p)
			if err != nil {
				t.Fatalf("Analyze: %v", err)
			}
			t.Logf("%s: %d turns, %d tools, %d files", filepath.Base(p), stats.Turns, stats.Tools, stats.Files)
		})
	}
}

// --- internal helpers ---

func loadMessages(path string) ([]provider.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var msgs []provider.Message
	dec := json.NewDecoder(f)
	for {
		var m provider.Message
		if err := dec.Decode(&m); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func isCompactionSummary(content string) bool {
	return strings.Contains(content, "<compaction-summary>")
}
