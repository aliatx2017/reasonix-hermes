# Reasonix Hermes

Customized Reasonix AI coding agent based on upstream [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) with added MCP bridges, Discord bot, skills hub, and community tooling.

## Project

- **Repo**: `github.com/aliatx2017/reasonix-hermes` (our fork)
- **Upstream**: `github.com/esengine/deepseek-reasonix` (tracked as `upstream` remote, branch `main-v2`)
- **Module**: `reasonix`
- **Stack**: Go 1.25 (CLI + backend), React 19 + TypeScript (desktop frontend), Wails v2 (desktop shell)
- **Models**: DeepSeek V4 Flash (default), DeepSeek V4 Pro, MiMo v2.5 Pro (planner)

## Syncing with Upstream

```bash
git fetch upstream
git merge upstream/main-v2     # merge upstream changes
# resolve any conflicts in our custom files
git push origin main
```

## Commands

```bash
# Build the CLI
go build -o bin/reasonix ./cmd/reasonix

# Build our custom binaries
go build -o bin/reasonix-bridge ./pkg/mcpbridge
go build -o bin/reasonix-memory ./pkg/memoryserver
go build -o bin/reasonix-bot ./bot

# Build + vet everything
go build ./...
go vet ./...

# Run the CLI
./bin/reasonix chat
./bin/reasonix run "task description"
./bin/reasonix setup

# Run the Discord bot
export DISCORD_BOT_TOKEN="..." DISCORD_SERVER_ID="..."
./bin/reasonix-bot

# Desktop dev mode
cd desktop/frontend && npm install && cd ../..
cd desktop && wails dev
```

## Architecture

```
cmd/reasonix/          CLI entry point (delegates to internal/cli/)
internal/              Upstream Reasonix engine (39 packages)
  agent/               Core agent loop, compaction, subagents
  checkpoint/          File-snapshot undo system
  config/              TOML config loader + model fallback
  permission/          Tool-call permission gating
  skill/               Built-in skills registry
  tool/                Built-in tools (bash, read, write, edit, etc.)
  plugin/              MCP client (stdio, HTTP, SSE)
  provider/            LLM providers (Anthropic, OpenAI/DeepSeek)
  bot/                 Feishu/WeChat/QQ bot adapters (upstream)
  codegraph/           Semantic code index
  lsp/                 Language server integration
  ...                  (30+ more packages)
pkg/                   ── Our custom additions ──
  mcpbridge/           MCP bridge server (5 tools: run, doctor, plan, orchestrate, skills)
  memoryserver/        Hindsight MCP server (3 tools: retain, recall, reflect)
bot/                   Our Discord bot gateway (slash commands + text triggers)
desktop/               Wails v2 desktop app (upstream, full-featured)
skills-hub/            Our 16-skill community registry + Markdown files
```

## Our Customizations

| Layer | What | Why |
|-------|------|-----|
| `pkg/mcpbridge/` | MCP bridge server | Expose Reasonix to Claude Code/Codex via MCP |
| `pkg/memoryserver/` | Hindsight memory server | Cross-session persistent memory |
| `bot/` | Discord bot gateway | Discord integration (upstream has Feishu/WeChat/QQ only) |
| `skills-hub/` | 16 community skills | Curated skill registry with frontmatter playbooks |
| `reasonix-deepseek-ecosystem-2026.md` | Ecosystem reference | Comprehensive survey of integrations and plugins |
| `.github/workflows/ci-hermes.yml` | Supplementary CI | Desktop frontend build in CI |

## Notes

- Upstream remote: `https://github.com/esengine/deepseek-reasonix.git` (branch `main-v2`)
- Our fork: `https://github.com/aliatx2017/reasonix-hermes.git` (branch `main`)
- To pull upstream updates: `git fetch upstream && git merge upstream/main-v2`
- `reasonix.toml` is gitignored (upstream convention) — never commit secrets
- Discord bot uses `github.com/bwmarrin/discordgo` (added to go.mod)
