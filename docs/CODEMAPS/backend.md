<!-- Generated: 2026-06-19 | Files: 368 Go (non-test) + 373 test + 180 TS/TSX | Token estimate: ~950 -->

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
| `cli/` | 18,490 | CLI command routing, chat TUI, /slash commands, eval, learn, models |
| `config/` | 8,299 | TOML loader, edit, migrate, render |
| `agent/` | 7,937 | Agent loop, subagents, compaction, goal mode, workshop |
| `control/` | 7,643 | Session controller: compose, dispatch, respond |
| `bot/` | 5,849 | Multi-platform IM gateway + 7 adapters |
| `tool/builtin/` | 5,114 | 18 built-in tools (init-registered) |
| `plugin/` | 3,144 | MCP client: stdio, HTTP, SSE transports |
| `provider/` | 2,510 | LLM providers: OpenAI, Anthropic, OllamaCloud |
| `boot/` | 1,822 | Controller bootstrap: config → provider → agent → controller |
| `skill/` | 1,720 | Built-in skills registry (7: init, explore, research, install, review, security-review, test) |
| `sandbox/` | — | macOS Seatbelt + Linux bubblewrap + remote OpenSandbox |
| `constitution/` | — | .reasonix/constitution.json invariants |
| `codegraph/` | — | Semantic code index (symbol-level queries) |
| `checkpoint/` | — | Snapshot-based edit safety net |
| `permission/` | — | Tool-call permission gating |
| `netclient/` | — | HTTP client with proxy resolution |
| `memory/` | — | Agent-triggered memory hooks |
| `hook/` | — | PreToolUse/Stop hook execution |
| `installsource/` | — | Skill/MCP install from URL/local/package |
| `history/` | — | Session history search (BM25) |
| `i18n/` | — | Translation engine (en, zh, zh-TW) |
| `agentlog/` | — | Structured operational logging (slog, log rotation) |
| `billing/` | — | Balance tracking + live CNY→USD exchange |
| `collab/` | — | Live collaboration WebSocket hub |
| `compress/` | — | Tool output token compressor (SHA-256, dedup, JSON min) |
| `e2e/` | — | Regression testing harness |
| `eval/` | — | Session comparison (Jaccard, structural diff) |
| `learn/` | — | Self-improving skill loops (pattern detection) |
| `marketplace/` | — | Community skill registry + LobeHub sync |
| `mesh/` | — | Agent-to-agent MCP mesh (delegate, broadcast, council) |
| `orchestrate/` | — | Multi-agent orchestration (chain, pair, CI-fix) |
| `publish/` | — | Session transcript export (HTML/JSON) |
| `scheduler/` | — | Cron-driven agent task scheduler |
| `serve/` | — | HTTP/SSE web UI server |
| `migration/` | — | Config/migration-rescue |
| `mcpdiag/` | — | MCP server diagnostics |
| `acp/` | — | Agent Client Protocol dispatch |
| `proc/` | — | Process management |
| `outputstyle/` | — | Output formatting |

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
