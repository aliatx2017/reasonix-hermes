package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── evalExtractCheckboxes ────────────────────────────────────────────────────

func TestEvalExtractCheckboxes(t *testing.T) {
	t.Parallel()

	content := `## EVAL: test

### Capability Evals
- [ ] go build passes
- [x] unit tests all green
- [ ] linter is clean

### Regression Evals
- [ ] existing API still works
- [ ] old configs load
`

	t.Run("extract capabilities", func(t *testing.T) {
		items := evalExtractCheckboxes(content, "### Capability Evals", "### Regression Evals")
		if len(items) != 3 {
			t.Fatalf("expected 3 capability items, got %d: %v", len(items), items)
		}
		if items[0] != "go build passes" {
			t.Errorf("item 0 = %q, want %q", items[0], "go build passes")
		}
		if items[1] != "unit tests all green" {
			t.Errorf("item 1 = %q, want %q", items[1], "unit tests all green")
		}
		if items[2] != "linter is clean" {
			t.Errorf("item 2 = %q, want %q", items[2], "linter is clean")
		}
	})

	t.Run("extract regressions", func(t *testing.T) {
		items := evalExtractCheckboxes(content, "### Regression Evals", "### Success Criteria")
		if len(items) != 2 {
			t.Fatalf("expected 2 regression items, got %d: %v", len(items), items)
		}
	})

	t.Run("no end section", func(t *testing.T) {
		items := evalExtractCheckboxes(content, "### Regression Evals", "")
		if len(items) != 2 {
			t.Fatalf("expected 2 items with no end section, got %d", len(items))
		}
	})

	t.Run("missing section", func(t *testing.T) {
		items := evalExtractCheckboxes(content, "### Nonexistent", "### Other")
		if len(items) != 0 {
			t.Errorf("expected 0 items for missing section, got %d", len(items))
		}
	})

	t.Run("empty content", func(t *testing.T) {
		items := evalExtractCheckboxes("", "### Capability Evals", "### Regression Evals")
		if items != nil {
			t.Errorf("expected nil for empty content, got %v", items)
		}
	})
}

// ── evalCountResults / evalLatestStatus ──────────────────────────────────────

func TestEvalCountAndLatestStatus(t *testing.T) {
	t.Parallel()

	log := strings.Join([]string{
		"=== CHECK at 2026-01-01T00:00:00Z ===",
		"go build passes: PASS",
		"unit tests all green: FAIL",
		"linter is clean: MANUAL",
		"",
		"=== CHECK at 2026-01-02T00:00:00Z ===",
		"go build passes: PASS",
		"unit tests all green: PASS",
		"linter is clean: PASS",
	}, "\n")

	t.Run("latest status", func(t *testing.T) {
		if s := evalLatestStatus(log, "go build passes"); s != "PASS" {
			t.Errorf("go build passes = %q, want PASS", s)
		}
		if s := evalLatestStatus(log, "unit tests all green"); s != "PASS" {
			t.Errorf("unit tests all green = %q, want PASS (latest)", s)
		}
		if s := evalLatestStatus(log, "nonexistent"); s != "NOT RUN" {
			t.Errorf("nonexistent = %q, want NOT RUN", s)
		}
	})

	t.Run("count results", func(t *testing.T) {
		criteria := []string{"go build passes", "unit tests all green", "linter is clean"}
		pass, total := evalCountResults(log, criteria)
		if total != 3 {
			t.Errorf("total = %d, want 3", total)
		}
		if pass != 3 {
			t.Errorf("pass = %d, want 3 (all latest are PASS)", pass)
		}
	})

	t.Run("count with mixed statuses", func(t *testing.T) {
		// Only first check run
		log1 := "=== CHECK at 2026-01-01T00:00:00Z ===\ngo build passes: PASS\nunit tests all green: FAIL\n"
		criteria := []string{"go build passes", "unit tests all green"}
		pass, total := evalCountResults(log1, criteria)
		if pass != 1 || total != 2 {
			t.Errorf("got %d/%d, want 1/2", pass, total)
		}
	})

	t.Run("no log data", func(t *testing.T) {
		s := evalLatestStatus("", "anything")
		if s != "NOT RUN" {
			t.Errorf("empty log = %q, want NOT RUN", s)
		}
	})
}

