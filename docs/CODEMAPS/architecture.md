<!-- Generated: 2026-07-06 | Packages: 52 internal + 4 cmd + 5 bot adapters + 2 pkg | Token estimate: ~950 -->

# Architecture — Reasonix Hermes

## Project Type
Monorepo: Go CLI/engine kernel + React/TypeScript desktop frontend (Wails v2) + standalone Go services.

## Top-Level Layout
```
reasonix/                Go module root
├── cmd/reasonix/        CLI entry (→ internal/cli/)
├── cmd/reasonix-mcpbridge/  MCP server: 6 tools
├── cmd/reasonix-memoryserver/ Hindsight MCP: 3 tools (SQLite + vector)
├── cmd/reasonix-hooks/  Native Go hook runner
├── cmd/reasonix-plugin-example/ Plugin skeleton
├── cmd/e2ebench/        E2E benchmarks
├── bot/main.go          Standalone Discord bot binary
├── internal/            52 packages — core engine
├── desktop/             Wails v2 app (Go backend + React 19/TS 6 frontend)
├── pkg/                 Shared libraries (mcputil, httputil)
├── skills-hub/          17 community skills + static catalog site
└── site/                Project website (Astro)
```

## Core Loop (per-turn)
```
User message
  → control.Controller.Compose()       // assemble prompt
  → agent.Agent.RunTurn()              // loop: LLM → parse → tools
  → provider.{OpenAI,Anthropic}        // HTTP/SSE to model API
  → tool.builtin.{bash,read,write,...} // tool execution
  → control.Controller.Respond()       // render response chunks
```

## Three Frontends, One Controller
```
discord.Adapter  → ┐
HTTP/SSE serve   → ├─ control.Controller (transport-agnostic)
Wails desktop    → ┘
```
All behavior lives in `control.Controller`. Frontends only handle I/O.

## Key Design Decisions
- **Cache-first**: System prompt prefix is byte-stable across turns for DeepSeek prefix cache warmth.
- **Provider registry**: `internal/provider/` uses `init()` registration; adding a provider is 1 file.
- **MCP client**: `internal/plugin/` supports stdio, HTTP, SSE transports; lazy/eager/background startup.
- **Sandbox**: macOS Seatbelt + Linux bubblewrap; read-only root, workspace writable, network isolatable.

## Data Flow — Session Lifecycle
```
config.Load() → boot.Build() → agent.New() → control.Controller
    ↓                ↓              ↓               ↓
 reasonix.toml   provider init   tool registry   session state
```
