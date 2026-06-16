# Reasonix Hermes

Customized Reasonix AI coding agent based on upstream [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) with added MCP bridges, Discord bot, skills hub, and community tooling.

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
go build -o bin/reasonix-mcpbridge ./cmd/reasonix-mcpbridge
go build -o bin/reasonix-memoryserver ./cmd/reasonix-memoryserver
go build -o bin/reasonix-bot ./bot
go build -o bin/reasonix-hooks ./cmd/reasonix-hooks

# Install skills via upstream install_source
reasonix install-source install --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills

# Build the Desktop app (Wails + React 19 + frontend)
cd desktop/frontend && npm install && cd ../..
cd desktop && wails build -o ../bin/reasonix-desktop

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
cmd/reasonix-pr-review/ PR review CLI for GitHub Actions
internal/              Reasonix engine (75 packages)
  agent/               Core agent loop, compaction, subagents
  boot/                Controller assembly, tool wiring
  bot/                 Multi-platform bot gateway (Discord/QQ/Feishu/WeChat/Telegram)
  collab/              Live collaboration WebSocket hub
  compress/            Tool output token compressor
  config/              TOML config loader + model fallback
  constitution/        Structured project invariants (.reasonix/constitution.json)
  control/             Transport-agnostic Controller
  eval/                Session comparison and evaluation (Jaccard, structural diff)
  learn/               Self-improving skill loops (pattern detection)
  mesh/                Agent-to-agent MCP mesh (delegate, broadcast, council, judge)
  orchestrate/         Multi-agent orchestration (chain, pair, CI-fix workflows)
  permission/          Tool-call permission gating
  plugin/              MCP client (stdio, HTTP, SSE)
  provider/            LLM providers (Anthropic, OpenAI/DeepSeek)
  publish/             Session transcript export (HTML/JSON)
  scheduler/           Cron-driven automated agent tasks
  skill/               Built-in skills registry
  tool/                Built-in tools (bash, read, write, edit, lsp_*, etc.)
  ...
pkg/                   ── Our custom additions ──
  httputil/            Shared Bearer auth middleware
  mcputil/             Shared MCP types and server helpers
cmd/                   ── Our custom binaries ──
  reasonix-mcpbridge/  MCP bridge server (6 tools)
  reasonix-memoryserver/ Hindsight MCP server (3 tools)
  reasonix-pr-review/  PR review CLI
