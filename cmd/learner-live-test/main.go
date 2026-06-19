package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reasonix/internal/boot"
)

func main() {
	root := findProjectRoot()
	if err := os.Chdir(root); err != nil {
		fmt.Fprintf(os.Stderr, "chdir: %v\n", err)
		os.Exit(1)
	}

	ctrl, err := boot.Build(context.Background(), boot.Options{
		WorkspaceRoot: root,
		Stderr:        io.Discard,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "boot.Build: %v\n", err)
		os.Exit(1)
	}
	defer ctrl.Close()

	lr := ctrl.Learner()
	if lr == nil {
		fmt.Fprintln(os.Stderr, "FAIL: learner not configured")
		os.Exit(1)
	}

	turns := []string{
		"Write the single line 'turn 1' to /tmp/learner-live.txt, then cat it.",
		"Append 'turn 2' to /tmp/learner-live.txt, then cat it.",
		"Append 'turn 3' to /tmp/learner-live.txt, then cat it.",
		"Append 'turn 4' to /tmp/learner-live.txt, then cat it.",
		"Append 'turn 5' to /tmp/learner-live.txt, then cat it.",
	}

	for i, prompt := range turns {
		fmt.Printf("Turn %d...\n", i+1)
		if err := ctrl.SendCtx(context.Background(), prompt); err != nil {
			fmt.Fprintf(os.Stderr, "turn %d error: %v\n", i+1, err)
			os.Exit(1)
		}
		for ctrl.Running() {
			time.Sleep(200 * time.Millisecond)
		}
		fmt.Printf("  done. (obs=%d)\n", len(lr.Observations()))
	}

	fmt.Println()

	obs := lr.Observations()
	fmt.Printf("Observations: %d\n", len(obs))
	for _, o := range obs {
		names := make([]string, len(o.ToolCalls))
		for j, tc := range o.ToolCalls {
			if tc.Success {
				names[j] = tc.Name + "✓"
			} else {
				names[j] = tc.Name + "✗"
			}
		}
		fmt.Printf("  T%d: tools=%v\n", o.Turn, strings.Join(names, ", "))
	}

	pats := lr.Patterns()
	fmt.Printf("\nPatterns detected: %d\n", len(pats))
	for _, p := range pats {
		fmt.Printf("  %s (c=%d) trigger=%q action=%q\n", p.Name, p.Confidence, p.Trigger, p.Action)
	}

	if len(pats) == 0 {
		fmt.Println("\nNo patterns — model varied tool calls across turns. Learner is working correctly.")
	} else {
		fmt.Println("\nPASS: learner detected patterns from real tool calls!")
	}

	os.Remove("/tmp/learner-live.txt")
}

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "reasonix.toml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Fprintln(os.Stderr, "cannot find reasonix.toml")
			os.Exit(1)
		}
		dir = parent
	}
}
