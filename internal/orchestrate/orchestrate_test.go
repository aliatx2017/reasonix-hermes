package orchestrate

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func TestChain(t *testing.T) {
	turn := 0
	runTurn := func(_ context.Context, prompt string) (string, error) {
		turn++
		if turn == 1 {
			if !strings.Contains(prompt, "Analyze") {
				t.Errorf("phase 1 should analyze, got: %s", prompt)
			}
			return "Analysis: needs refactoring", nil
		}
		if !strings.Contains(prompt, "implement") {
			t.Errorf("phase 2 should implement, got: %s", prompt)
		}
		return "Refactored code", nil
	}

	r, err := Chain(context.Background(), runTurn, "refactor auth module")
	if err != nil {
		t.Fatalf("Chain: %v", err)
	}
	if r.FirstOutput != "Analysis: needs refactoring" {
		t.Errorf("first = %q", r.FirstOutput)
	}
	if r.SecondOutput != "Refactored code" {
		t.Errorf("second = %q", r.SecondOutput)
	}
	if turn != 2 {
		t.Errorf("expected 2 turns, got %d", turn)
	}
}

func TestPair(t *testing.T) {
	callCount := 0
	var mu sync.Mutex
	runTurn := func(_ context.Context, prompt string) (string, error) {
		mu.Lock()
		callCount++
		c := callCount
		mu.Unlock()
		if strings.Contains(prompt, "Do NOT implement — only review") {
			return "Review: looks good", nil
		}
		if strings.Contains(prompt, "don't explain") {
			return "fn main() {}", nil
		}
		if strings.Contains(prompt, "Merge these") {
			_ = c
			return "Merged solution", nil
		}
		return "unknown", nil
	}

	r, err := Pair(context.Background(), runTurn, "write hello world")
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	if r.Review != "Review: looks good" {
		t.Errorf("review = %q", r.Review)
	}
	if r.Impl != "fn main() {}" {
		t.Errorf("impl = %q", r.Impl)
	}
	if r.Merged != "Merged solution" {
		t.Errorf("merged = %q", r.Merged)
	}
	if callCount != 3 {
		t.Errorf("expected 3 turn calls, got %d", callCount)
	}
}

func TestCIFix(t *testing.T) {
	runBash := func(_ context.Context, cmd string) (string, error) {
		return "--- FAIL: TestFoo\n--- FAIL: TestBar\n", nil
	}
	runTurn := func(_ context.Context, prompt string) (string, error) {
		return "fix applied", nil
	}

	r, err := CIFix(context.Background(), runBash, runTurn, "go test ./...")
	if err != nil {
		t.Fatalf("CIFix: %v", err)
	}
	if r.FailuresFound != 2 {
		t.Errorf("FailuresFound = %d, want 2", r.FailuresFound)
	}
	if len(r.Fixes) != 2 {
		t.Errorf("len(Fixes) = %d, want 2", len(r.Fixes))
	}
	for _, f := range r.Fixes {
		if !f.Success {
			t.Errorf("fix success = %v, want true", f.Success)
		}
	}
}

func TestCIFixNoFailures(t *testing.T) {
	runBash := func(_ context.Context, cmd string) (string, error) {
		return "ok\treasonix/internal/foo\t0.123s\n", nil
	}
	runTurn := func(_ context.Context, prompt string) (string, error) {
		return "", nil
	}

	r, err := CIFix(context.Background(), runBash, runTurn, "go test ./...")
	if err != nil {
		t.Fatalf("CIFix: %v", err)
	}
	if r.FailuresFound != 0 {
		t.Errorf("FailuresFound = %d, want 0", r.FailuresFound)
	}
	if !strings.Contains(r.Summary, "No CI failures") {
		t.Errorf("summary = %q", r.Summary)
	}
}

func TestParseCIFailures(t *testing.T) {
	input := "ok  test1\n--- FAIL: TestFoo (0.00s)\n    foo_test.go:10: expected 1\nFAIL\n--- FAIL: TestBar\n"
	f := parseCIFailures(input)
	if len(f) < 2 {
		t.Errorf("parseCIFailures = %d items, want >= 2", len(f))
	}
}

func TestParseCIFailuresEmpty(t *testing.T) {
	f := parseCIFailures("")
	if f != nil {
		t.Errorf("parseCIFailures('') = %v, want nil", f)
	}
	f = parseCIFailures("ok  test1\nok  test2\n")
	if f != nil {
		t.Errorf("parseCIFailures(all-pass) = %v, want nil", f)
	}
}
