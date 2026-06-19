# Reasonix Hermes Project

> A DeepSeek-native AI coding agent — forked and extended for the community.

**Reasonix Hermes** is an extended fork of [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) (synced to v1.9.x). We build on upstream's config-driven, plugin-driven Go core and add cross-agent connectivity, persistent memory, multi-platform bot adapters, and community tooling.

- **Repo**: <https://github.com/aliatx2017/reasonix-hermes>
- **Upstream**: <https://github.com/esengine/deepseek-reasonix> (branch `main-v2`)
- **Module**: `reasonix`
- **Stack**: Go 1.25 (CLI + backend), React 19 + TypeScript (desktop frontend), Wails v2 (desktop shell)
- **License**: MIT

---

## Quick Start

### One-line (npm)

```sh
npm i -g reasonix-hermes
```

This pulls the prebuilt binary for your platform and adds `reasonix-hermes` (and `reasonix`) to your PATH.

### Prebuilt binaries

Binaries are available for macOS (amd64 + arm64), Linux (amd64 + arm64), and Windows (amd64) — attach them directly from the [GitHub Releases](https://github.com/aliatx2017/reasonix-hermes/releases) page.

| Binary | Size | Purpose |
|--------|------|---------|
| `reasonix` | ~27 MB | Core CLI — chat, run, serve, setup |
| `reasonix-desktop` | ~34 MB | Wails desktop app |
| `reasonix-bot` | ~15 MB | Discord, Telegram, LINE, Slack bot gateway |
| `reasonix-mcpbridge` | ~9 MB | MCP bridge server (6 tools) |
| `reasonix-memoryserver` | ~14 MB | Hindsight memory server (SQLite + vector search) |
| `reasonix-hooks` | ~8 MB | Native Go hook runner |
| `reasonix-pr-review` | ~9 MB | PR review CLI for GitHub Actions |
| `reasonix-e2ebench` | ~9 MB | E2E benchmarking tool |

### Build from source

```sh
git clone https://github.com/aliatx2017/reasonix-hermes.git
cd reasonix-hermes

# Core CLI
go build -o bin/reasonix ./cmd/reasonix

# Hermes services
go build -o bin/reasonix-mcpbridge  ./cmd/reasonix-mcpbridge
go build -o bin/reasonix-memoryserver     ./cmd/reasonix-memoryserver
go build -o bin/reasonix-bot        ./bot
go build -o bin/reasonix-hooks      ./cmd/reasonix-hooks
go build -o bin/reasonix-pr-review     ./cmd/reasonix-pr-review
go build -o bin/reasonix-e2ebench      ./cmd/e2ebench

# Desktop app
cd desktop && wails build -o ../bin/reasonix-desktop
```

All binaries are CGO-free and cross-compile to six targets.

---

## Architecture

```
Command-line frontends          HTTP frontends            Desktop frontend
┌──────────────┐    ┌─────────────────────┐    ┌──────────────────────┐
│  reasonix    │    │  reasonix serve     │    │  reasonix-desktop    │
│  chat / run  │    │  reasonix bot       │    │  (Wails + React 19)  │
│  / setup     │    │  (SSE streaming)    │    │                      │
└──────┬───────┘    └──────────┬──────────┘    └──────────┬───────────┘
       │                       │                          │
       └──────────────┬────────┴──────────────┘
                      │
             ┌────────▼────────┐
             │    Controller   │  ← transport-agnostic core
             │  (internal/)    │
             └────────┬────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
   ┌────▼────┐  ┌────▼────┐  ┌────▼────┐
   │ Agent   │  │  Tools  │  │  MCP    │
   │ loop    │  │ builtin │  │ plugins │
   └─────────┘  └─────────┘  └─────────┘
```

### Go packages (71 internal packages)

| Area | Packages | Purpose |
|------|----------|---------|
| **Core** | `agent/`, `control/`, `config/`, `boot/` | Agent loop, transport-agnostic controller, TOML config, boot wiring |
| **Providers** | `provider/openai/`, `provider/anthropic/`, `provider/ollamacloud/` | LLM backends — DeepSeek, Anthropic, Ollama Cloud |
| **Tools** | `tool/builtin/`, `tool/lsp/`, `tool/lsp_ext/` | Bash, read, write, edit, LSP integration, file tree |
| **MCP** | `plugin/`, `mcp/` | MCP client (stdio/HTTP/SSE) |
| **Permissions** | `permission/` | Per-call allow/ask/deny rules with glob matching |
| **Bot adapters** | `bot/discord/`, `bot/telegram/`, `bot/line/`, `bot/slack/`, `bot/` | Multi-platform gateway — Discord, Telegram, LINE, Slack |
| **Memory** | `publish/`, `compress/`, `scheduler/` | Session export (HTML/JSON), tool output compressor, cron scheduler |
| **Mesh/Collab** | `mesh/`, `collab/` | Agent-to-agent MCP mesh, live collaboration WebSocket hub |
| **Skills** | `skill/`, `marketplace/`, `learn/` | Built-in skills, community registry, self-improving skill loops |
| **Evaluation** | `eval/`, `e2e/`, `orchestrate/` | Session comparison, e2e harness, multi-agent workflows |
| **Constitution** | `constitution/` | Structured project invariants from `.reasonix/constitution.json` |
| **Observability** | `agentlog/`, `billing/` | Structured JSON agent logging with log rotation; live CNY→USD exchange rate for cost display |

---

## Hermes Customizations

### Bot adapters

| Platform | Adapter | Protocol | Features |
|----------|---------|----------|---------|
| **Discord** | `internal/bot/discord/` | Gateway + slash commands | `/goal` autonomous loop, `/model` switching, approval flow, multi-platform gateway |
| **Telegram** | `internal/bot/telegram/` | Long-polling | DMs, groups, supergroups, media extraction, message splitting |
| **LINE** | `internal/bot/line/` | Webhook | Webhook server, reply API, paragraph-aware splitting |
| **Slack** | `internal/bot/slack/` | Socket Mode | DMs, @mentions, message splitting at paragraph boundaries |

### Desktop enrichment (19 features)

The Wails desktop app adds a Hermes dashboard with live data push, token sparkline chart, compaction timeline, checkpoint file preview, Write Mode (split-pane Markdown editor with live preview), memory fact graph, cache economy gauge, hindsight memory dashboard, Discord bot monitor, goal progress widget, skills hub browser, sub-agent task tree, constitution health check, hotbar (Ctrl+1-7 keyboard shortcuts), and named profiles for fast/cheap vs. deep reasoning.

See the [Desktop Guide](./DESKTOP.md) for full details.

### Community integration

| Feature | Package / Binary | What it does |
|---------|-----------------|-------------|
| **MCP bridge server** | `cmd/reasonix-mcpbridge/` | 6 tools — connect Claude Code, Codex to Reasonix over stdio/HTTP |
| **Hindsight memory** | `cmd/reasonix-memoryserver/` | 3 tools — SQLite + TF-IDF vector search + dense embeddings, TTL/importance decay |
| **Skills hub** | `skills-hub/` | 17 curated community skills with frontmatter playbooks |
| **Skill marketplace** | `internal/marketplace/` | Community registry + LobeHub sync (360k+ skills) |
| **PR review CLI** | `cmd/reasonix-pr-review/` | GitHub Action with 6-dimension review prompt |
| **Native hooks** | `cmd/reasonix-hooks/` | Zero-dependency Go binary for PreToolUse/Stop hooks |
| **Helm chart + docker-compose** | `deploy/` | One-command deploy to K8s or $5 VPS |
| **npm package** | `npm/hermes/` | `npm i -g reasonix-hermes` — cross-platform installer |

### Expansion packs

| Feature | Package | Description |
|---------|---------|-------------|
| Self-improving skill loops | `internal/learn/` | Pattern detection → automated SKILL.md generation |
| Agent-to-agent MCP mesh | `internal/mesh/` | Peer delegation, broadcast, council mode |
| Live collaboration | `internal/collab/` | WebSocket session sharing between instances |
| Session publishing | `internal/publish/` | Session transcript export (HTML/JSON) |
| Tool output compressor | `internal/compress/` | SHA-256 cache, dedup, JSON minification |
| Cron scheduler | `internal/scheduler/` | Automated agent runs at scheduled times |
| Session comparison | `internal/eval/` | Structural diff + Jaccard similarity |
| Multi-agent orchestration | `internal/orchestrate/` | Chain, pair, and CI-fix workflows |
| Ollama Cloud provider | `internal/provider/ollamacloud/` | 42 models via OpenAI-compatible API |
| Portable mode | — | `REASONIX_PORTABLE=1` — all data in `<binary_dir>/.reasonix/` |

---

## Upstream sync

Hermes tracks [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) (branch `main-v2`). Automated sync via `.github/workflows/sync-upstream.yml` runs daily at 20:00 UTC — clean merge → build+test → push. On conflict, opens a PR for manual resolution.

**Current upstream target**: v1.9.x (commit f944dfb, 2026-06-17).

```sh
git fetch upstream
git merge upstream/main-v2
# resolve conflicts
git push origin main
```

---

## Further reading

| Doc | What it covers |
|-----|---------------|
| **[Guide](./GUIDE.md)** | Configuration, permissions & sandbox, plugins (MCP), slash commands |
| **[Spec](./SPEC.md)** | Engineering contract — architecture, registries, data types |
| **[Hermes Guide](./HERMES-GUIDE.md)** | Comprehensive Hermes feature guide — 20+ sections |
| **[Desktop App](./DESKTOP.md)** | Wails desktop — Hermes dashboard, write mode, bot connections |
| **[Bot Guide](./BOT_GUIDE.md)** | Connect Discord, Telegram, LINE, Slack, Feishu, WeChat, QQ bots |
| **[Marketplace](./MARKETPLACE.md)** | Community skill registry + LobeHub sync |
| **[Constitution](./CONSTITUTION.md)** | Project invariants — principles, constraints, code rules |
| **[Eval](./EVAL.md)** | Compare two agent sessions for eval-driven development |
| **[Checkpoints](./CHECKPOINTS.md)** | Snapshot-based edit safety net |
| **[Changelog](./CHANGELOG-HERMES.md)** | Hermes fork milestones, expansion packs, bot platforms |
| **[Ecosystem Reference](../reasonix-deepseek-ecosystem-2026.md)** | Full landscape: MCP bridges, skills, desktop, IDE, forks, protocols |
| **[Releasing](./RELEASING.md)** | Build, tag, and release process |
| **[Session Memory Retrieval](./SESSION_MEMORY_RETRIEVAL.md)** | Local history + BM25 retrieval |
| **[Session Reference Architecture](./SESSION_REFERENCE_ARCHITECTURE.md)** | Session state routing reference |
