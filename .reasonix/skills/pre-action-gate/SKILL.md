---
name: pre-action-gate
description: "Pre-write safety gate — re-read targets, lock hashes, and verify the build chain before calling any change complete."
version: 2.0.0
author: Reasonix-Hermes (rewritten with writing-great-skills principles)
tags: [gate, verification, safety]
---

# Pre-Action Gate

Load before any write, edit, or deploy. This is a _gate_ — it doesn't produce output; it blocks premature moves.

## The gate

**Before any file mutation** — re-read the target area. Code drifts between reads; stale context is the #1 cause of garbled edits. Use `content_hash` on `edit_file` whenever you touch a file.

**After any change** — `go build ./...` must pass before you call the step done. Not "should pass," not "probably passes" — you run it and show the output. One change → one build. Iterate.

**Before claiming completion** — evidence must be in the _same response_ as the claim. `complete_step` enforces this structurally: the host rejects completions without verification commands. Don't fight it — feed it.

## What this gate replaces

This skill encodes three constitution rules into one behavioral checkpoint. Load it instead of consulting the constitution directly — it's the distilled form:

- `go-vet-clean` → build chain check
- `substantiate-every-claim` → evidence-before-claim
- `never-say-fixed` → only the user declares fixed

## Failure modes

- **Stale edit** — file changed since last read → garbled code. Re-read every time.
- **Premature "done"** — claiming completion without build output. The host blocks it; don't waste the turn.
- **Batch-and-pray** — 10 edits, then one build. One edit → one build. `multi_edit` for single-file batches only.

## Related

- `ready-means-tested` — evidence gate for status claims
- `evidence-first-reasoning` — diagnostic verification under ambiguity
- Constitution: `.reasonix/constitution.json`
