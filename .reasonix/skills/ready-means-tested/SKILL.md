---
name: ready-means-tested
description: "Nothing is ready until end-to-end tested with real evidence. Blocks DONE/READY/LIVE claims without build output in the same response."
version: 2.0.0
author: Reasonix-Hermes (rewritten with writing-great-skills principles)
tags: [testing, verification, evidence, gate]
---

# Ready Means Tested

One rule, one gate, one evidence chain.

## Rule

No change is ready unless end-to-end verified with real evidence in the same response. Code inspection, "it compiles," "the tests pass locally" — these do not qualify. Show the tool output.

"Fixed" is a banned word — only the user declares something fixed. Say "should be resolved" or "try it."

## Gate

Must pass all before claiming a step is done:

1. `go build ./...` — clean
2. `go vet ./...` — clean
3. `go test ./...` for affected packages — pass
4. `cd desktop/frontend && npx tsc --noEmit` for desktop changes — clean
5. Show the output in the same response

If any step fails, the step is not done. Iterate, don't claim.

## Evidence chain

`complete_step` requires ≥1 evidence item per completion. Each verification evidence must include the exact command as it ran. The host matches commands against the session transcript — approximate citations are rejected. Use the command string from the bash output, verbatim.

## Failure modes

- **Inspection-as-evidence** — reading code and concluding it works. Build output only.
- **Stale-timestamp "passes"** — citing a test run from a previous turn. Re-run it.
- **Command mismatch** — the host rejects evidence when the cited command doesn't match what actually ran. Copy the command from bash output exactly.

## Related

- `pre-action-gate` — pre-write safety checkpoint
- `evidence-first-reasoning` — diagnostic verification under ambiguity
