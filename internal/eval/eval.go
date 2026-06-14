// Package eval provides session comparison and evaluation tools for
// Reasonix-Hermes. It loads saved session transcripts, compares them
// structurally (turns, tools, tokens, decisions), and produces a
// human-readable diff or a structured report for eval-driven development.
package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"reasonix/internal/agent"
	"reasonix/internal/provider"
)

// SessionSnapshot captures the comparable shape of a single agent session.
// It is derived from a loaded session file and its .meta sidecar.
type SessionSnapshot struct {
	Path     string `json:"path"`
	Meta     agent.SessionMeta
	Turns    []TurnSnapshot  `json:"turns"`
	// Tools maps tool name → call count across all turns.
	Tools   map[string]int `json:"tools"`
	// ToolSeq is the ordered sequence of tool calls across all turns.
	ToolSeq []string `json:"toolSeq"`
}

// TurnSnapshot summarizes one agent turn.
type TurnSnapshot struct {
	Index       int      `json:"index"`
	UserPrompt  string   `json:"userPrompt"`  // truncated
	Assistant   string   `json:"assistant"`   // truncated, first 200 chars
	ToolCalls   []string `json:"toolCalls"`   // tool names called this turn
}

// ComparisonResult holds the structured diff between two sessions.
type ComparisonResult struct {
	SessionA    string          `json:"sessionA"`
	SessionB    string          `json:"sessionB"`
	StatsDiff   StatsDiff       `json:"statsDiff"`
	ToolDiffs   []ToolDiff      `json:"toolDiffs"`
	TurnDiffs   []TurnDiff      `json:"turnDiffs"`
	Similarity  float64         `json:"similarity"` // 0.0–1.0 structural similarity
}

// StatsDiff compares aggregate session metrics.
type StatsDiff struct {
	TokensIn   [2]int     `json:"tokensIn"`
	TokensOut  [2]int     `json:"tokensOut"`
	Turns      [2]int     `json:"turns"`
	Cost       [2]float64 `json:"cost"`
	Currency   [2]string  `json:"currency"`
}

// ToolDiff compares one tool's usage across two sessions.
type ToolDiff struct {
	Name    string `json:"name"`
	CountA  int    `json:"countA"`
	CountB  int    `json:"countB"`
	Delta   int    `json:"delta"` // B - A
}

// TurnDiff compares paired turns by index. Extra turns in the longer
// session are listed as unpaired.
type TurnDiff struct {
	Index    int      `json:"index"`
	ToolsA   []string `json:"toolsA"`
	ToolsB   []string `json:"toolsB"`
	Match    bool     `json:"match"`    // same tools in same order
	MissingA []string `json:"missingA"` // in A but not B
	MissingB []string `json:"missingB"` // in B but not A
}

// LoadSessionSnapshot reads a session file and its .meta sidecar,
// returning a comparable snapshot.
func LoadSessionSnapshot(sessionPath string) (*SessionSnapshot, error) {
	// Load messages
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	var msgs []provider.Message
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	snap := &SessionSnapshot{
		Path:  sessionPath,
		Tools: make(map[string]int),
	}

	// Try to load .meta sidecar
	metaPath := strings.TrimSuffix(sessionPath, ".json") + ".sessionstats"
	if metaData, err := os.ReadFile(metaPath); err == nil {
		var meta agent.SessionMeta
		if json.Unmarshal(metaData, &meta) == nil {
			snap.Meta = meta
		}
	}

	// Parse turns: a "turn" starts with a user message and ends before
	// the next user message (or end of messages).
	var turn *TurnSnapshot
	turnIdx := 0
	for _, msg := range msgs {
		switch msg.Role {
		case provider.RoleUser:
			if turn != nil {
				snap.Turns = append(snap.Turns, *turn)
			}
			turnIdx++
			turn = &TurnSnapshot{
				Index:      turnIdx,
				UserPrompt: truncate(msg.Content, 200),
			}
		case provider.RoleAssistant:
			if turn != nil {
				turn.Assistant = truncate(msg.Content, 200)
				// Extract tool calls from the message
				for _, tc := range msg.ToolCalls {
					name := tc.Name
					turn.ToolCalls = append(turn.ToolCalls, name)
					snap.ToolSeq = append(snap.ToolSeq, name)
					snap.Tools[name]++
				}
			}
		}
	}
	if turn != nil {
		snap.Turns = append(snap.Turns, *turn)
	}

	return snap, nil
}

// Compare produces a structured diff between two session snapshots.
func Compare(a, b *SessionSnapshot) *ComparisonResult {
	r := &ComparisonResult{
		SessionA: a.Path,
		SessionB: b.Path,
		StatsDiff: StatsDiff{
			TokensIn:  [2]int{a.Meta.TokensIn, b.Meta.TokensIn},
			TokensOut: [2]int{a.Meta.TokensOut, b.Meta.TokensOut},
			Turns:     [2]int{a.Meta.TurnCount, b.Meta.TurnCount},
			Cost:      [2]float64{a.Meta.Cost, b.Meta.Cost},
			Currency:  [2]string{a.Meta.Currency, b.Meta.Currency},
		},
	}

	// Tool diffs
	allTools := make(map[string]struct{})
	for t := range a.Tools { allTools[t] = struct{}{} }
	for t := range b.Tools { allTools[t] = struct{}{} }
	for t := range allTools {
		r.ToolDiffs = append(r.ToolDiffs, ToolDiff{
			Name:   t,
			CountA: a.Tools[t],
			CountB: b.Tools[t],
			Delta:  b.Tools[t] - a.Tools[t],
		})
	}
	sort.Slice(r.ToolDiffs, func(i, j int) bool {
		return r.ToolDiffs[i].Name < r.ToolDiffs[j].Name
	})

	// Turn diffs
	maxTurns := len(a.Turns)
	if len(b.Turns) > maxTurns {
		maxTurns = len(b.Turns)
	}
	for i := 0; i < maxTurns; i++ {
		td := TurnDiff{Index: i + 1}
		if i < len(a.Turns) { td.ToolsA = a.Turns[i].ToolCalls }
		if i < len(b.Turns) { td.ToolsB = b.Turns[i].ToolCalls }
		td.Match = stringSlicesEqual(td.ToolsA, td.ToolsB)
		td.MissingA, td.MissingB = setDiff(td.ToolsA, td.ToolsB)
		r.TurnDiffs = append(r.TurnDiffs, td)
	}

	// Structural similarity: Jaccard index of tool sequences
	r.Similarity = jaccard(a.ToolSeq, b.ToolSeq)

	return r
}

