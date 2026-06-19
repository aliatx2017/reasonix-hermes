---
name: doc-sweep
description: "Systematic documentation audit — inventory every doc, cross-reference against the live codebase, fix stale claims, create missing docs, cross-link."
version: 2.0.0
author: Reasonix-Hermes (rewritten with writing-great-skills principles)
tags: [documentation, audit, maintenance]
---

# Documentation Sweep

Five phases. Each has a completion criterion — don't move to the next until the current one is done.

## Phase A — inventory

Catalog every documentation file and every code package:

```bash
ls docs/*.md *.md deploy/*.md skills-hub/**/*.md .reasonix/**/*.md 2>/dev/null
ls -d internal/*/ cmd/*/ desktop/ 2>/dev/null
```

**Done when**: you have two lists — every doc path, every package path. Count them. State the counts.

## Phase B — cross-reference

For each doc, answer four questions:
1. What packages does it claim to cover?
2. What packages actually exist? (from Phase A inventory)
3. Which actual packages have zero mentions in any doc? → **gap**
4. Which doc claims are false against current code? → **staleness**

Key checks: binary names correct? package paths exist? features still live? clone URLs point at the right repo? version numbers current? bot platform list complete? i18n locales match?

**Done when**: you have a gap list and a staleness list. Every item cites a specific doc:line and the evidence (grep/read that proves it).

## Phase C — prioritize gaps

For each undocumented feature, decide its tier:
- **Standalone doc** — major feature, user-facing
- **Section in existing guide** — small feature, developer-facing
- **Covered by SPEC.md §2** — package reference, no doc needed

**Done when**: every gap has a tier assignment. State it.

## Phase D — fix and enrich

For each stale claim: update inline, add a historical note if the doc is very old.

For each new doc, use this template:

```markdown
# Feature Name

> One-line elevator pitch.

## Quick Start
```bash
reasonix <command> <args>
```

## What It Does
...

## Configuration
```toml
[section]
key = "value"
```

## Architecture
Which packages power it.

## Related
- `docs/OTHER-DOC.md` — related features
```

Every new doc must be linked from `README.md` (Documentation table) and `docs/HERMES-GUIDE.md` (Contents + relevant section).

**Done when**: every stale claim is fixed, every needed doc is created, and every new doc is cross-linked from README and HERMES-GUIDE.

## Phase E — output

Produce a summary table:

```
## Documentation Sweep

### Stale (fixed)
| Doc | Issue | Fix |
|-----|-------|-----|

### Created
| Doc | Covers |
|-----|--------|

### Enriched
| Doc | Added sections |
|-----|----------------|

### Remaining gaps (deferred)
- feature-X: covered in HERMES-GUIDE §Y, standalone doc deferred
```

**Done when**: the summary is complete and `go build ./...` passes.

## Rules

- Never delete docs — mark as historical instead
- Keep new docs focused — link to HERMES-GUIDE for depth
- Code examples must match the actual CLI/API as it exists today
