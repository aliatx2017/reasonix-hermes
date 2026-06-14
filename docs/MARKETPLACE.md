# Skill Marketplace

> Community skill registry and LobeHub marketplace integration for Reasonix-Hermes.
> Discover, install, and share agent skills.

## Quick Start

```bash
# List locally installed marketplace skills
reasonix marketplace list

# Search the registry
reasonix marketplace search "git"

# Install a skill
reasonix install-source install --source https://github.com/author/skill-repo

# Sync from LobeHub (360k+ community skills)
reasonix marketplace sync
```

## What Is the Marketplace?

The Reasonix-Hermes marketplace is a community-driven skill registry with two
sources:

1. **Built-in registry** — 12 curated skills shipped with Hermes, covering
   common agent patterns (adversarial review, API design, code review, CI/CD,
   debugging, documentation, exploration, frontend, git commit, migration,
   refactoring, security audit, test generation).

2. **LobeHub integration** — sync from [LobeHub](https://lobehub.com)'s
   marketplace with 360k+ community-authored skills. Uses machine-to-machine
   OAuth2 (HS256 JWT, stdlib-only) for automatic registration.

## Registry Format

Skills are indexed in `skills-hub/registry.json` (agentskills.io-compatible format):

```json
{
  "name": "code-review",
  "description": "Comprehensive code review with security, performance, and style checks",
  "author": "Innei",
  "tags": ["review", "quality", "security"],
  "rating": 4.8,
  "url": "https://raw.githubusercontent.com/Innei/skills/main/code-review.md"
}
```

## LobeHub Sync

```toml
# In reasonix.toml — persists LobeHub OAuth credentials across sessions
[marketplace.lobehub]
enabled = true
client_id = ""        # auto-generated on first sync
client_secret = ""    # auto-generated on first sync
sort_by = "installCount" # downloads, rating, installCount
model = ""            # optional model for skill evaluation
```

On first `reasonix marketplace sync`, the CLI registers an M2M client with
LobeHub and saves credentials to your config. Subsequent syncs use the saved
credentials. Fetched skills are merged into the local registry.

## Desktop

The desktop app has a **Skill Store** panel (4 tabs):

| Tab | Content |
|-----|---------|
| **LobeHub** | Sync from LobeHub marketplace, browse fetched skills |
| **Market** | Browse the built-in registry, search by name/tag |
| **MCP** | Live MCP server status (tools, prompts, transport) |
| **Custom** | Manage locally installed skills |

Click **Sync from LobeHub** to pull the latest community skills. Skills can be
installed with one click.

## Architecture

`internal/marketplace/` — registry management, skill indexing, LobeHub API client.
Skills are loaded as Markdown playbooks that the agent can invoke via
`run_skill({ name: "<name>" })` or the `/skill` slash command.

## Related

- `skills-hub/` — the curated skill registry itself
- `docs/HERMES-GUIDE.md` §11 — skills and subagents
- `docs/SPEC.md` §2 — `internal/marketplace/` package