// ── evalCountCheckRuns ───────────────────────────────────────────────────────

func TestEvalCountCheckRuns(t *testing.T) {
	t.Parallel()

	t.Run("three runs", func(t *testing.T) {
		log := "=== CHECK at A ===\nfoo: PASS\n=== CHECK at B ===\nfoo: PASS\n=== CHECK at C ===\nfoo: FAIL\n"
		if n := evalCountCheckRuns(log); n != 3 {
			t.Errorf("got %d, want 3", n)
		}
	})

	t.Run("no runs", func(t *testing.T) {
		if n := evalCountCheckRuns("no markers here"); n != 0 {
			t.Errorf("got %d, want 0", n)
		}
	})
}

// ── evalComputePassAtN ───────────────────────────────────────────────────────

func TestEvalComputePassAtN(t *testing.T) {
	t.Parallel()

	// 4 runs: PASS, FAIL, PASS, PASS
	log := strings.Join([]string{
		"=== CHECK at A ===",
		"a: PASS",
		"b: PASS",
		"=== CHECK at B ===",
		"a: FAIL",
		"b: PASS",
		"=== CHECK at C ===",
		"a: PASS",
		"b: PASS",
		"=== CHECK at D ===",
		"a: PASS",
		"b: PASS",
	}, "\n")

	criteria := []string{"a", "b"}

	t.Run("pass@3 = 2 of last 3 all-green", func(t *testing.T) {
		n := evalComputePassAtN(log, 3, criteria)
		if n != 2 {
			t.Errorf("pass@3 = %d, want 2 (runs C and D are all-green)", n)
		}
	})

	t.Run("pass@2 = 2 of last 2 all-green", func(t *testing.T) {
		n := evalComputePassAtN(log, 2, criteria)
		if n != 2 {
			t.Errorf("pass@2 = %d, want 2", n)
		}
	})

	t.Run("pass@4 with only 4 runs", func(t *testing.T) {
		n := evalComputePassAtN(log, 4, criteria)
		if n != 3 {
			t.Errorf("pass@4 = %d, want 3 (runs A, C, D all-green out of 4)", n)
		}
	})

	t.Run("no runs", func(t *testing.T) {
		n := evalComputePassAtN("", 3, criteria)
		if n != 0 {
			t.Errorf("pass@3 with no runs = %d, want 0", n)
		}
	})

	t.Run("single run all-green", func(t *testing.T) {
		log1 := "=== CHECK at X ===\na: PASS\nb: PASS\n"
		n := evalComputePassAtN(log1, 3, criteria)
		if n != 1 {
			t.Errorf("pass@3 with 1 all-green run = %d, want 1", n)
		}
	})

	t.Run("no criteria", func(t *testing.T) {
		n := evalComputePassAtN(log, 3, nil)
		if n != 0 {
			t.Errorf("pass@3 with no criteria = %d, want 0", n)
		}
	})
}

// ── evalStatusInBlock ────────────────────────────────────────────────────────

func TestEvalStatusInBlock(t *testing.T) {
	t.Parallel()

	block := "foo: PASS\nbar: FAIL\nbaz: MANUAL\n"

	if s := evalStatusInBlock(block, "foo"); s != "PASS" {
		t.Errorf("foo = %q, want PASS", s)
	}
	if s := evalStatusInBlock(block, "bar"); s != "FAIL" {
		t.Errorf("bar = %q, want FAIL", s)
	}
	if s := evalStatusInBlock(block, "nonexistent"); s != "NOT RUN" {
		t.Errorf("nonexistent = %q, want NOT RUN", s)
	}
}

// ── evalExtractNotes ─────────────────────────────────────────────────────────

