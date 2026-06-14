package cli

import (
	"fmt"
	"os"

	"reasonix/internal/eval"
)

func evalCommand(args []string) int {
	if len(args) == 0 || args[0] != "compare" || len(args) < 3 {
		fmt.Println("reasonix eval — session evaluation and comparison tool")
		fmt.Println()
		fmt.Println("  Commands:")
		fmt.Println("    reasonix eval compare <session-a> <session-b>")
		fmt.Println("        Compare two saved session files structurally")
		fmt.Println("        (turns, tools, tokens, cost, similarity)")
		fmt.Println()
		return 0
	}

	sessionA := args[1]
	sessionB := args[2]

	a, err := eval.LoadSessionSnapshot(sessionA)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %s: %v\n", sessionA, err)
		return 1
	}
	b, err := eval.LoadSessionSnapshot(sessionB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %s: %v\n", sessionB, err)
		return 1
	}

	result := eval.Compare(a, b)
	fmt.Println(result.FormatText())
	return 0
}
