# Reasonix Hermes

Customized Reasonix AI coding agent based on upstream [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) with added MCP bridges, Discord bot, skills hub, and community tooling.

## Project

- **Repo**: `github.com/aliatx2017/reasonix-hermes` (our fork)
- **Upstream**: `github.com/esengine/deepseek-reasonix` (tracked as `upstream` remote, branch `main-v2`)
- **Module**: `reasonix`
- **Stack**: Go 1.25 (CLI + backend), React 19 + TypeScript (desktop frontend), Wails v2 (desktop shell)
- **Models**: DeepSeek V4 Flash (default), DeepSeek V4 Pro, MiMo v2.5 Pro (planner)

## Syncing with Upstream

Automated: `.github/workflows/sync-upstream.yml` runs daily at 20:00 UTC (04:00 CST, when upstream devs are asleep). It fetches upstream `main-v2`, merges cleanly, runs `go build ./...` + `go test ./...`, and pushes. On conflict, it opens a PR for manual resolution.

Manual (if needed):

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
go build -o bin/reasonix-bridge ./cmd/reasonix-mcpbridge
go build -o bin/reasonix-memory ./cmd/reasonix-memoryserver
go build -o bin/reasonix-bot ./bot
go build -o bin/reasonix-hooks ./cmd/reasonix-hooks

# Install skills via upstream install_source
reasonix install-source install --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills

# Build + vet everything
go build ./...
go vet ./...

# Run tests (228 tests across all custom packages)
go test ./cmd/... ./pkg/... ./internal/bot/...

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
internal/              Upstream Reasonix engine (40+ packages)
  agent/               Core agent loop, compaction, subagents
  checkpoint/          File-snapshot undo system
  config/              TOML config loader + model fallback
  constitution/        Structured project invariants (.reasonix/constitution.json)
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
  httputil/            Shared Bearer auth middleware
  mcputil/             Shared MCP types and server helpers
  (see pkg/README.md)  Library documentation
cmd/                   ── Our custom binaries ──
  reasonix-mcpbridge/  MCP bridge server (6 tools: run, doctor, plan, orchestrate, get_skill, get_skills)
  reasonix-memoryserver/ Hindsight MCP server (3 tools: retain, recall, reflect; SQLite + file, TTL/importance, vector search)
bot/                   Our Discord bot gateway (slash commands + /goal + /model)
cmd/reasonix-hooks/    Native Go hook runner (zero-dependency binary)
desktop/               Wails v2 desktop app (upstream, full-featured)
skills-hub/            17-skill community registry + static catalog site
```

## Our Customizations

| Layer | What | Why |
|-------|------|-----|
| `pkg/mcputil/` + `pkg/httputil/` | Shared Go libraries | Bearer auth middleware + MCP types/helpers |
| `cmd/reasonix-mcpbridge/` | MCP bridge server (6 tools) | Expose Reasonix to Claude Code/Codex via MCP |
| `cmd/reasonix-memoryserver/` | Hindsight memory (3 tools, SQLite, TTL, vector) | Cross-session persistent memory with semantic search |
| `bot/` + `internal/bot/discord/` | Discord bot (+ /goal + /model) | Discord integration (upstream has Feishu/WeChat/QQ only) |
| `cmd/reasonix-hooks/` | Native Go hook runner | Zero-dependency binary for PreToolUse/Stop hooks |
| `skills-hub/` | 17 community skills + catalog site | Curated skill registry with frontmatter playbooks |
| `reasonix-hermes.json` | Install source manifest | `reasonix install-source install --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills` |
| `reasonix-deepseek-ecosystem-2026.md` | Ecosystem reference | Comprehensive survey of integrations and plugins |
| `.github/workflows/ci-hermes.yml` | Supplementary CI | Desktop frontend build + Hermes package tests in CI |

## Docs

- **[Implementation Plan](docs/HERMES-IMPLEMENTATION-PLAN.md)** — phased roadmap: 17/17 items complete across P0–P3
- **[Research Findings](docs/RESEARCH-FINDINGS-JUNE-2026.md)** — June 2026 deep-web sweep: upstream v1.5.0, 4 new MCP bridges, 4 skill packs, 2 domain apps, 4 desktop clients, 4 IDE extensions, 11 undocumented features
- **[Ecosystem Reference](reasonix-deepseek-ecosystem-2026.md)** — full landscape: MCP bridges, skills, desktop, IDE, forks, cost model, protocols, use cases

## Notes

- Upstream remote: `https://github.com/esengine/deepseek-reasonix.git` (branch `main-v2`)
- **Upstream target**: v1.5.0 (June 10, 2026) — ✅ synced (e5e8f02). Key additions inherited: bot gateway, goal mode, read_skill, PDF extraction, themeable workspace, React 19/TS 6, ACP sessions, 100+ fixes.
- Our fork: `https://github.com/aliatx2017/reasonix-hermes.git` (branch `main`)
- To pull upstream updates: `git fetch upstream && git merge upstream/main-v2`
- `reasonix.toml` is gitignored (upstream convention) — never commit secrets
- Discord bot uses `github.com/bwmarrin/discordgo` (added to go.mod)
- Discord bot must use `control.Controller` like every other frontend — not inline chat history
- **Tests**: 228 tests across 7 packages. `go test ./cmd/... ./pkg/... ./internal/bot/...`
- **CodeWhale features** (10/10 done, 2026-07-04): Shell env hooks, parallel sub-agent batch dispatch, completion sound, harness profiles, constitution system, workshop sidecar, desktop hotbar, external sandbox, Nix flake, Dockerfile.
- **New packages**: `internal/constitution/` (structured project invariants from `.reasonix/constitution.json`)
- **New files**: `flake.nix`, `Dockerfile`, `.dockerignore`, `internal/sandbox/remote.go`
- **Config additions**: `[notifications].sound`, `active_profile`, `[profiles.<name>]` blocks, `[sandbox].remote_sandbox_url`, `[sandbox].remote_sandbox_token`

