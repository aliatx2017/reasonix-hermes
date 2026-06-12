<!-- Generated: 2026-07-06 | External deps: 174 (go.sum) + Nix + Docker | Token estimate: ~550 -->

# Dependencies — Reasonix Hermes

## Go Dependencies (go.mod — 174 lines in go.sum)

### Core Libraries
| Module | Purpose |
|--------|---------|
| `github.com/BurntSushi/toml` | Config file parsing (v1.4+) |
| `github.com/bwmarrin/discordgo` | Discord gateway/API client (v0.29) |
| `github.com/wailsapp/wails/v2` | Desktop framework (Go ↔ JS bridge) |
| `golang.org/x/net` | HTTP/proxy, websocket (v0.56) |
| `golang.org/x/text` | Unicode/i18n |
| `golang.org/x/sync` | Concurrency primitives |

### Database
| Module | Purpose |
|--------|---------|
| `github.com/mattn/go-sqlite3` | SQLite driver (CGo-free via modernc fallback) |

### MCP / Code Intelligence
| Module | Purpose |
|--------|---------|
| `github.com/mark3labs/mcp-go` | MCP protocol types + server framework |
| Tree-sitter grammars | Go, TypeScript, Python, Rust code parsing |

### Desktop Frontend (desktop/frontend/package.json)
| Library | Purpose |
|---------|---------|
| React 19 | UI framework |
| TypeScript 6 | Type system |
| Vite | Build tooling |
| Wails runtime | JS ↔ Go bridge |

## Nix Packages (flake.nix)
| Package | Description |
|---------|-------------|
| `reasonix` | Main CLI |
| `reasonix-mcpbridge` | MCP bridge server |
| `reasonix-memoryserver` | Hindsight memory server |
| `reasonix-hooks` | Hook runner |
| `reasonix-bot` | Discord bot |
| `reasonix-full` | Meta-package (all binaries) |
| `devShells.default` | Go 1.24 + gopls + golangci-lint + nodejs 22 + pnpm |

`vendorHash = null` → proxy vendor mode via go.sum (fully reproducible on nixos-unstable).

## Docker (Dockerfile)
- Multi-stage: `golang:1.24` build → `gcr.io/distroless/static` runtime
- 5 binaries: reasonix, reasonix-mcpbridge, reasonix-memoryserver, reasonix-hooks, reasonix-bot

## External API Endpoints (reachable at runtime)
| Endpoint | Used By |
|----------|---------|
| `api.deepseek.com` | DeepSeek V4 Flash/Pro models |
| `token-plan-cn.xiaomimimo.com/v1` | MiMo v2.5 Pro (planner) |
| `api.openai.com` | Optional OpenAI models |
| `api.anthropic.com` | Optional Anthropic models |
| Any MCP server URL | Plugin system (stdio/HTTP/SSE) |
| Any OpenSandbox API URL | Remote sandbox backend |

## Build Tools
- `go 1.25+` — compiler
- `gopls` — language server
- `staticcheck`, `golangci-lint` — static analysis
- `pnpm` — frontend package manager
