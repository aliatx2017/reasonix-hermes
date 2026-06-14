---
name: pre-action-gate
description: "Mandatory pre-action verification — load before ANY write, edit, or deploy. Gate checklist to prevent common mistakes: wrong target, missing evidence, premature claims."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [gate, verification, mandatory, safety]
---

# Pre-Action Gate

## When to Use

Load BEFORE:
- Any `edit_file`, `write_file`, `delete_range`, `multi_edit`, or `move_file` call
- Any `bash` command that mutates files or the environment
- Any claim of completion, success, or readiness
- Any upstream merge or config change

## Gate Checklist

### G1: Target Verification
- Before any file write: re-read the target area to confirm the code hasn't changed since your last read. Use `content_hash` parameter on `edit_file` when possible.
- Before any bash command: verify the working directory and target files exist.
- After any config change: verify `go build ./...` still passes before proceeding.

### G2: Evidence Before Claim
- Am I about to say "done," "works," "passes," "fixed"? → **BLOCKED.** Show tool output FIRST.
- Use `complete_step` with verification evidence (command + result) before claiming a step is done.
- Never claim something is fixed — say "should be resolved" or "try it." Only the user declares fixed.

### G3: Constitution Check
- Does this change violate any constitution rule? Check:
  - `spec-first`: Did I update SPEC.md first for internal changes?
  - `controller-seam`: Am I adding behavior to control.Controller, not individual frontends?
  - `init-registration`: Does a new built-in tool/provider need `init()` registration?
  - `go-vet-clean`: Will `go build ./... && go vet ./...` still pass?
  - `no-nil-slices`: Do Wails Go bindings return `[]T{}` not `nil`?
  - `i18n-complete`: Do new i18n fields populate all 3 catalogs?

### G4: Build Chain
1. `go build ./...` — must pass before claiming a step is done
2. `go vet ./...` — must pass
3. Relevant tests must pass
4. For desktop: `tsc --noEmit` must pass
5. Full test suite for broad changes: `go test ./internal/...`

### G5: Upstream Awareness
- After every session: `git fetch upstream`, merge `upstream/main-v2`, rebuild, re-vet.
- Never leave a session without checking upstream.

## Failure Modes

- **Stale context edit**: Editing a file that changed since last read → garbled code. Always re-read before editing.
- **Unverified claim**: Saying "done" without showing build output. Use `complete_step` with verification.
- **Constitution violation**: Missing SPEC update, bypassing controller seam, returning nil slices.
- **Forgotten upstream sync**: Session ends without `git fetch upstream`.

## Related

- Constitution: `.reasonix/constitution.json` — 7 principles, 6 constraints, 7 rules
- Project skill: `ready-means-tested` — evidence gate for status claims
- Project skill: `evidence-first-reasoning` — diagnostic verification method
- REASONIX.md — "Upstream sync every session wrap-up"
