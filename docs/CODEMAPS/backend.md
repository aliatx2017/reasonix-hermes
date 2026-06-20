<!-- Generated: 2026-06-20 | Files: 371 Go (non-test) + 378 test + 180 TS/TSX | Token estimate: ~950 -->

# Backend — Reasonix Hermes Engine

## Entry Points (9 binaries)
| Binary | Source | Description |
|--------|--------|-------------|
| `reasonix` | `cmd/reasonix/` | CLI chat, run, setup, bot, doctor, serve, models, marketplace, eval |
| `reasonix-bot` | `bot/` | Multi-platform bot gateway (Discord/Telegram/LINE/Slack/QQ/Feishu/WeChat) |
| `reasonix-mcpbridge` | `cmd/reasonix-mcpbridge/` | MCP bridge server (6 tools) |
| `reasonix-memoryserver` | `cmd/reasonix-memoryserver/` | Hindsight MCP server (SQLite, TTL, dense+sparse vector) |
| `reasonix-hooks` | `cmd/reasonix-hooks/` | Native Go hook runner |
| `reasonix-pr-review` | `cmd/reasonix-pr-review/` | PR review CLI for GitHub Actions |
| `reasonix-e2ebench` | `cmd/reasonix-e2ebench/` | E2E benchmarking tool |
| `reasonix-learner-live-test` | `cmd/reasonix-learner-live-test/` | Learner live e2e test |
| `reasonix-desktop` | `desktop/` | Wails v2 desktop app (Go + React 19) |

## Core Engine (internal/ — 56 packages)

| Package | Lines (non-test) | Concern |
|---------|------------------|---------|
| `cli/` | 18,484 | CLI command routing, chat TUI, /slash commands, eval, learn, models |
| `config/` | 8,365 | TOML loader, edit, migrate, render |
| `agent/` | 8,288 | Agent loop, subagents, compaction, goal mode, workshop |
| `control/` | 8,253 | Session controller: compose, dispatch, respond |
| `bot/` | 5,858 | Multi-platform IM gateway + 7 adapters |
| `tool/builtin/` | 5,114 | 18 built-in tools (init-registered) |
| `plugin/` | 3,202 | MCP client: stdio, HTTP, SSE transports |
| `provider/` | 2,510 | LLM providers: OpenAI, Anthropic, OllamaCloud |
| `boot/` | 1,729 | Controller bootstrap: config → provider → agent → controller |
| `skill/` | 1,720 | Built-in skills registry (7: init, explore, research, install, review, security-review, test) |
| `sandbox/` | 999 | macOS Seatbelt + Linux bubblewrap + remote OpenSandbox |
| `constitution/` | 171 | .reasonix/constitution.json invariants |
| `codegraph/` | 1,003 | Semantic code index (symbol-level queries) |
| `checkpoint/` | 333 | Snapshot-based edit safety net |
| `permission/` | 734 | Tool-call permission gating |
| `netclient/` | — | HTTP client with proxy resolution |
| `memory/` | 1,610 | Agent-triggered memory hooks |
| `hook/` | 950 | PreToolUse/Stop hook execution |
| `installsource/` | — | Skill/MCP install from URL/local/package |
| `history/` | 753 | Session history search (BM25) |
| `i18n/` | 1,685 | Translation engine (en, zh, zh-TW) |
| `agentlog/` | 177 | Structured operational logging (slog, log rotation) |
| `billing/` | 174 | Balance tracking + live CNY→USD exchange |
| `collab/` | 323 | Live collaboration WebSocket hub |
| `compress/` | 386 | Tool output token compressor (SHA-256, dedup, JSON min) |
| `e2e/` | 246 | Regression testing harness |
| `eval/` | 311 | Session comparison (Jaccard, structural diff) |
| `learn/` | 484 | Self-improving skill loops (pattern detection) |
| `marketplace/` | 560 | Community skill registry + LobeHub sync |
| `mesh/` | 678 | Agent-to-agent MCP mesh (delegate, broadcast, council) |
| `orchestrate/` | 195 | Multi-agent orchestration (chain, pair, CI-fix) |
| `publish/` | 298 | Session transcript export (HTML/JSON) |
| `scheduler/` | 449 | Cron-driven agent task scheduler |
| `serve/` | 1,440 | HTTP/SSE web UI server |
| `migration/` | 406 | Config/migration-rescue |
| `mcpdiag/` | 184 | MCP server diagnostics |
| `acp/` | 2,801 | Agent Client Protocol dispatch |
| `proc/` | 267 | Process management |
| `outputstyle/` | 197 | Output formatting |

## Provider Chain
```
Config → ProviderEntry registry → provider.Provider interface
  ├── openai.Provider      (DeepSeek, MiMo, any OpenAI-compatible)
  ├── anthropic.Provider   (Claude models)
  └── ollamacloud.Provider (42 models via ollama.com/v1)
```
Each provider: `RoundTrip() → model response`, `ListModels() → []string`, `Balance() → float64`.

## Tool Registration (18 init-registered)
```
tool.Registry ← init() in internal/tool/builtin/*.go
  18 tools: bash, bgjobs, codeindex, completestep, council_judge,
            delete_range, delete_symbol, edit_file, glob, grep,
            ls, move_file, multi_edit, notebook_edit, read_file,
            todo_write, web_fetch, write_file
```
Plus: dynamic MCP tools from `internal/plugin/` (any MCP server).

## Bot Gateway Pipeline (7 platforms)
```
discord.Adapter    → ┐
telegram.Adapter   → ├─ chan InboundMessage
line.Adapter       → ├─ BotGateway.processMessage()
slack.Adapter      → ├── allowlist check / debounce
qq.Adapter         → ├── slash command dispatch
feishu.Adapter     → └── control.Controller per session
weixin.Adapter     → ┘
```

## Config File (~/.config/reasonix/config.toml)
Sections: `config_version`, `default_model`, `language`, `active_profile`, `[ui]`, `[desktop]` (incl. `[desktop.hotbar]`), `[notifications]`, `[agent]` (incl. `[agent.auxiliary]`), `[[providers]]`, `[tools]`, `[permissions]`, `[sandbox]`, `[network]`, `[[plugins]]`, `[skills]`, `[codegraph]`, `[lsp]`, `[bot]` (incl. `[bot.discord]`, `[bot.telegram]`, `[bot.line]`, `[bot.slack]`), `[schedule]`, `[learn]`, `[mesh]`, `[collab]`, `[marketplace]` (incl. `[marketplace.lobehub]`), `[embedding]`, `[billing]`, `[agentlog]`, `[profiles.<name>]`.
