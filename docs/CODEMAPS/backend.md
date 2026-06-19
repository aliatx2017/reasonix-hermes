<!-- Generated: 2026-06-06 | Files: 373 (non-test Go) + 101 TS | Token estimate: ~900 -->

# Backend — Reasonix Hermes Engine

## Entry Points
| Binary | Main | Description |
|--------|------|-------------|
| `reasonix` | `cmd/reasonix/main.go` | CLI chat, run, setup, bot, doctor, serve |
| `reasonix-mcpbridge` | `cmd/reasonix-mcpbridge/main.go` | MCP bridge server (6 tools) |
| `reasonix-memoryserver` | `cmd/reasonix-memoryserver/main.go` | Hindsight MCP server (SQLite, TTL, vector) |
| `reasonix-hooks` | `cmd/reasonix-hooks/main.go` | Shell env hooks runner |
| `reasonix-bot` | `bot/main.go` | Multi-platform bot gateway (Discord/Telegram/LINE/Slack/QQ/Feishu/WeChat) |
| `reasonix-plugin-example` | `cmd/reasonix-plugin-example/main.go` | Plugin skeleton |
| `reasonix-pr-review` | `cmd/reasonix-pr-review/main.go` | PR review CLI for GitHub Actions |
| `reasonix-e2ebench` | `cmd/reasonix-e2ebench/main.go` | E2E benchmarking tool |
| `reasonix-learner-live-test` | `cmd/reasonix-learner-live-test/main.go` | Learner live e2e test |

## Core Engine (internal/ — 69 packages)

| Package | Lines | Concern |
|---------|-------|---------|
| `control/` | 5,557 | Session controller: compose, dispatch, respond |
| `agent/` | 5,153 | Agent loop, subagents, compaction, goal mode, workshop |
| `config/` | 5,046 | TOML loader, edit, migrate, legacy config |
| `tool/builtin/` | 4,185 | 17 built-in tools (bash, read, write, edit, grep, glob…) |
| `bot/` | 3,732 | Multi-platform IM gateway + Discord/Telegram/LINE/Slack/Feishu/QQ/WeChat adapters |
| `plugin/` | 2,647 | MCP client: stdio, HTTP, SSE transports |
| `provider/` | 2,127 | LLM providers: OpenAI-compatible (DeepSeek, MiMo), Anthropic |
| `skill/` | 1,448 | Built-in skills registry, explore/research/review subagents |
| `boot/` | 1,224 | Controller bootstrap: config → provider → agent → controller |
| `sandbox/` | 479 | macOS Seatbelt + Linux bubblewrap + remote OpenSandbox |
| `cli/` | — | CLI command routing, chat TUI, /slash commands |
| `constitution/` | — | .reasonix/constitution.json invariants |
| `codegraph/` | — | Semantic code index (symbol-level queries) |
| `checkpoint/` | — | Snapshot-based edit safety net |
| `permission/` | — | Tool-call permission gating |
| `netclient/` | — | HTTP client with proxy resolution |
| `memory/` | — | Agent-triggered memory hooks |
| `hook/` | — | PreToolUse/Stop hook execution |
| `installsource/` | — | Skill/MCP install from URL/local/package |

## Provider Chain
```
Config → ProviderEntry registry → provider.Provider interface
  ├── openai.Provider   (DeepSeek, MiMo, any OpenAI-compatible)
  └── anthropic.Provider
```
Each provider: `RoundTrip() → model response`, `ListModels() → []string`, `Balance() → float64`.

## Tool Registration
```
tool.Registry ← init() in each internal/tool/builtin/*.go
  17 tools: bash, bgjobs, read_file, write_file, edit_file, multi_edit,
            delete_range, delete_symbol, glob, grep, ls, notebook_edit,
            web_fetch, todo, complete_step, gitignore
```
Plus: dynamic MCP tools from `internal/plugin/` (any MCP server).

## Bot Gateway Pipeline
```
discord.Adapter → chan InboundMessage
feishu.Adapter  → chan InboundMessage     → BotGateway.processMessage()
qq.Adapter      → chan InboundMessage       ├── allowlist check
weixin.Adapter  → chan InboundMessage       ├── debounce/merge
                                             ├── slash command dispatch
                                             └── control.Controller per session
```

## Config File (~/.config/reasonix/config.toml)
Top-level sections: `config_version`, `default_model`, `language`, `active_profile`, `[ui]`, `[desktop]` (incl. `[desktop.hotbar]`), `[notifications]`, `[agent]`, `[[providers]]`, `[tools]`, `[permissions]`, `[sandbox]`, `[network]`, `[[plugins]]`, `[skills]`, `[codegraph]`, `[lsp]`, `[bot]`, `[profiles.<name>]`.
