---
name: simplify-code
description: "Parallel 3-agent cleanup of recent code changes — one reviews correctness, one reviews style/patterns, one reviews security. Merge findings into a unified fix pass."
version: 1.0.0
author: Hermes Agent (adapted for Reasonix-Hermes)
tags: [code-review, cleanup, refactor, subagent, parallel, simplify]
---

# Simplify Code — Parallel Review & Cleanup

## When to Use

- After a multi-step feature or bug fix, before committing
- When a PR has grown messy across several commits
- User says "clean this up" or "simplify"
- Before upstream merge — review our changes against upstream code

## Process

Launch three subagent reviews in parallel, then merge findings:

### Agent 1: Correctness Review
```
Review the current branch diff. Focus on:
- Logic errors, off-by-one, nil dereference potential
- Missing error handling
- Race conditions (check mutex usage, goroutine lifecycle)
- Breaking changes to interfaces or public APIs
Flag severity: critical / warning / note
```

### Agent 2: Style & Patterns Review
```
Review the current branch diff. Focus on:
- Go idiom violations (use go vet findings)
- Inconsistent naming, formatting, comment style
- Over-engineered code — could this be simpler?
- Missing tests for new code paths
- Duplicated logic that could be extracted
Flag severity: should-fix / nice-to-have
```

### Agent 3: Security Review
```
Review the current branch diff. Focus on:
- Injection vectors (shell, SQL, template)
- AuthZ/AuthN bypass potential
- Secrets exposure (api keys, tokens in logs)
- Input validation gaps
- Path traversal, SSRF
Flag severity: critical / warning / note
```

## Merge & Fix

1. Collect all three review outputs
2. Triage: fix criticals first, then warnings, then notes
3. Apply fixes in priority order
4. Re-run `go build ./... && go vet ./... && go test ./...`
5. Confirm all three agents would now pass

## Parallel Dispatch

```bash
# The review, security_review, and explore tools can run concurrently
# since they're read-only. Launch all three, wait for all, then fix.
```

## When NOT to Use

- Trivial single-file changes (one review is enough)
- Changes already covered by CI
- Emergency hotfixes (defer cleanup to follow-up)

## Related

- Built-in tool: `review` — single-agent code review subagent
- Built-in tool: `security_review` — single-agent security review subagent
- Constitution rule: `go-vet-clean` — `go build ./... && go vet ./...` must pass
- Project skill: `ready-means-tested` — verify fixes with evidence
