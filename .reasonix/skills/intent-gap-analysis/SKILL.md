---
name: intent-gap-analysis
description: "Analyze a codebase by identifying where the implementation lags behind the stated intent, rather than searching for generic improvements or missing features."
version: 1.0.0
author: Hermes Agent (adapted for Reasonix-Hermes)
tags: [analysis, audit, architecture, gap-analysis, diagnostic]
---

# Intent Gap Analysis

## When to Use

- User asks "is this codebase doing what it claims to do?"
- Auditing a project against its stated goals or SPEC.md
- Comparing implementation against constitution or design principles
- Finding drift between documentation and reality

## Method

Instead of searching for "what's missing" (which returns everything), compare the codebase against its OWN stated intent:

1. **Extract intent** — Read SPEC.md, README, constitution, package comments, design docs
2. **Map implementation** — For each stated intent, find the corresponding code paths
3. **Measure gap** — Where does the implementation fall short of the intent?
4. **Prioritize** — Rank gaps by impact: correctness > security > performance > completeness

## Output

A markdown report with:

```
## Intent vs. Implementation

| Stated Intent | Source | Implementation | Gap | Severity |
|--------------|--------|---------------|-----|----------|
| "Single static binary" | SPEC.md §1.2 | go build produces 6 binaries | 6 binaries, not 1 | Low |
| "Cache-first prefix" | constitution.json | controller.go Compose | ✅ matches | None |
...
```

## Key Questions

- What does this codebase CLAIM to be/do?
- Where in the code is each claim implemented?
- What claims have no corresponding code?
- What code exists that contradicts a claim?
- What's changed since the docs were written?

## Related

- Project skill: `evidence-first-reasoning` — verify discriminatively
- Project skill: `github-repo-eval` — evaluate external repos
- Constitution: `.reasonix/constitution.json` — project invariants
- SPEC: `docs/SPEC.md` — engineering contract
