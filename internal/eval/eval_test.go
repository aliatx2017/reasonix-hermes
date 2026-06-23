package eval

import (
	"os"
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

// ── LoadSessionSnapshot ──────────────────────────────────────────────────────

func TestLoadSessionSnapshotValid(t *testing.T) {
	// Not parallel — writes temp files.
	dir := t.TempDir()
	sessionPath := dir + "/session.json"
	content := `[{"role":"user","content":"hello"},{"role":"assistant","content":"hi there","tool_calls":[{"name":"read","arguments":"{}"}]}]`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	snap, err := LoadSessionSnapshot(sessionPath)
	if err != nil {
		t.Fatalf("LoadSessionSnapshot: %v", err)
	}
	if snap.Path != sessionPath {
		t.Errorf("Path = %q, want %q", snap.Path, sessionPath)
	}
	if len(snap.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(snap.Turns))
	}
	if snap.Turns[0].UserPrompt != "hello" {
		t.Errorf("UserPrompt = %q, want %q", snap.Turns[0].UserPrompt, "hello")
	}
	if snap.Turns[0].Assistant != "hi there" {
		t.Errorf("Assistant = %q, want %q", snap.Turns[0].Assistant, "hi there")
	}
	if len(snap.Turns[0].ToolCalls) != 1 || snap.Turns[0].ToolCalls[0] != "read" {
		t.Errorf("ToolCalls = %v, want [read]", snap.Turns[0].ToolCalls)
	}
	if snap.Tools["read"] != 1 {
		t.Errorf("Tools[read] = %d, want 1", snap.Tools["read"])
	}
	if len(snap.ToolSeq) != 1 || snap.ToolSeq[0] != "read" {
		t.Errorf("ToolSeq = %v, want [read]", snap.ToolSeq)
	}
}

