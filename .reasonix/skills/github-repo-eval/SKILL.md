---
name: github-repo-eval
description: "Evaluate a GitHub repo by deep-diving into source code, not just README marketing. Produces an honest assessment of what's real vs. vaporware."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [github, evaluation, audit, due-diligence, competitive-analysis]
---

# GitHub Repo Evaluation

## When to Use

- Evaluating a dependency before adding it to `go.mod`
- Competitive analysis — what's real vs. marketing in a competitor's repo
- Deciding whether to adopt patterns from another project
- Auditing a fork or upstream for quality

## Process

### 1. Surface Scan (5 min)
- README: what do they CLAIM to do?
- Stars, forks, open issues, last commit date
- License, contributing guide, code of conduct
- Release frequency and versioning

### 2. Structure Assessment (10 min)
- Does the directory layout match the README claims?
- Are there tests? What's the coverage like?
- How many dependencies? Any abandoned ones?
- Build system: does it actually build?

### 3. Deep Dive (20 min)
- Pick 2-3 "hero" features from the README
- Trace them from entry point to implementation
- Are they real or stubbed?
- Code quality: error handling, documentation, patterns

### 4. Community Health
- Issue response time (sample last 10 closed issues)
- PR merge rate and review quality
- Bus factor: how many unique contributors in last 6 months?

## Output: Honest Assessment

```
## <repo-name> Evaluation

### Claims vs. Reality
| Claim | Status | Evidence |
|-------|--------|----------|
| "Production-ready" | ⚠️ Partial | No tests, no CI, panics on edge cases |
| "Cross-platform" | ✅ Real | Build tags for darwin/linux/windows |

### Red Flags
- Last commit 18 months ago
- 47 open PRs, oldest 2 years
- No license file

### Green Flags
- Comprehensive test suite (89% coverage)
- Active maintainer responding within 24h
- Clear architecture, well-documented

### Verdict
Adopt / Fork / Watch / Skip
```

## For Go Dependencies

```bash
# Quick health check
gh repo view owner/repo --json name,stargazerCount,forkCount,updatedAt,licenseInfo
gh api repos/owner/repo/commits --jq '.[0].commit.author.date'
gh issue list --repo owner/repo --limit 5 --state open

# Code quality
git clone https://github.com/owner/repo /tmp/eval-repo
cd /tmp/eval-repo
go build ./... 2>&1 | head -20
go vet ./... 2>&1 | head -20
go test ./... 2>&1 | tail -5
```

## Related

- Project skill: `intent-gap-analysis` — compare implementation against stated intent
- Project skill: `upstream-repo-audit` — audit tracked upstream dependencies
- Project skill: `evidence-first-reasoning` — verify claims with evidence
- `docs/CHANGELOG-HERMES.md` — existing competitive analysis
