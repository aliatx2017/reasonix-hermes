// Package orchestrate provides multi-agent orchestration patterns for
// Reasonix-Hermes. It implements chain, pair, and CI-fix workflows that
// coordinate multiple agent turns in sequence or parallel via a simple
// turn function (RunTurn).
package orchestrate

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// TurnFunc runs a single agent turn: given a prompt, it returns the
// agent's response text. This is the minimal interface the orchestration
// package needs — any frontend (CLI, controller, API) can supply it.
type TurnFunc func(ctx context.Context, prompt string) (string, error)

// ChainResult holds the output of a sequential two-agent chain.
type ChainResult struct {
	FirstOutput  string `json:"firstOutput"`
	SecondOutput string `json:"secondOutput"`
}

// Chain runs two agent turns in sequence: the first analyzes the task,
// the second receives the analysis and acts on it (review-then-fix,
// plan-then-implement).
func Chain(ctx context.Context, runTurn TurnFunc, task string) (*ChainResult, error) {
	firstPrompt := fmt.Sprintf(
		"Analyze the following task and produce a concise summary of what needs to be done, "+
			"key risks, and recommended approach. Do NOT make changes — just analyze.\n\nTask: %s",
		task,
	)
	first, err := runTurn(ctx, firstPrompt)
	if err != nil {
		return nil, fmt.Errorf("chain phase 1 (analyze): %w", err)
	}

	secondPrompt := fmt.Sprintf(
		"Task: %s\n\nAnalysis from previous agent:\n%s\n\n"+
			"Now implement the task based on this analysis. Be thorough.",
		task, truncate(first, 4000),
	)
	second, err := runTurn(ctx, secondPrompt)
	if err != nil {
		return nil, fmt.Errorf("chain phase 2 (implement): %w", err)
	}
	return &ChainResult{FirstOutput: first, SecondOutput: second}, nil
}

// PairResult holds the output of a parallel reviewer + implementer pair.
type PairResult struct {
	Review string `json:"review"`
	Impl   string `json:"impl"`
	Merged string `json:"merged"`
}

// Pair runs a reviewer and an implementer in parallel, then merges their
// results via a third synthesis turn.
func Pair(ctx context.Context, runTurn TurnFunc, task string) (*PairResult, error) {
	var (
		review, impl string
		rErr, iErr   error
		wg           sync.WaitGroup
	)
	wg.Add(2)

	go func() {
		defer wg.Done()
		review, rErr = runTurn(ctx,
			fmt.Sprintf("Review this task for correctness, edge cases, and potential issues. "+
				"Do NOT implement — only review.\n\nTask: %s", task))
	}()
	go func() {
		defer wg.Done()
		impl, iErr = runTurn(ctx,
			fmt.Sprintf("Implement the following task directly — write code, don't explain. "+
				"Be thorough.\n\nTask: %s", task))
	}()
	wg.Wait()

	if rErr != nil {
		return nil, fmt.Errorf("pair reviewer: %w", rErr)
	}
	if iErr != nil {
		return nil, fmt.Errorf("pair implementer: %w", iErr)
	}

	merged, err := runTurn(ctx,
		fmt.Sprintf("Task: %s\n\nReview findings:\n%s\n\nImplementation:\n%s\n\n"+
			"Merge these into a final, production-quality solution. "+
			"Address the reviewer's concerns and use the implementation as a starting point.",
			task, truncate(review, 2000), truncate(impl, 2000)),
	)
	if err != nil {
		return nil, fmt.Errorf("pair merge: %w", err)
	}
	return &PairResult{Review: review, Impl: impl, Merged: merged}, nil
}

// CIFixResult holds the result of a CI-driven auto-fix workflow.
type CIFixResult struct {
	FailuresFound int            `json:"failuresFound"`
	Fixes         []CIFixAttempt `json:"fixes"`
	Summary       string         `json:"summary"`
}

// CIFixAttempt is one fix attempt for a specific CI failure.
type CIFixAttempt struct {
	Failure string `json:"failure"`
	Output  string `json:"output"`
	Success bool   `json:"success"`
}

// BashFunc runs a shell command and returns its combined stdout+stderr.
type BashFunc func(ctx context.Context, command string) (string, error)

// CIFix runs a CI command, parses failures, and spawns one fix turn per
// failing test/check. Each fix agent sees only its specific failure context.
func CIFix(ctx context.Context, runBash BashFunc, runTurn TurnFunc, ciCommand string) (*CIFixResult, error) {
	ciOutput, err := runBash(ctx, ciCommand)
	if err != nil {
		_ = err // expected: non-zero exit when tests fail, still parse output
	}
	failures := parseCIFailures(ciOutput)
	if len(failures) == 0 {
		return &CIFixResult{Summary: "No CI failures detected in output."}, nil
	}

	r := &CIFixResult{FailuresFound: len(failures), Fixes: make([]CIFixAttempt, len(failures))}

	var wg sync.WaitGroup
	for i, failure := range failures {
		wg.Add(1)
		go func(idx int, f string) {
			defer wg.Done()
			out, err := runTurn(ctx,
				fmt.Sprintf("The CI pipeline failed with this error. Fix the issue.\n\n"+
					"CI Failure:\n%s\n\nFix the code so this test passes. Minimal, targeted changes only.", f))
			r.Fixes[idx] = CIFixAttempt{
				Failure: truncate(f, 200),
				Output:  out,
				Success: err == nil,
			}
		}(i, failure)
	}
	wg.Wait()

	good := 0
	for _, a := range r.Fixes {
		if a.Success {
			good++
		}
	}
	r.Summary = fmt.Sprintf("%d CI failures found. %d/%d fix turns completed.", r.FailuresFound, good, len(r.Fixes))
	return r, nil
}

func parseCIFailures(output string) []string {
	if output == "" {
		return nil
	}
	var failures []string
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "fail") || strings.Contains(lower, "error:") ||
			strings.Contains(lower, "--- fail:") {
			start := i
			if start > 1 {
				start--
			}
			end := i + 4
			if end > len(lines) {
				end = len(lines)
			}
			failures = append(failures, strings.Join(lines[start:end], "\n"))
		}
	}
	// deduplicate
	seen := make(map[string]bool)
	var deduped []string
	for _, f := range failures {
		if !seen[f] {
			seen[f] = true
			deduped = append(deduped, f)
		}
	}
	return deduped
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