bot/                   Discord bot gateway
cmd/reasonix-hooks/    Native Go hook runner
deploy/                Helm chart + docker-compose for one-click deploy
desktop/               Wails v2 desktop app + React 19 frontend
skills-hub/            17-skill community registry + static catalog site
```

## Our Customizations

| Layer | What | Why |
|-------|------|-----|
| `pkg/mcputil/` + `pkg/httputil/` | Shared Go libraries | Bearer auth middleware + MCP types/helpers |
| `cmd/reasonix-mcpbridge/` | MCP bridge server (6 tools) | Expose Reasonix to Claude Code/Codex via MCP |
| `cmd/reasonix-memoryserver/` | Hindsight memory (3 tools, SQLite, TTL, vector) | Cross-session persistent memory with dense+sparse vector search |
| `bot/` + `internal/bot/discord/` | Discord bot (+ /goal + /model) | Discord integration (upstream has Feishu/WeChat/QQ only) |
| `internal/bot/telegram/` | Telegram bot adapter | Long-polling Telegram integration via go-telegram-bot-api/v5 |
| `internal/bot/line/` | LINE bot adapter | Webhook-based LINE integration via line-bot-sdk-go/v8 |
| `internal/bot/slack/` | Slack bot adapter | Socket Mode Slack integration via slack-go/slack |
| `cmd/reasonix-hooks/` | Native Go hook runner | Zero-dependency binary for PreToolUse/Stop hooks |
| `skills-hub/` | 17 community skills + catalog site | Curated skill registry with frontmatter playbooks |
| `internal/learn/` | Self-improving skill loops | Observes agent patterns, detects repeated sequences, generates skills |
| `internal/mesh/` | Agent-to-agent MCP mesh | Peer delegation, broadcast, council + judge (structured Fusion Router-inspired analysis) |
| `internal/collab/` | Live collaboration hub | WebSocket session sharing between Reasonix instances |
| `internal/compress/` | Tool output token compressor | SHA-256 cache, repeated-line collapsing, JSON minification |
| `internal/scheduler/` | Cron-driven task scheduler | Automated agent runs at scheduled times |
| `internal/publish/` | Session transcript export | Self-contained HTML + JSON export |
| `internal/marketplace/` | Community skill registry | agentskills.io-compatible + LobeHub marketplace sync (360k+ skills) |
| `internal/provider/ollamacloud/` | Ollama Cloud provider | 42 models via ollama.com/v1 OpenAI-compatible API |
| `internal/constitution/` | Project invariants | Structured principles/constraints/rules from .reasonix/constitution.json |
| `internal/e2e/` | Regression testing harness | Replay-based session testing (inputs, turns, assertions) |
| `internal/eval/` | Session comparison tool | Structural diff, Jaccard similarity, CLI + desktop binding |
| `internal/orchestrate/` | Multi-agent orchestration | Chain, pair, and CI-fix workflows (6 tests) |
| `cmd/reasonix-pr-review/` | PR review CLI | Fetches PR diff, runs review with 6-dimension prompt |
| `npm/hermes/` | npm package | One-line install: `npm i -g reasonix-hermes` (7 sub-packages) |
| `deploy/` | Helm chart + docker-compose | One-command deploy to K8s or $5 VPS |
| `desktop/` | Wails v2 desktop app | React 19 frontend + Go kernel; Hermes dashboard; live data push |
| `.reasonix/skills/research/` (+4 siblings) | 5-skill research workflow | `/research` → `/research-deep` → `/research-report` pipeline with SearXNG (discovery) + Crawl4AI (extraction) |
| `internal/cli/eval.go` | `/eval` slash command | Eval-driven development: define, check, report, list, clean subcommands |
| `reasonix-hermes.json` | Install source manifest | `reasonix install-source install --source ...` |
| `.github/workflows/ci-hermes.yml` | Supplementary CI | Desktop frontend build + Hermes package tests in CI |
| `.github/workflows/pr-review.yml` | PR review action | Auto-reviews PRs with Reasonix |
| `.github/workflows/release-hermes-npm.yml` | npm release pipeline | Cross-compiles 6 platforms → npm publish |

## Docs

- **[Changelog](docs/CHANGELOG-HERMES.md)** — Hermes fork milestones, expansion packs, bot platforms
- **[Ecosystem Reference](reasonix-deepseek-ecosystem-2026.md)** — full landscape: MCP bridges, skills, desktop, IDE, forks, cost model, protocols, use cases

## Notes

- Upstream remote: `https://github.com/esengine/deepseek-reasonix.git` (branch `main-v2`)
- **Upstream target**: v1.8.x (July 2026) — ✅ synced (a2709fc). 165 commits merged across 6 syncs.
- Our fork: `https://github.com/aliatx2017/reasonix-hermes.git` (branch `main`)
- To pull upstream updates: `git fetch upstream && git merge upstream/main-v2`
- `reasonix.toml` is gitignored (upstream convention) — never commit secrets
- Discord bot uses `github.com/bwmarrin/discordgo` (added to go.mod)
- Discord bot must use `control.Controller` like every other frontend — not inline chat history
- **Tests**: ~2,430 tests across 75 packages. `go test ./...`
- **New packages (custom)**: `internal/acp/` (Agent Client Protocol), `internal/learn/` (self-improving skill loops), `internal/mesh/` (agent-to-agent MCP mesh), `internal/collab/` (live collaboration WebSocket hub), `internal/compress/` (tool output token compressor), `internal/scheduler/` (cron-driven tasks), `internal/publish/` (session transcript export), `internal/bot/telegram/`, `internal/bot/line/`, `internal/bot/slack/` (multi-platform bot adapters), `internal/e2e/` (regression testing harness), `internal/marketplace/` (community skill registry + LobeHub sync), `internal/provider/ollamacloud/` (Ollama Cloud API provider), `internal/constitution/` (project invariants), `cmd/reasonix-pr-review/` (PR review CLI), `cmd/e2ebench/` (e2e benchmark tool).
- **CodeWhale features** (10/10 done, 2026-07-04): Shell env hooks, parallel sub-agent batch dispatch, completion sound, harness profiles, constitution system, workshop sidecar, desktop hotbar, external sandbox, Nix flake, Dockerfile.
- **CI & tooling** (2026-07-06): `biome format` check on desktop frontend (105 files), `wails build` CI job, `taplo` TOML lint (CI + pre-commit hook), Go `go-version-file: go.mod` (toolchain 1.26.4), 7-job Hermes CI pipeline all-green.
- **Bug fixes** (2026-07-06): duplicate `price` key in `reasonix.example.toml`, data race in `mockProvider.Stream()`, `TestSaveToScopes` cross-platform fix.
- **New packages**: `internal/constitution/` (structured project invariants from `.reasonix/constitution.json`)
- **New files**: `flake.nix`, `Dockerfile`, `.dockerignore`, `internal/sandbox/remote.go`
- **Config additions**: `[notifications].sound`, `active_profile`, `[profiles.<name>]` blocks, `[sandbox].remote_sandbox_url`, `[sandbox].remote_sandbox_token`
- **2026-07-12 session** — 13 features shipped: Hermes accent theme, live data push (Wails events), token sparkline chart, compaction timeline, checkpoint file preview, Write Mode (Go fs bindings + React editor), memory fact graph, reasonix.example.toml full update, remote sandbox e2e tests, workspace slug fix ($HOME relativization), CLI TUI enhancements (pinned banner, bottom counters, /stats sparkline+compaction+memory+goal). Built CLI (26MB) + desktop (33MB). VS Code fork removed.
- **2026-07-14 session** (Ollama Cloud + aux models + 4 features):
  - **Ollama Cloud provider**: New `ollamacloud` provider kind, 42 models, OpenAI-compatible at ollama.com/v1. `reasoning` field name fix in openai provider.
  - **Auxiliary model routing**: `[agent.auxiliary]` config block — compression/vision/web_extract each take their own provider+model. Agent routes compaction summarizer through compressionProv, vision requests through visionProv when images present. Tested with `deepseek-v4-flash` (compression) + `gemini-3-flash-preview` (vision). **Vision pipeline hardened** (2026-07-15 h14): `classifyRef` now detects arbitrary filesystem images, `visionImageDataURLFromPath` reads non-attachment images, workspace-path fallthrough for images, and `properties` defaulted to `{}` in empty-object schemas for Gemini/Ollama Cloud compatibility.
  - **Desktop collab panel**: Go collab Hub + CollabDashboard binding, React CollabPanel (live badge, watcher count, session list), integrated into live data push + polling.
  - **Multi-model council UI**: Controller mesh integration (SetMesh/Council/MeshStatus), boot.go mesh creation, CLI `/council` command, desktop CouncilPanel.
  - **E2E test harness**: New `internal/e2e/` — Harness, SessionInputs, SessionTools, Analyze, AssertTools/Turns, RunAll. 7 tests.
  - **Skill marketplace**: Community registry (12 skills, agentskills.io-compatible SKILL.md format), `internal/marketplace/` Go package, CLI `reasonix marketplace` command, desktop MarketplacePanel with tag filters + install buttons.
  - **LobeHub marketplace API integration**: Full M2M OAuth2 client at `internal/marketplace/lobehub_client.go` (stdlib-only HS256 JWT), auto-registration, paginated sync from 360k+ community skills at `market.lobehub.com`, desktop "Sync from LobeHub" button, CLI `reasonix marketplace sync`, `[marketplace.lobehub]` config section. Zero new dependencies.
  - **LAN skills**: 4 project skills (`searxng-local`, `crawl4ai-local`, `google-maps-scraper`, `last30days`) for local network services at 192.168.1.214.
  - **Total**: 30+ files changed, 3 new Go packages, 4 new React components, 80+ tests. All binaries rebuilt. Upstream synced to ed07684.
- **2026-07-14 session (h6)** (banner + version + savings stats):
  - **Dynamic version**: `resolveVersion()` in `style.go` — uses ldflags first, then `git describe --tags --match 'v*'`, falls back to `"v1.8.0"` only as last resort. Pinned banner shows live git tag in dev builds.
  - **Diamond Wing logo**: `◆` replaces `⚚` caduceus in pinned header + session banner, gold accent preserved.
  - **Savings stats in status bar**: `aux↓N` (tokens saved via auxiliary providers) + `sqz↓N` (bytes saved by compressor). Atomic counters in compressor, exposed through agent → controller → TUI.
  - **Total**: 6 files changed, +91/-12 lines. Committed f0ba51b.

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