## roach-code Multi-Provider Research

**Repo**: `tmdgusya/roach-code` (⭐34, v1.3.5, 44 commits, 15 releases)

A multi-model rewrite of deepseek-reasonix that generalizes from DeepSeek-only to any provider. Same architecture (same internal/ layout, CLI, tools, MCP client) — a rebrand + multi-provider extension of upstream.

**Provider additions beyond upstream:**
| Provider | Kind | Notes |
|----------|------|-------|
| Codex/OpenAI | `codex` | Responses API + ChatGPT OAuth login (`roach-code codex login`) |
| MiniMax | `minimax` | Multimodal: text, image, video, speech, music, vision |
| GLM | `glm` | Z.ai API — Chinese LLM provider |

**Key patterns worth adopting:**
1. **`roach-code models` / `roach-code models refresh`** — CLI command to list/refresh configured models. Upstream Reasonix has model switching but no dedicated model list command.
2. **OAuth login flow** (`roach-code codex login`) — browser-based OAuth for ChatGPT subscribers. Pattern for adding auth-bound providers.
3. **Self-update** (`roach-code update`) — downloads latest release binary. Already in upstream goreleaser but not surfaced as a CLI command.
4. **Install scripts** (`install.sh`, `install.ps1`) — bash/PowerShell installers with SHA256 verification. Upstream relies on npm/brew/prebuilt archives.
5. **Config namespace** — uses `roach-code.toml` / `~/.config/roach-code/` / `.roach-code/` (not `reasonix.toml` / `.reasonix/`). Simplest way to avoid conflicts when both are installed.
6. **Short alias** (`roach`) — `make install` creates `roach` symlink. Upstream has `dsnix` alias built-in.

**Relevance to Hermes**: The `internal/provider/` registry already supports adding new providers via `init()` registration. Adding MiniMax/GLM would follow the same pattern as `provider/openai/` and `provider/anthropic/`. The key work is implementing each provider's wire format (OpenAI-compatible for MiniMax/GLM, proprietary for Codex).

## VS Code Extension Fork

**Source**: `whishi47/deepseekcode-reasonix-vscode` (⭐1, MIT, TypeScript)

**To fork** (manual GitHub action):
1. Fork to `aliatx2017/reasonix-hermes-vscode`
2. Rename branding: `DeepSeekCode` → `Reasonix Hermes`, 🐋 → Hermes icon
3. Update `package.json`: name, displayName, description, publisher, repository
4. Update `README.md`: replace DeepSeekCode references with Hermes
5. Terminal name: `DeepSeekCode` → `Hermes`
6. Command prefix: `deepseekcode` → `reasonix` (or keep as configurable)
7. Publish to VS Code Marketplace as `reasonix-hermes-vscode`

**Key features to preserve**:
- 3 keyboard shortcuts (Ctrl+Esc, Ctrl+Shift+Esc, Ctrl+Alt+K)
- 2.5s readiness delay before auto-injecting `@file#L10-L20`
- Smart terminal reuse (same-name terminal not duplicated)
- Compatible with Windsurf/Trae