// FormatText returns a human-readable text report.
func (r *ComparisonResult) FormatText() string {
	var b strings.Builder
	shortA := shortPath(r.SessionA)
	shortB := shortPath(r.SessionB)

	fmt.Fprintf(&b, "=== Session Comparison ===\n")
	fmt.Fprintf(&b, "A: %s\n", shortA)
	fmt.Fprintf(&b, "B: %s\n", shortB)
	fmt.Fprintf(&b, "\n--- Stats ---\n")
	sd := r.StatsDiff
	fmt.Fprintf(&b, "Turns:     %d vs %d  (Δ %+d)\n", sd.Turns[0], sd.Turns[1], sd.Turns[1]-sd.Turns[0])
	fmt.Fprintf(&b, "Tokens in: %d vs %d  (Δ %+d)\n", sd.TokensIn[0], sd.TokensIn[1], sd.TokensIn[1]-sd.TokensIn[0])
	fmt.Fprintf(&b, "Tokens out:%d vs %d  (Δ %+d)\n", sd.TokensOut[0], sd.TokensOut[1], sd.TokensOut[1]-sd.TokensOut[0])
	if sd.Cost[0] > 0 || sd.Cost[1] > 0 {
		fmt.Fprintf(&b, "Cost:      %.4f vs %.4f  (Δ %+.4f %s)\n", sd.Cost[0], sd.Cost[1], sd.Cost[1]-sd.Cost[0], sd.Currency[1])
	}

	fmt.Fprintf(&b, "\n--- Tool Usage ---\n")
	for _, td := range r.ToolDiffs {
		if td.Delta == 0 && td.CountA == 0 {
			continue
		}
		marker := ""
		if td.Delta != 0 {
			marker = fmt.Sprintf("  (Δ %+d)", td.Delta)
		}
		fmt.Fprintf(&b, "  %-20s %3d vs %3d%s\n", td.Name, td.CountA, td.CountB, marker)
	}

	fmt.Fprintf(&b, "\n--- Turn-by-Turn ---\n")
	matchCount := 0
	for _, td := range r.TurnDiffs {
		status := "✓"
		if !td.Match { status = "✗" }
		fmt.Fprintf(&b, "  Turn %2d %s\n", td.Index, status)
		if !td.Match {
			if len(td.MissingA) > 0 {
				fmt.Fprintf(&b, "        only in A: %s\n", strings.Join(td.MissingA, ", "))
			}
			if len(td.MissingB) > 0 {
				fmt.Fprintf(&b, "        only in B: %s\n", strings.Join(td.MissingB, ", "))
			}
		}
		if td.Match { matchCount++ }
	}
	fmt.Fprintf(&b, "\n  Matched: %d/%d turns\n", matchCount, len(r.TurnDiffs))

	fmt.Fprintf(&b, "\n--- Similarity ---\n")
	fmt.Fprintf(&b, "  Jaccard (tool seq): %.2f\n", r.Similarity)

	return b.String()
}

func truncate(s string, n int) string {
	if n < 4 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-3]) + "..."
}

func shortPath(p string) string {
	if strings.HasPrefix(p, os.Getenv("HOME")) {
		p = "~" + strings.TrimPrefix(p, os.Getenv("HOME"))
	}
	// Take last 60 chars
	if len(p) > 60 {
		p = "..." + p[len(p)-57:]
	}
	return p
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) { return false }
	for i := range a {
		if a[i] != b[i] { return false }
	}
	return true
}

func setDiff(a, b []string) (onlyA, onlyB []string) {
	am := make(map[string]int)
	for _, s := range a { am[s]++ }
	bm := make(map[string]int)
	for _, s := range b { bm[s]++ }
	for s, ca := range am {
		if cb, ok := bm[s]; ok {
			if ca > cb {
				for i := 0; i < ca-cb; i++ { onlyA = append(onlyA, s) }
			}
		} else {
			for i := 0; i < ca; i++ { onlyA = append(onlyA, s) }
		}
	}
	for s, cb := range bm {
		if ca, ok := am[s]; ok {
			if cb > ca {
				for i := 0; i < cb-ca; i++ { onlyB = append(onlyB, s) }
			}
		} else {
			for i := 0; i < cb; i++ { onlyB = append(onlyB, s) }
		}
	}
	return
}

func jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 { return 1.0 }
	am := make(map[string]struct{})
	for _, s := range a { am[s] = struct{}{} }
	bm := make(map[string]struct{})
	for _, s := range b { bm[s] = struct{}{} }
	intersection := 0
	for s := range am {
		if _, ok := bm[s]; ok { intersection++ }
	}
	union := len(am) + len(bm) - intersection
	if union == 0 { return 1.0 }
	return float64(intersection) / float64(union)
}
