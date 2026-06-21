<!-- Generated: 2026-06-20 | Packages: 56 internal + 8 cmd + 7 bot adapters + 2 pkg | Token estimate: ~950 -->

# Architecture — Reasonix Hermes

## Project Type
Monorepo: Go CLI/engine kernel + React/TypeScript desktop frontend (Wails v2) + standalone Go services.

## Top-Level Layout
```
reasonix/                Go module root
├── cmd/reasonix/        CLI entry (→ internal/cli/) — also builds reasonix-bot
├── cmd/reasonix-mcpbridge/  MCP server: 6 tools
├── cmd/reasonix-memoryserver/ Hindsight MCP: 3 tools (SQLite + dense/sparse vector)
├── cmd/reasonix-hooks/  Native Go hook runner
├── cmd/reasonix-pr-review/  PR review CLI for GitHub Actions
├── cmd/e2ebench/   E2E benchmarking tool
├── cmd/learner-live-test/ Learner live e2e test
├── cmd/reasonix-plugin-example/ Plugin skeleton
├── bot/main.go          Standalone multi-platform bot binary
├── internal/            56 packages — core engine + Hermes extensions
├── desktop/             Wails v2 app (Go backend + React 19/TS 6 frontend)
├── pkg/                 Shared libraries (mcputil, httputil)
├── skills-hub/          17 community skills + static catalog site
├── deploy/              Helm chart + docker-compose
└── npm/hermes/          npm package: `npm i -g reasonix-hermes`
```

## Core Loop (per-turn)
```
User message
  → control.Controller.Compose()       // assemble prompt
  → agent.Agent.RunTurn()              // loop: LLM → parse → tools
  → provider.{OpenAI,Anthropic,OllamaCloud} // HTTP/SSE to model API
  → tool.builtin.{18 tools}            // tool execution
  → control.Controller.Respond()       // render response chunks
```

## Three Frontends, One Controller
```
CLI TUI (chat)    → ┐
HTTP/SSE serve    → ├─ control.Controller (transport-agnostic)
Wails desktop     → ┘
Bot gateway (7 IM) → ┘  (Discord, Telegram, LINE, Slack, QQ, Feishu, WeChat)
```
All behavior lives in `control.Controller`. Frontends only handle I/O.

## Key Design Decisions
- **Cache-first**: System prompt prefix is byte-stable across turns for DeepSeek prefix cache warmth.
- **Provider registry**: `internal/provider/` uses `init()` registration; adding a provider is 1 file.
- **MCP client**: `internal/plugin/` supports stdio, HTTP, SSE transports; lazy/eager/background startup.
- **Sandbox**: macOS Seatbelt + Linux bubblewrap + remote OpenSandbox; read-only root, workspace writable.
- **Auxiliary routing**: Separate providers for compression/vision/web-extract (cost optimization).
- **Multi-agent mesh**: Agent-to-agent MCP delegation, broadcast, and council judge.
- **Self-improving**: Learner observes patterns → suggests skills; instinct-based continuous learning.

## Data Flow — Session Lifecycle
```
config.Load() → boot.Build() → agent.New() → control.Controller
    ↓                ↓              ↓               ↓
 reasonix.toml   provider init   tool registry   session state
```

## Hermes Custom Packages (14 new)
```
acp/ agentlog/ billing/ collab/ compress/ constitution/
e2e/ eval/ learn/ marketplace/ mesh/ orchestrate/
publish/ scheduler/
```
Plus: `provider/ollamacloud/`, `cmd/reasonix-pr-review/`, `cmd/e2ebench/`, `cmd/learner-live-test/`.
