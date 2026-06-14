---
name: ready-means-tested
description: "Enforce 'nothing is ready until end-to-end tested with real evidence.' Gate that must pass before any component status changes to DONE/READY/LIVE. Encodes the substantiate-every-claim constitution rule."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [testing, verification, evidence, gate, constitution]
---

# Ready Means Tested

## Rule

NO change, fix, or feature is READY/DONE/LIVE unless end-to-end verified with real evidence. Code inspection, "it compiles," "it deployed," "the tests pass locally" — do NOT qualify. You must show tool output from the actual execution proving the change works.

**"Fixed" / "done" / "verified" / "working" — ONLY valid with evidence in the SAME response.** If you can't point to `go build`, `go test`, or `go vet` output proving it, say "should be resolved" — never "fixed." This is encoded in the constitution as the `never-say-fixed` and `substantiate-every-claim` rules.

## Claim Words Require Evidence

Before emitting any claim word, the SAME response must contain:
- `go build ./...` output showing clean build
- OR `go test ./...` output showing pass
- OR `go vet ./...` output showing no warnings
- OR an actual file diff or command output proving the change

Default: if there's no tool output in the same response producing build/test/vet evidence, don't use claim words. Describe what was done, not that it works. Only the user declares something fixed after testing.

## Red Flags (BLOCK status change)

- Agent claims "fixed" based on reading source code only
- Agent claims "fixed" based on edit success without build verification
- Agent claims "tests pass" without running them
- Agent claims "builds" without `go build ./...`
- Agent claims "done" without completing ALL steps of an approved plan

## Gate — must pass ALL before claiming DONE

1. Run `go build ./...` — must pass
2. Run `go vet ./...` — must pass
3. Run `go test ./...` for the affected package — must pass
4. Run `go test ./internal/...` for broader changes — must pass
5. For desktop changes: `cd desktop/frontend && npx tsc --noEmit` — must pass
6. Show the evidence inline (tool output in the same response)
7. If ANY step fails: the step is not done, do not claim it is

## Status vocabulary

- NOT VERIFIED: change made, zero build/test verification
- BUILDING: `go build` in progress
- VERIFIED: build + vet + tests pass, evidence shown

## Evidence Chain

Every complete_step requires ≥1 evidence item with kind=verification, diff, files, or manual. Each verification evidence must include the exact command that was run. This is structural enforcement — the host rejects completions without evidence.

## Related

- Constitution rule: `never-say-fixed` — banned word, only user declares fixed
- Constitution rule: `substantiate-every-claim` — verify first, speak second
- Constitution rule: `go-vet-clean` — `go build ./... && go vet ./...` must pass before committing
- Constitution rule: `typescript-clean` — `tsc --noEmit` must pass for desktop changes
- Project skill: `evidence-first-reasoning` — diagnostic method for ambiguity