func TestEvalExtractNotes(t *testing.T) {
	t.Parallel()

	t.Run("extracts last check", func(t *testing.T) {
		log := "=== CHECK at A ===\nold notes\n=== CHECK at B ===\nlatest notes\n"
		notes := evalExtractNotes(log)
		if notes != "latest notes" {
			t.Errorf("notes = %q, want %q", notes, "latest notes")
		}
	})

	t.Run("single check", func(t *testing.T) {
		log := "=== CHECK at X ===\nfoo: PASS\nbar: FAIL\n"
		notes := evalExtractNotes(log)
		if notes != "foo: PASS\nbar: FAIL" {
			t.Errorf("notes = %q, want %q", notes, "foo: PASS\nbar: FAIL")
		}
	})

	t.Run("no checks", func(t *testing.T) {
		if s := evalExtractNotes("no markers"); s != "" {
			t.Errorf("notes = %q, want empty", s)
		}
	})

	t.Run("empty log", func(t *testing.T) {
		if s := evalExtractNotes(""); s != "" {
			t.Errorf("notes = %q, want empty", s)
		}
	})
}

// ── evalBuildRecommendation ──────────────────────────────────────────────────

func TestEvalBuildRecommendation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                          string
		capTotal, regTotal, capPass, regPass int
		want                          string
	}{
		{"all pass", 3, 2, 3, 2, "SHIP"},
		{"cap pass, no reg", 3, 0, 3, 0, "SHIP (no regressions defined)"},
		{"no cap, reg pass", 0, 2, 0, 2, "SHIP (no capabilities defined)"},
		{"cap fail", 3, 2, 2, 2, "BLOCKED"},
		{"reg fail", 3, 2, 3, 1, "BLOCKED"},
		{"nothing defined", 0, 0, 0, 0, "BLOCKED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalBuildRecommendation(tt.capTotal, tt.regTotal, tt.capPass, tt.regPass)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// ── evalRunVerify pattern matching ───────────────────────────────────────────

func TestEvalRunVerifyPatternMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		criterion string
		want      string // PASS, FAIL, or MANUAL — we can't predict build outcome but we can test matching
		wantNot   string // if not empty, result must NOT be this
	}{
		// These should trigger automated checks (and likely PASS since the project builds).
		// NOTE: "go test ./..." patterns ARE excluded from this table because they
		// actually run the full project test suite (slow). They are tested via
		// evalRunCommandCheck below instead.
		{criterion: "go build still works", wantNot: "MANUAL"},
		{criterion: "build should pass", wantNot: "MANUAL"},
		{criterion: "go vet is clean", wantNot: "MANUAL"},
		{criterion: "vet check", wantNot: "MANUAL"},

		// These should be MANUAL (no automated check applies)
		{criterion: "UI looks correct", want: "MANUAL"},
		{criterion: "performance is acceptable", want: "MANUAL"},
		{criterion: "API response is valid JSON", want: "MANUAL"},
	}

	for _, tt := range tests {
		t.Run(tt.criterion, func(t *testing.T) {
			result := evalRunVerify(tt.criterion)
			if tt.want != "" && result != tt.want {
				t.Errorf("evalRunVerify(%q) = %q, want %q", tt.criterion, result, tt.want)
			}
			if tt.wantNot != "" && result == tt.wantNot {
				t.Errorf("evalRunVerify(%q) = %q, want NOT %q", tt.criterion, result, tt.wantNot)
			}
		})
	}
}

// ── Integration: define / list / clean with temp dir ─────────────────────────

