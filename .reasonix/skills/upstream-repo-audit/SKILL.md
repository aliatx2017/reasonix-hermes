---
name: upstream-repo-audit
description: "Audit upstream GitHub repos for new releases, security fixes, open issues, and integration opportunities. Supports the mandatory upstream-sync workflow."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [upstream, audit, github, security, dependencies, sync]
---

# Upstream Repo Audit

## When to Use

- Checking whether upstream (`esengine/deepseek-reasonix`) has new commits
- Before/after `git fetch upstream && git merge upstream/main-v2`
- Periodic audit of tracked dependencies for security fixes
- Before integration work — check what's changed upstream

## Primary Target

**Upstream:** `esengine/deepseek-reasonix` (branch `main-v2`)
**Our fork:** `aliatx2017/reasonix-hermes` (branch `main`)

Automated sync runs daily via `.github/workflows/sync-upstream.yml` at 20:00 UTC.

## Quick Check (Drift Detection)

```bash
# Check if upstream has new commits since our last sync
git fetch upstream
git log upstream/main-v2 --oneline -10

# Compare with our main
git log main --oneline -10

# See what upstream has that we don't
git log main..upstream/main-v2 --oneline
```

## Full Audit

```bash
# 1. Check our version first (know baseline)
git describe --tags --always

# 2. Upstream releases
gh release list --repo esengine/deepseek-reasonix --limit 5 2>/dev/null

# 3. Recent upstream commits (14 days)
SINCE=$(date -v-14d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -d '14 days ago' +%Y-%m-%dT%H:%M:%SZ)
gh api "repos/esengine/deepseek-reasonix/commits?per_page=10&since=$SINCE" \
  --jq '.[] | "\(.sha[:7]) \(.commit.author.date[:10]) \(.commit.message | split("\n")[0])"' 2>/dev/null

# 4. Security advisories
gh api "repos/esengine/deepseek-reasonix/security-advisories?per_page=5" \
  --jq '.[] | "\(.ghsa_id) \(.severity) \(.summary)"' 2>/dev/null

# 5. Open issues tagged as bug
gh issue list --repo esengine/deepseek-reasonix --label bug --limit 10 --state open
```

## Tracked Dependencies

Key Go dependencies to watch:

| Dependency | Purpose |
|-----------|---------|
| `github.com/BurntSushi/toml` | Config parsing |
| `github.com/slack-go/slack` | Slack bot adapter |
| `github.com/go-telegram-bot-api/telegram-bot-api/v5` | Telegram bot adapter |
| `github.com/bwmarrin/discordgo` | Discord bot adapter |
| `github.com/gorilla/websocket` | Collab WebSocket hub |
| `github.com/wailsapp/wails/v2` | Desktop framework |

```bash
# Check for available updates
go list -u -m -json all | grep -E '"Path"|"Version"'
```

## Rate Limit Awareness

GitHub API: 60 req/hr unauthenticated, 5,000 req/hr with `gh` CLI token. `gh` CLI is preferred — it uses keychain-authenticated requests.

**Never run parallel async calls** — instant rate limit. Sequential calls only.

## After Audit

1. If upstream has new commits → merge: `git fetch upstream && git merge upstream/main-v2`
2. Resolve conflicts, keeping our custom features
3. `go build ./... && go vet ./...` — must pass
4. `go test ./internal/...` — must pass
5. Update `REASONIX.md` with new sync point and commit hash
6. Rebuild all binaries

## Related

- Constitution constraint: "Always sync upstream (git fetch upstream) before ending a session"
- Constitution constraint: "Always run go build ./... && go vet ./... after upstream merges"
- `.github/workflows/sync-upstream.yml` — automated daily sync
- `REASONIX.md` — sync history and merge notes
- `AGENTS.md` — "Syncing with Upstream" section