func TestLoadSessionSnapshotFileNotFound(t *testing.T) {
	_, err := LoadSessionSnapshot("/nonexistent/path/session.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadSessionSnapshotInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := dir + "/bad.json"
	if err := os.WriteFile(sessionPath, []byte(`{not json`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSessionSnapshot(sessionPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadSessionSnapshotWithMeta(t *testing.T) {
	dir := t.TempDir()
	sessionPath := dir + "/session.json"
	metaPath := dir + "/session.sessionstats"
	sessionContent := `[{"role":"user","content":"x"},{"role":"assistant","content":"y"}]`
	metaContent := `{"turnCount":1,"tokensIn":10,"tokensOut":5,"cost":0.001,"currency":"$"}`
	if err := os.WriteFile(sessionPath, []byte(sessionContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, []byte(metaContent), 0644); err != nil {
		t.Fatal(err)
	}

	snap, err := LoadSessionSnapshot(sessionPath)
	if err != nil {
		t.Fatalf("LoadSessionSnapshot: %v", err)
	}
	if snap.Meta.TurnCount != 1 {
		t.Errorf("TurnCount = %d, want 1", snap.Meta.TurnCount)
	}
	if snap.Meta.TokensIn != 10 {
		t.Errorf("TokensIn = %d, want 10", snap.Meta.TokensIn)
	}
	if snap.Meta.Cost != 0.001 {
		t.Errorf("Cost = %f, want 0.001", snap.Meta.Cost)
	}
	if snap.Meta.Currency != "$" {
		t.Errorf("Currency = %q, want $", snap.Meta.Currency)
	}
}

func TestLoadSessionSnapshotMultiTurn(t *testing.T) {
	dir := t.TempDir()
	sessionPath := dir + "/multi.json"
	content := `[
		{"role":"user","content":"turn1"},
		{"role":"assistant","content":"a1","tool_calls":[{"name":"read","arguments":"{}"}]},
		{"role":"user","content":"turn2"},
		{"role":"assistant","content":"a2","tool_calls":[{"name":"write","arguments":"{}"},{"name":"bash","arguments":"{}"}]}
	]`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	snap, err := LoadSessionSnapshot(sessionPath)
	if err != nil {
		t.Fatalf("LoadSessionSnapshot: %v", err)
	}
	if len(snap.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(snap.Turns))
	}
	if snap.Tools["read"] != 1 {
		t.Errorf("Tools[read] = %d, want 1", snap.Tools["read"])
	}
	if snap.Tools["write"] != 1 {
		t.Errorf("Tools[write] = %d, want 1", snap.Tools["write"])
	}
	if snap.Tools["bash"] != 1 {
		t.Errorf("Tools[bash] = %d, want 1", snap.Tools["bash"])
	}
}

func TestLoadSessionSnapshotWithMultipleToolCalls(t *testing.T) {
	dir := t.TempDir()
	sessionPath := dir + "/dup.json"
	content := `[
		{"role":"user","content":"x"},
		{"role":"assistant","content":"y","tool_calls":[{"name":"read","arguments":"{}"},{"name":"read","arguments":"{}"}]}
	]`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	snap, err := LoadSessionSnapshot(sessionPath)
	if err != nil {
		t.Fatalf("LoadSessionSnapshot: %v", err)
	}
	// Two read calls in one turn.
	if snap.Tools["read"] != 2 {
		t.Errorf("Tools[read] = %d, want 2", snap.Tools["read"])
	}
	if len(snap.ToolSeq) != 2 {
		t.Errorf("ToolSeq len = %d, want 2", len(snap.ToolSeq))
	}
}

func TestLoadSessionSnapshotMalformedMeta(t *testing.T) {
	dir := t.TempDir()
	sessionPath := dir + "/session.json"
	metaPath := dir + "/session.sessionstats"
	sessionContent := `[{"role":"user","content":"x"},{"role":"assistant","content":"y"}]`
	if err := os.WriteFile(sessionPath, []byte(sessionContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, []byte(`{bad meta`), 0644); err != nil {
		t.Fatal(err)
	}

	// Should succeed — malformed meta is silently ignored.
	snap, err := LoadSessionSnapshot(sessionPath)
	if err != nil {
		t.Fatalf("LoadSessionSnapshot should succeed even with malformed meta: %v", err)
	}
	if snap.Meta.TurnCount != 0 {
		t.Error("meta should be zero when malformed")
	}
}

// ── truncate ─────────────────────────────────────────────────────────────────

func TestTruncate(t *testing.T) {
	t.Parallel()
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("short string unchanged: got %q", got)
	}
	if got := truncate("hello world", 5); got != "he..." {
		t.Errorf("truncated: got %q, want %q", got, "he...")
	}
	if got := truncate("hello", 3); got != "hello" {
		t.Errorf("n < 4 returns full string: got %q", got)
	}
	if got := truncate("", 10); got != "" {
		t.Errorf("empty: got %q", got)
	}
}

// ── shortPath ────────────────────────────────────────────────────────────────

func TestShortPath(t *testing.T) {
	t.Parallel()
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}
	p := shortPath(home + "/Library/Application Support/reasonix/sessions/x.json")
	if !strings.HasPrefix(p, "~") {
		t.Errorf("should relativize HOME: got %q", p)
	}
}

func TestShortPathLong(t *testing.T) {
	t.Parallel()
	// Create a path longer than 60 chars.
	long := "/very/long/path/that/exceeds/sixty/characters/and/needs/truncation/at/the/end/session.json"
	p := shortPath(long)
	if len(p) > 60 {
		t.Errorf("long path should be truncated to <= 60 chars: len=%d", len(p))
	}
	if !strings.HasPrefix(p, "...") {
		t.Errorf("long path should start with ...: got %q", p)
	}
}

// ── stringSlicesEqual ────────────────────────────────────────────────────────

func TestStringSlicesEqual(t *testing.T) {
	t.Parallel()
	if !stringSlicesEqual(nil, nil) {
		t.Error("two nils should be equal")
	}
	if !stringSlicesEqual([]string{}, []string{}) {
		t.Error("two empties should be equal")
	}
	if stringSlicesEqual([]string{"a"}, []string{"b"}) {
		t.Error("different elements should not be equal")
	}
	if stringSlicesEqual([]string{"a"}, []string{"a", "b"}) {
		t.Error("different lengths should not be equal")
	}
	if stringSlicesEqual([]string{"a", "b"}, []string{"a", "c"}) {
		t.Error("mismatch at second element should not be equal")
	}
}

// ── setDiff multi-count ──────────────────────────────────────────────────────

func TestSetDiffMultiCount(t *testing.T) {
	t.Parallel()
	// A has 3 reads, B has 1 read → onlyA should have 2 reads.
	a := []string{"read", "read", "read"}
	b := []string{"read"}
	onlyA, onlyB := setDiff(a, b)
	if len(onlyA) != 2 {
		t.Errorf("onlyA = %v, want 2 reads", onlyA)
	}
	if len(onlyB) != 0 {
		t.Errorf("onlyB = %v, want empty", onlyB)
	}
}

func TestSetDiffReversed(t *testing.T) {
	t.Parallel()
	// B has more writes than A.
	a := []string{"write"}
	b := []string{"write", "write", "write"}
	onlyA, onlyB := setDiff(a, b)
	if len(onlyA) != 0 {
		t.Errorf("onlyA = %v, want empty", onlyA)
	}
	if len(onlyB) != 2 {
		t.Errorf("onlyB = %v, want 2 writes", onlyB)
	}
}

func TestSetDiffEmpty(t *testing.T) {
	t.Parallel()
	onlyA, onlyB := setDiff(nil, nil)
	if len(onlyA) != 0 || len(onlyB) != 0 {
		t.Errorf("empty diff should be empty: A=%v B=%v", onlyA, onlyB)
	}
}

// ── jaccard edge cases ───────────────────────────────────────────────────────

func TestJaccardIdentical(t *testing.T) {
	t.Parallel()
	seq := []string{"read", "write", "bash", "read"}
	if j := jaccard(seq, seq); j != 1.0 {
		t.Errorf("identical sequences: got %v, want 1.0", j)
	}
}

func TestJaccardCompletelyDifferent(t *testing.T) {
	t.Parallel()
	if j := jaccard([]string{"read"}, []string{"write"}); j != 0.0 {
		t.Errorf("disjoint: got %v, want 0.0", j)
	}
}

func TestJaccardOneEmpty(t *testing.T) {
	t.Parallel()
	if j := jaccard([]string{"read"}, nil); j != 0.0 {
		t.Errorf("one empty: got %v, want 0.0", j)
	}
}

// ── FormatText edge cases ────────────────────────────────────────────────────

func TestFormatTextNoCost(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{Path: "a.json"}
	b := &SessionSnapshot{Path: "b.json"}
	r := Compare(a, b)
	text := r.FormatText()
	// Cost section should be omitted when both costs are 0.
	if strings.Contains(text, "Cost:") {
		t.Error("cost section should be omitted when costs are 0")
	}
}

func TestFormatTextMatchedCount(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{
		Path: "a.json",
		Turns: []TurnSnapshot{
			{Index: 1, ToolCalls: []string{"read"}},
			{Index: 2, ToolCalls: []string{"write"}},
		},
	}
	b := &SessionSnapshot{
		Path: "b.json",
		Turns: []TurnSnapshot{
			{Index: 1, ToolCalls: []string{"read"}},
			{Index: 2, ToolCalls: []string{"bash"}},
		},
	}
	r := Compare(a, b)
	text := r.FormatText()
	if !strings.Contains(text, "Matched:") {
		t.Error("missing matched count")
	}
}

// ── Compare edge cases ───────────────────────────────────────────────────────

func TestCompareToolDiffsSorted(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{
		Path:  "a.json",
		Tools: map[string]int{"z": 1, "a": 2},
	}
	b := &SessionSnapshot{
		Path:  "b.json",
		Tools: map[string]int{"m": 1, "a": 2},
	}
	r := Compare(a, b)
	// Tools should be sorted alphabetically.
	if len(r.ToolDiffs) != 3 {
		t.Fatalf("expected 3 tool diffs, got %d", len(r.ToolDiffs))
	}
	if r.ToolDiffs[0].Name != "a" {
		t.Errorf("first tool should be 'a', got %q", r.ToolDiffs[0].Name)
	}
	if r.ToolDiffs[1].Name != "m" {
		t.Errorf("second tool should be 'm', got %q", r.ToolDiffs[1].Name)
	}
	if r.ToolDiffs[2].Name != "z" {
		t.Errorf("third tool should be 'z', got %q", r.ToolDiffs[2].Name)
	}
}

func TestCompareStatsDiff(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{
		Path: "a.json",
		Meta: agent.SessionMeta{TokensIn: 100, TokensOut: 50, TurnCount: 3, Cost: 0.01, Currency: "$"},
	}
	b := &SessionSnapshot{
		Path: "b.json",
		Meta: agent.SessionMeta{TokensIn: 200, TokensOut: 100, TurnCount: 5, Cost: 0.02, Currency: "$"},
	}
	r := Compare(a, b)
	if r.StatsDiff.TokensIn[0] != 100 || r.StatsDiff.TokensIn[1] != 200 {
		t.Errorf("TokensIn = %v, want [100, 200]", r.StatsDiff.TokensIn)
	}
	if r.StatsDiff.Turns[0] != 3 || r.StatsDiff.Turns[1] != 5 {
		t.Errorf("Turns = %v, want [3, 5]", r.StatsDiff.Turns)
	}
	if r.StatsDiff.Cost[0] != 0.01 || r.StatsDiff.Cost[1] != 0.02 {
		t.Errorf("Cost = %v, want [0.01, 0.02]", r.StatsDiff.Cost)
	}
}

func TestCompareTurnDiffsMissing(t *testing.T) {
	t.Parallel()
	a := &SessionSnapshot{
		Path: "a.json",
		Turns: []TurnSnapshot{
			{Index: 1, ToolCalls: []string{"read", "write", "grep"}},
		},
	}
	b := &SessionSnapshot{
		Path: "b.json",
		Turns: []TurnSnapshot{
			{Index: 1, ToolCalls: []string{"read", "bash", "grep"}},
		},
	}
	r := Compare(a, b)
	if len(r.TurnDiffs) != 1 {
		t.Fatalf("expected 1 turn diff, got %d", len(r.TurnDiffs))
	}
	td := r.TurnDiffs[0]
	if td.Match {
		t.Error("turns should not match")
	}
	if !containsString(td.MissingA, "write") {
		t.Errorf("missingA should contain 'write': %v", td.MissingA)
	}
	if !containsString(td.MissingB, "bash") {
		t.Errorf("missingB should contain 'bash': %v", td.MissingB)
	}
}

func TestCompareEmptyToolDiffsFiltered(t *testing.T) {
	t.Parallel()
	// Tools with 0 count and 0 delta should be filtered from FormatText output.
	a := &SessionSnapshot{Path: "a.json", Tools: map[string]int{}}
	b := &SessionSnapshot{Path: "b.json", Tools: map[string]int{}}
	// Even though we have empty tools maps, Compare won't produce any ToolDiffs
	// because the allTools set is empty. Let's verify.
	r := Compare(a, b)
	if len(r.ToolDiffs) != 0 {
		t.Errorf("expected 0 tool diffs for empty sessions, got %d", len(r.ToolDiffs))
	}
}
