<!-- Generated: 2026-06-19 | go.sum: 193 lines | desktop go.sum: 165 lines | Nix + Docker | Token estimate: ~600 -->

# Dependencies — Reasonix Hermes

## Go Dependencies (go.mod — 193 lines in go.sum)

### Core Libraries
| Module | Purpose |
|--------|---------|
| `github.com/BurntSushi/toml` | Config file parsing (v1.4+) |
| `github.com/gdamore/tcell/v2` | CLI TUI rendering |
| `github.com/wailsapp/wails/v2` | Desktop framework (Go ↔ JS bridge) |
| `golang.org/x/net` | HTTP/proxy, websocket (v0.56) |
| `golang.org/x/text` | Unicode/i18n |
| `golang.org/x/sync` | Concurrency primitives |

### Bot Platform SDKs
| Module | Purpose |
|--------|---------|
| `github.com/bwmarrin/discordgo` | Discord gateway/API client (v0.29) |
| `github.com/go-telegram-bot-api/telegram-bot-api/v5` | Telegram long-polling client |
| `github.com/line/line-bot-sdk-go/v8` | LINE webhook-based messaging |
| `github.com/slack-go/slack` | Slack Socket Mode client (v0.26) |

### Database
| Module | Purpose |
|--------|---------|
| `github.com/mattn/go-sqlite3` | SQLite driver (CGo-free via modernc fallback) |

### MCP / Code Intelligence / WebSocket
| Module | Purpose |
|--------|---------|
| `github.com/mark3labs/mcp-go` | MCP protocol types + server framework |
| Tree-sitter grammars | Go, TypeScript, Python, Rust code parsing |
| `github.com/gorilla/websocket` | WebSocket (collab hub) |

### Desktop Frontend (desktop/frontend/package.json)
| Library | Purpose |
|---------|---------|
| React 19 | UI framework |
| TypeScript 6 | Type system |
| Vite | Build tooling |
| Wails runtime | JS ↔ Go bridge |
| CodeMirror 6 | Write Mode markdown editor |
| D3.js | Memory fact force-directed graph |

## Nix Packages (flake.nix)
| Package | Description |
|---------|-------------|
| `reasonix` | Main CLI |
| `reasonix-mcpbridge` | MCP bridge server |
| `reasonix-memoryserver` | Hindsight memory server |
| `reasonix-hooks` | Hook runner |
| `reasonix-full` | Meta-package (all binaries) |
| `devShells.default` | Go 1.25 + gopls + golangci-lint + nodejs 22 + pnpm |

`vendorHash = null` → proxy vendor mode via go.sum (fully reproducible on nixos-unstable).

## Docker (Dockerfile)
- Multi-stage: `golang:1.25-bookworm` build → `gcr.io/distroless/static` runtime
- 9 binaries: reasonix, reasonix-bot, reasonix-mcpbridge, reasonix-memoryserver, reasonix-hooks, reasonix-pr-review, reasonix-e2ebench, reasonix-learner-live-test, reasonix-desktop

## External API Endpoints (reachable at runtime)
| Endpoint | Used By |
|----------|---------|
| `api.deepseek.com` | DeepSeek V4 Flash/Pro models |
| `token-plan-cn.xiaomimimo.com/v1` | MiMo v2.5 Pro (planner) |
| `ollama.com/v1` | Ollama Cloud (42 models, OpenAI-compatible) |
| `api.openai.com` | Optional OpenAI models |
| `api.anthropic.com` | Optional Anthropic models |
| `market.lobehub.com` | LobeHub marketplace API (M2M OAuth2) |
| `api.exchangerate-api.com` | Live CNY→USD exchange rate |
| Any MCP server URL | Plugin system (stdio/HTTP/SSE) |
| Any OpenSandbox API URL | Remote sandbox backend |

## Build Tools
- `go 1.25+` (toolchain go1.26.4) — compiler
- `gopls` — language server
- `staticcheck`, `golangci-lint` — static analysis
- `pnpm` — frontend package manager
- `wails` — desktop build tool (~/go/bin/wails)
- `taplo` — TOML linter (CI + pre-commit hook)
- `biome` — frontend formatter
