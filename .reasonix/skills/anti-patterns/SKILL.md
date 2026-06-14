---
name: anti-patterns
description: "Technical anti-patterns and tool defaults to avoid — crop vs. generate, delegating research-heavy tasks to subagents, premature abstraction, and other patterns that waste tokens or produce wrong results."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [anti-patterns, best-practices, code-quality, agent-behavior, pitfalls]
---

# Anti-Patterns — What NOT to Do

## When to Load

Before any non-trivial implementation work. These are behavioral rules that prevent common agent mistakes.

## Anti-Patterns

### AP1: Delegate Research-Heavy Tasks to Subagents

**Wrong:** Searching 20 files sequentially in the parent agent, burning context on intermediate results.
**Right:** Use `task` or `explore` for wide-net research. Only the distilled answer returns to your context.

### AP2: Read-File Instead of Grep/Glob/CodeGraph

**Wrong:** `read_file` on large files hunting for a symbol.
**Right:** `grep` for patterns, `glob` for file discovery, `mcp__codegraph__search` for symbol-level search. Read files only when you know the exact location.

### AP3: Premature Abstraction

**Wrong:** Extracting a helper function before the pattern repeats 3+ times.
**Right:** Copy-paste is OK for the first two occurrences. Extract on the third.

### AP4: Big-Bang Edits

**Wrong:** Making 10 changes across 8 files, then building.
**Right:** One atomic change → build → test. Iterate. `multi_edit` for multiple edits to a single file.

### AP5: Unverified Claims

**Wrong:** "Fixed." "Done." "Works." without evidence.
**Right:** Show `go build`, `go test`, `go vet` output. Use `complete_step` with verification evidence. Never say "fixed" — only the user declares it.

### AP6: Write Without Read

**Wrong:** Editing a file you haven't read this turn.
**Right:** Re-read the target area before editing. Use `content_hash` on `edit_file`.

### AP7: Shell Instead of File Tools

**Wrong:** `bash("echo 'content' > file.go")` or `bash("sed -i ...")`
**Right:** Use `write_file`, `edit_file`, `multi_edit`, `delete_range` — they're safer, undoable, and tracked.

### AP8: Token Waste on Verbose Tool Output

**Wrong:** Printing full 10K-line build output to context.
**Right:** Pipe through `tail -20` or `grep` for relevant lines. The compressor (`internal/compress/`) handles dedup.

### AP9: Mid-Session Prompt Mutation

**Wrong:** Editing the system prompt or tool descriptions between turns.
**Right:** Ride the turn tail (`pendingMemory` in `controller.go`). Constitution rule: never mutate mid-session.

### AP10: Forgetting Upstream Sync

**Wrong:** Ending a session without `git fetch upstream`.
**Right:** Constitution requirement — always sync upstream before session end. Merge, rebuild, re-vet.

## Tool Defaults to Prefer

| Situation | Prefer | Avoid |
|-----------|--------|-------|
| Symbol search | `mcp__codegraph__search` / `grep` | `read_file` on whole files |
| Architecture questions | `mcp__codegraph__context` | Manual grep + read chain |
| Multi-file edits | `multi_edit` (single file) | Chained `edit_file` calls |
| Wide-net research | `task` / `explore` / `research` | Sequential reads in parent |
| Complex file ops | `write_file` / `move_file` | `bash("mv ...")` / `bash("cat > ...")` |

## Related

- Project skill: `ready-means-tested` — evidence gate
- Project skill: `pre-action-gate` — pre-write verification
- Project skill: `cache-first-architecture` — don't break the prefix
- Constitution: `.reasonix/constitution.json` — all 7 rules
