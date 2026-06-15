package eval

import (
	"strings"
	"testing"

	"reasonix/internal/agent"
)

func TestSetDiff(t *testing.T) {
	t.Parallel()
	a, b := setDiff([]string{"read", "write", "read"}, []string{"read", "bash"})
	if len(a) != 2 {
		t.Errorf("onlyA len = %d, want 2", len(a))
	}
	if !containsString(a, "write") || !containsString(a, "read") {
		t.Errorf("onlyA = %v, want [write read]", a)
	}
	if len(b) != 1 || b[0] != "bash" {
		t.Errorf("onlyB = %v, want [bash]", b)
	}
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s { return true }
	}
	return false
}

func TestJaccard(t *testing.T) {
	t.Parallel()
	if j := jaccard(nil, nil); j != 1.0 {
		t.Errorf("empty = %v, want 1.0", j)
	}
	a := []string{"read", "write", "bash"}
	b := []string{"read", "bash", "grep"}
	j := jaccard(a, b)
	// intersection = read, bash (2), union = read, write, bash, grep (4)
	if j < 0.49 || j > 0.51 {
		t.Errorf("jaccard = %v, want ~0.5", j)
	}
}

func TestCompareEmpty(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{Path: "a.json"}
	b := &SessionSnapshot{Path: "b.json"}
	r := Compare(a, b)
	if r.Similarity != 1.0 {
		t.Errorf("similarity = %v, want 1.0", r.Similarity)
	}
}

func TestCompareIdentical(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{
		Path:  "a.json",
		Tools: map[string]int{"read": 2, "write": 1},
		ToolSeq: []string{"read", "read", "write"},
		Turns: []TurnSnapshot{
			{Index: 1, ToolCalls: []string{"read"}},
			{Index: 2, ToolCalls: []string{"read", "write"}},
		},
	}
	b := &SessionSnapshot{
		Path:  "b.json",
		Tools: map[string]int{"read": 2, "write": 1},
		ToolSeq: []string{"read", "read", "write"},
		Turns: []TurnSnapshot{
			{Index: 1, ToolCalls: []string{"read"}},
			{Index: 2, ToolCalls: []string{"read", "write"}},
		},
	}
	r := Compare(a, b)
	if r.Similarity != 1.0 {
		t.Errorf("similarity = %v, want 1.0", r.Similarity)
	}
	for _, td := range r.TurnDiffs {
		if !td.Match {
			t.Errorf("turn %d not matched", td.Index)
		}
	}
}

func TestCompareDifferent(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{
		Path:  "a.json",
		Tools: map[string]int{"read": 3},
		ToolSeq: []string{"read", "read", "read"},
		Turns: []TurnSnapshot{
			{Index: 1, ToolCalls: []string{"read"}},
		},
	}
	b := &SessionSnapshot{
		Path:  "b.json",
		Tools: map[string]int{"read": 1, "write": 2},
		ToolSeq: []string{"read", "write", "write"},
		Turns: []TurnSnapshot{
			{Index: 1, ToolCalls: []string{"read"}},
			{Index: 2, ToolCalls: []string{"write"}},
		},
	}
	r := Compare(a, b)
	if r.Similarity >= 1.0 {
		t.Errorf("expected similarity < 1.0, got %v", r.Similarity)
	}
	// TurnCount comes from meta, not auto-derived from Turns in Compare.
	_ = r.StatsDiff
	// Turn 1 should match (both read)
	if !r.TurnDiffs[0].Match {
		t.Errorf("turn 1 should match")
	}
	// Turn 2 only exists in B
	if len(r.TurnDiffs) != 2 {
		t.Errorf("expected 2 turn diffs, got %d", len(r.TurnDiffs))
	}
	if r.TurnDiffs[1].Index != 2 || r.TurnDiffs[1].Match {
		t.Errorf("turn 2 should not match")
	}
}

func TestFormatText(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{
		Path:  "/Users/test/a.json",
		Meta:  agent.SessionMeta{TurnCount: 3, TokensIn: 100, TokensOut: 50, Cost: 0.01, Currency: "¥"},
		Turns: []TurnSnapshot{{Index: 1, ToolCalls: []string{"read"}}},
		Tools: map[string]int{"read": 1},
	}
	b := &SessionSnapshot{
		Path:  "/Users/test/b.json",
		Meta:  agent.SessionMeta{TurnCount: 5, TokensIn: 200, TokensOut: 100, Cost: 0.02, Currency: "¥"},
		Turns: []TurnSnapshot{{Index: 1, ToolCalls: []string{"read", "write"}}},
		Tools: map[string]int{"read": 1, "write": 1},
	}
	r := Compare(a, b)
	text := r.FormatText()
	if text == "" {
		t.Error("FormatText returned empty")
	}
	if !strings.Contains(text, "Session Comparison") {
		t.Error("missing header")
	}
	if !strings.Contains(text, "Turns:") {
		t.Error("missing stats section")
	}
	if !strings.Contains(text, "Tool Usage") {
		t.Error("missing tools section")
	}
	if !strings.Contains(text, "Turn-by-Turn") {
		t.Error("missing turns section")
	}
	if !strings.Contains(text, "Similarity") {
		t.Error("missing similarity section")
	}
}
