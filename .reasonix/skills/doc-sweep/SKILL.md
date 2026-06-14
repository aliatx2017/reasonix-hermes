---
name: doc-sweep
description: Systematic audit and enrichment of project documentation against the live codebase — inventory, cross-reference, identify gaps, fix stale docs, create new docs, cross-link.
---

# Documentation Sweep

> Systematic audit and enrichment of project documentation against the live codebase.

## When to Use

- After a major feature push — ensure docs match reality
- Before a release — hunt stale claims and missing coverage
- When onboarding — verify docs are complete and accurate
- When `docs/` has grown organically and needs a fresh cross-reference

## Process

### Phase A: Inventory (5 min)

Catalog every documentation file in the project:

```bash
ls docs/*.md *.md deploy/*.md skills-hub/**/*.md .reasonix/**/*.md 2>/dev/null
```

Also inventory all code packages:

```bash
ls -d internal/*/ cmd/*/ desktop/ 2>/dev/null
```

Count and group: root-level docs, feature guides, reference docs, historical docs.

### Phase B: Cross-Reference Against Codebase (15 min)

For each doc, answer:

1. **What features/packages does it claim to cover?** — list them
2. **What features/packages actually exist?** — from `internal/*/` + `cmd/*/`
3. **Gap**: which actual packages have zero mentions in any doc?
4. **Staleness**: which doc claims are false against the current code?

Key checks per doc:
- Binary names still correct? (e.g. `reasonix-mcpbridge`, not `reasonix-bridge`)
- Package paths still correct? (e.g. `internal/eval/`, not a deleted package)
- Features listed still exist? (grep for symbols/files)
- Clone URLs point to the right repo?
- Version numbers current?
- Platform/bot names complete? (Discord, Telegram, LINE, Slack, Feishu, WeChat, QQ)
- i18n locales match? (en, zh, zh-TW)

### Phase C: Identify Gaps (5 min)

Features with **no standalone doc** at all — for each, decide:
- Needs a standalone doc (major feature, user-facing)
- Needs a section in an existing guide (small feature, developer-facing)
- Already covered in SPEC.md §2 (package reference, not user guide)

Priority order: user-facing > developer-facing > internal-only.

### Phase D: Fix and Enrich (20 min)

For each identified issue:

**Stale doc fix pattern:**
1. Add a historical note at the top if the doc is old ("Historical — June 2025")
2. Update all stale claims inline
3. Add cross-reference to current docs

**New doc template:**
```markdown
# Feature Name

> One-line elevator pitch — what it does and how to use it.

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

**Cross-linking**: Every new doc must be linked from:
1. `README.md` — Documentation table
2. `docs/HERMES-GUIDE.md` — Contents + relevant section
3. Any related feature guides

### Phase E: Verify (2 min)

```bash
go build ./... && go vet ./...
ls -la docs/NEW-DOC.md docs/OTHER-NEW-DOC.md
```

## Output

A report like:

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

## Rules

- Never delete docs — mark as historical instead
- Keep new docs focused (2000–5000 bytes) — link to HERMES-GUIDE for depth
- Always cross-link new docs from README and HERMES-GUIDE
- Code examples must match the actual CLI/API as it exists today
- Verify with `go build ./...` after every batch of changes
