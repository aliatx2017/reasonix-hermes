package agent

import (
	"testing"

	"reasonix/internal/provider"
)

// realContract mirrors the shape compileExecutionContract emits: the
// source_event lives under planner_ir, and the block replaces the whole user
// turn.
func realContract(sourceEvent string) string {
	return "<memory-compiler-execution>\n" +
		`{"type":"memory_v5_execution_contract","instruction":"Execute source_event through planner_ir.",` +
		`"ir_explanation":{},"planner_ir":{"version":5,"goal":"g","source_event":` + jsonString(sourceEvent) + `}}` +
		"\n</memory-compiler-execution>"
}

func jsonString(s string) string {
	// minimal JSON string quoting for test fixtures (no control chars used here)
	return `"` + s + `"`
}

// TestStripTransientUserBlocksUnwrapsMemoryCompilerExecution guards the #5307
// contract: the Memory v5 <memory-compiler-execution> block REPLACES the user
// turn (the prompt survives only inside the contract's source_event), so the
// display/preview path must unwrap it to the original prompt — not drop it like
// a prepended transient block, which would blank out the turn.
func TestStripTransientUserBlocksUnwrapsMemoryCompilerExecution(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "block only (compiled contract replaced the whole turn)",
			in:   realContract("add a config loader"),
			want: "add a config loader",
		},
		{
			name: "language blocks before the compiler block",
			// Real composition order: withTurnPreferences wraps the compiled
			// contract, so the language blocks lead and the compiler block
			// follows. Both must resolve to the original prompt.
			in: "<reasoning-language>zh</reasoning-language>\n\n" +
				"<response-language>zh</response-language>\n\n" + realContract("do the thing"),
			want: "do the thing",
		},
		{
			name: "top-level source_event fallback shape",
			in:   "<memory-compiler-execution>\n{\"source_event\":\"older shape\"}\n</memory-compiler-execution>",
			want: "older shape",
		},
		{
			name: "unrecoverable contract falls back to empty",
			in:   "<memory-compiler-execution>\n{\"type\":\"memory_v5_execution_contract\"}\n</memory-compiler-execution>",
			want: "",
		},
		{
			name: "non-contract content is untouched",
			in:   "just a normal prompt",
			want: "just a normal prompt",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := StripTransientUserBlocks(tc.in); got != tc.want {
				t.Fatalf("StripTransientUserBlocks(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestUserPreviewTextPreservesCompiledTurnPrompt is the regression for the bot
// finding: a session whose first turn was compiled must still show the user's
// prompt in history/sidebar previews, not a blank line.
func TestUserPreviewTextPreservesCompiledTurnPrompt(t *testing.T) {
	in := realContract("ship the refactor")
	if got := UserPreviewText(in); got != "ship the refactor" {
		t.Fatalf("UserPreviewText = %q, want %q (compiled turn must not blank the preview)", got, "ship the refactor")
	}
}

// TestSessionPreviewFromMessagesPreservesCompiledFirstTurn proves the end-to-end
// preview path (used for the picker/sidebar) recovers the prompt when the first
// persisted user turn is a compiled contract.
func TestSessionPreviewFromMessagesPreservesCompiledFirstTurn(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: realContract("add pagination to the users endpoint")},
		{Role: provider.RoleAssistant, Content: "done"},
	}
	preview, turns := SessionPreviewFromMessages(msgs)
	if preview != "add pagination to the users endpoint" {
		t.Fatalf("preview = %q, want the compiled turn's source_event", preview)
	}
	if turns != 1 {
		t.Fatalf("user turns = %d, want 1", turns)
	}
}