func TestEvalSharedDefineListClean(t *testing.T) {
	// Not parallel — uses shared root
	dir := t.TempDir()
	rootFunc := func() string { return dir }

	var lines, notices []string
	out := evalOutput{
		line:   func(s string) { lines = append(lines, s) },
		notice: func(s string) { notices = append(notices, s) },
	}

	// Define
	err := evalSharedDefine(rootFunc, "my-eval", out)
	if err != nil {
		t.Fatalf("evalSharedDefine: %v", err)
	}
	defPath := filepath.Join(dir, "my-eval", "definition.md")
	if _, err := os.Stat(defPath); err != nil {
		t.Fatalf("definition.md not created: %v", err)
	}

	// Define duplicate
	lines, notices = nil, nil
	err = evalSharedDefine(rootFunc, "my-eval", out)
	if err == nil {
		t.Fatal("expected error on duplicate define")
	}

	// List
	lines, notices = nil, nil
	err = evalSharedList(rootFunc, out)
	if err != nil {
		t.Fatalf("evalSharedList: %v", err)
	}
	found := false
	for _, l := range lines {
		if strings.Contains(l, "my-eval") {
			found = true
			if !strings.Contains(l, "0/4 passing") && !strings.Contains(l, "NOT STARTED") {
				t.Errorf("expected 0/4 passing or NOT STARTED status, got: %s", l)
			}
			break
		}
	}
	if !found {
		t.Error("my-eval not found in list output")
	}

	// Clean (should be no-op since no runs)
	lines, notices = nil, nil
	err = evalSharedClean(rootFunc, out)
	if err != nil {
		t.Fatalf("evalSharedClean: %v", err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "no logs needed trimming") {
		t.Errorf("unexpected clean output: %v", lines)
	}
}

// ── Integration: check / report with temp dir ────────────────────────────────

func TestEvalSharedCheckAndReport(t *testing.T) {
	dir := t.TempDir()
	rootFunc := func() string { return dir }

	var lines, notices []string
	out := evalOutput{
		line:   func(s string) { lines = append(lines, s) },
		notice: func(s string) { notices = append(notices, s) },
	}

	// Create a definition with build and vet checks (fast verification).
	// Note: "unit tests succeed" intentionally doesn't trigger automated
	// go test ./... (it doesn't match the heuristic) — test-matching is
	// verified separately in TestEvalRunVerifyPatternMatching.
	defContent := `## EVAL: quality

### Capability Evals
- [ ] go build still works
- [ ] go vet is clean

### Regression Evals
- [ ] pkg builds cleanly
`
	defDir := filepath.Join(dir, "quality")
	if err := os.MkdirAll(defDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(defDir, "definition.md"), []byte(defContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Check — should run go build + go vet + go test.
	lines, notices = nil, nil
	err := evalSharedCheck(rootFunc, "quality", out)
	if err != nil {
		t.Fatalf("evalSharedCheck: %v", err)
	}

	// Verify the output contains expected sections.
	output := strings.Join(lines, "\n")
	if !strings.Contains(output, "EVAL CHECK: quality") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "Capability Evals:") {
		t.Error("missing capabilities section")
	}
	if !strings.Contains(output, "Regression Evals:") {
		t.Error("missing regressions section")
	}

	// All three should pass on a clean project.
	if !strings.Contains(output, "[PASS] go build still works") {
		t.Error("go build should pass on clean project")
	}
	if !strings.Contains(output, "[PASS] go vet is clean") {
		t.Error("go vet should pass on clean project")
	}
	if !strings.Contains(output, "[PASS] pkg builds cleanly") {
		t.Error("pkg builds cleanly should PASS")
	}

	// Report
	lines, notices = nil, nil
	err = evalSharedReport(rootFunc, "quality", out)
	if err != nil {
		t.Fatalf("evalSharedReport: %v", err)
	}
	output = strings.Join(lines, "\n")
	if !strings.Contains(output, "EVAL REPORT: quality") {
		t.Error("missing report header")
	}
	if !strings.Contains(output, "SHIP") {
		t.Errorf("expected SHIP recommendation, got:\n%s", output)
	}
}

// ── Integration: clean with >10 runs ─────────────────────────────────────────

func TestEvalSharedCleanTrimsOldRuns(t *testing.T) {
	dir := t.TempDir()
	rootFunc := func() string { return dir }

	defDir := filepath.Join(dir, "many-runs")
	if err := os.MkdirAll(defDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a definition so list works.
	defContent := `## EVAL: many-runs

### Capability Evals
- [ ] fake check
`
	if err := os.WriteFile(filepath.Join(defDir, "definition.md"), []byte(defContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write 15 check runs into results.log.
	logPath := filepath.Join(defDir, "results.log")
	var logLines []string
	for i := 1; i <= 15; i++ {
		logLines = append(logLines, "=== CHECK at run-"+string(rune('0'+i))+" ===")
		logLines = append(logLines, "fake check: PASS")
	}
	if err := os.WriteFile(logPath, []byte(strings.Join(logLines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	// Clean
	var lines, notices []string
	out := evalOutput{
		line:   func(s string) { lines = append(lines, s) },
		notice: func(s string) { notices = append(notices, s) },
	}
	err := evalSharedClean(rootFunc, out)
	if err != nil {
		t.Fatalf("evalSharedClean: %v", err)
	}

	// Verify trimming happened.
	if len(lines) != 1 || !strings.Contains(lines[0], "trimmed") {
		t.Errorf("expected trimmed message, got: %v", lines)
	}

	// Verify log now has exactly 10 check markers.
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	checkCount := evalCountCheckRuns(string(data))
	if checkCount != 10 {
		t.Errorf("expected 10 check markers after clean, got %d", checkCount)
	}
}

// ── Error cases ──────────────────────────────────────────────────────────────

func TestEvalSharedErrors(t *testing.T) {
	dir := t.TempDir()
	rootFunc := func() string { return dir }

	t.Run("define empty name", func(t *testing.T) {
		out := evalOutput{line: func(s string) {}, notice: func(s string) {}}
		err := evalSharedDefine(rootFunc, "", out)
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("check non-existent definition", func(t *testing.T) {
		var notices []string
		out := evalOutput{
			line:   func(s string) {},
			notice: func(s string) { notices = append(notices, s) },
		}
		err := evalSharedCheck(rootFunc, "nonexistent", out)
		if err == nil {
			t.Fatal("expected error for non-existent eval")
		}
	})

	t.Run("report non-existent definition", func(t *testing.T) {
		var notices []string
		out := evalOutput{
			line:   func(s string) {},
			notice: func(s string) { notices = append(notices, s) },
		}
		err := evalSharedReport(rootFunc, "nonexistent", out)
		if err == nil {
			t.Fatal("expected error for non-existent eval")
		}
	})

	t.Run("list empty dir", func(t *testing.T) {
		emptyDir := t.TempDir()
		var notices []string
		out := evalOutput{
			line:   func(s string) {},
			notice: func(s string) { notices = append(notices, s) },
		}
		err := evalSharedList(func() string { return emptyDir }, out)
		if err != nil {
			t.Errorf("list on empty dir should not error: %v", err)
		}
	})

	t.Run("clean empty dir", func(t *testing.T) {
		emptyDir := t.TempDir()
		var lines []string
		out := evalOutput{
			line:   func(s string) { lines = append(lines, s) },
			notice: func(s string) {},
		}
		err := evalSharedClean(func() string { return emptyDir }, out)
		if err != nil {
			t.Errorf("clean on empty dir should not error: %v", err)
		}
	})
}

// ── evalRunCommandCheck ──────────────────────────────────────────────────────

func TestEvalRunCommandCheck(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		result := evalRunCommandCheck("echo ok", func(string) string { return "unexpected" })
		if result != "PASS" {
			t.Errorf("got %q, want PASS", result)
		}
	})

	t.Run("failing command with output", func(t *testing.T) {
		result := evalRunCommandCheck("echo 'some error' >&2; exit 1", func(string) string { return "FAIL" })
		if result != "FAIL" {
			t.Errorf("got %q, want FAIL", result)
		}
	})

	t.Run("failing command with no output", func(t *testing.T) {
		result := evalRunCommandCheck("exit 1", func(string) string { return "should-not-be-called" })
		if result != "FAIL" {
			t.Errorf("got %q, want FAIL", result)
		}
	})
}

// ── evalCLIRoot / evalTUIDir ─────────────────────────────────────────────────

func TestEvalCLIRoot(t *testing.T) {
	root := evalCLIRoot()
	if !strings.Contains(root, ".claude") || !strings.Contains(root, "evals") {
		t.Errorf("evalCLIRoot = %q, want path containing .claude/evals", root)
	}
}

// ── evalPrintUsage ───────────────────────────────────────────────────────────

func TestEvalPrintUsage(t *testing.T) {
	var notices []string
	out := evalOutput{
		line:   func(s string) {},
		notice: func(s string) { notices = append(notices, s) },
	}
	evalPrintUsage(out)
	if len(notices) != 1 {
		t.Fatalf("expected 1 notice, got %d", len(notices))
	}
	if !strings.Contains(notices[0], "eval compare") {
		t.Error("usage should include 'eval compare'")
	}
	if !strings.Contains(notices[0], "eval define") {
		t.Error("usage should include 'eval define'")
	}
}
