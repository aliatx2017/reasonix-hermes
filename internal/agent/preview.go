package agent

import (
	"encoding/json"
	"regexp"
	"strings"
)

var reTransientUserBlock = regexp.MustCompile(`(?s)^\s*<(?:response-language|reasoning-language|memory-update|background-jobs)>.*?</(?:response-language|reasoning-language|memory-update|background-jobs)>\s*\n?`)

var reMemoryCompilerExecution = regexp.MustCompile(`(?s)<memory-compiler-execution>\s*(.*?)\s*</memory-compiler-execution>`)

// StripTransientUserBlocks removes controller-injected transient XML blocks
// from persisted user messages before deriving display text, previews, or
// titles. The blocks are sent in user turns so they never affect the stable
// prompt prefix, but they should not become user-facing text later.
//
// The Memory v5 <memory-compiler-execution> block is handled differently from
// the prepended transient blocks: it does not prefix the user's prompt, it
// REPLACES the whole turn (Agent.Run swaps the compiled contract in for the
// original input, keeping the user's text only in the contract's source_event
// field). Dropping it like a prefix block would leave an empty string, so we
// unwrap it to the original prompt instead — otherwise sessions whose first
// turn was compiled would show a blank history/sidebar preview (#5307).
func StripTransientUserBlocks(content string) string {
	s := unwrapMemoryCompilerExecution(content)
	for {
		next := reTransientUserBlock.ReplaceAllStringFunc(s, func(string) string {
			return ""
		})
		if next == s {
			break
		}
		s = next
	}
	return strings.TrimLeft(s, " \t\r\n")
}

// unwrapMemoryCompilerExecution replaces a <memory-compiler-execution> contract
// with the user prompt it was compiled from (the contract's source_event), so
// display text and previews show what the user typed rather than the raw IR
// JSON or an empty string. Non-contract content is returned unchanged; a
// contract without a recoverable source_event collapses to empty, matching the
// prior "strip the block" behavior only as a last resort.
func unwrapMemoryCompilerExecution(content string) string {
	if !strings.Contains(content, "<memory-compiler-execution>") {
		return content
	}
	return reMemoryCompilerExecution.ReplaceAllStringFunc(content, func(block string) string {
		m := reMemoryCompilerExecution.FindStringSubmatch(block)
		if len(m) < 2 {
			return ""
		}
		return memoryCompilerSourceEvent(m[1])
	})
}

// memoryCompilerSourceEvent pulls the original user prompt out of a compiled
// execution contract's JSON body. The source_event lives under planner_ir; an
// older/looser shape may carry it at the top level, so both are checked.
// Returns "" when the body is not the expected JSON or carries no source_event.
func memoryCompilerSourceEvent(body string) string {
	var contract struct {
		SourceEvent string `json:"source_event"`
		PlannerIR   struct {
			SourceEvent string `json:"source_event"`
		} `json:"planner_ir"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(body)), &contract); err != nil {
		return ""
	}
	if s := strings.TrimSpace(contract.PlannerIR.SourceEvent); s != "" {
		return s
	}
	return strings.TrimSpace(contract.SourceEvent)
}

// UserPreviewText returns the user-authored part of a persisted user message.
func UserPreviewText(content string) string {
	s := StripTransientUserBlocks(content)
	s = HandoffTask(s)
	s = StripTransientUserBlocks(s)
	return strings.TrimSpace(s)
}
