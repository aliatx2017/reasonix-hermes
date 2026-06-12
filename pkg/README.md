# Hermes shared libraries (`pkg/`)

These packages are importable Go libraries shared by Hermes binaries.
They contain no `package main` — executables live under `cmd/`.

## Packages

### `httputil` — Bearer token auth middleware

```go
import "reasonix/pkg/httputil"
```

- `AuthMiddleware` — HTTP middleware that enforces `Bearer <token>` on all
  paths except `/health`. Uses `crypto/subtle.ConstantTimeCompare`.
- `LoadAPIKey(envVar)` — read the API key from an environment variable.
- Used by: `cmd/reasonix-mcpbridge`, `cmd/reasonix-memoryserver`

### `mcputil` — MCP JSON-RPC server framework

```go
import "reasonix/pkg/mcputil"
```

- `Server` — JSON-RPC 2.0 server with stdio and HTTP transports.
- `Tool` struct — MCP tool schema (name, description, inputSchema).
- `HandleToolCall` — dispatch tool invocations to handler functions.
- Used by: `cmd/reasonix-mcpbridge`, `cmd/reasonix-memoryserver`

## Layout

```
pkg/
├── httputil/
│   ├── auth.go          # Bearer auth middleware
│   └── auth_test.go     # Tests
├── mcputil/
│   ├── server.go         # JSON-RPC server framework
│   └── server_test.go   # Tests
└── README.md            # This file
```

## Hermes binaries

The Hermes fork ships four executable binaries, all under `cmd/`:

| Binary | Location | Purpose |
|--------|----------|---------|
| `reasonix-mcpbridge` | `cmd/reasonix-mcpbridge/` | MCP bridge — expose Reasonix tools to other agents |
| `reasonix-memoryserver` | `cmd/reasonix-memoryserver/` | Hindsight memory — cross-session persistent memory |
| `reasonix-hooks` | `cmd/reasonix-hooks/` | Native Go hook runner |
| `reasonix-bot` | `bot/` | Discord bot gateway |

The main Reasonix CLI and desktop app come from upstream (`cmd/reasonix/` and
`desktop/`).

## Relationship to `internal/`

Go convention: `internal/` packages are not importable by external modules.
Hermes' `pkg/` libraries are importable by any Go code within the `reasonix`
module (including test packages and the desktop app). They intentionally have
no dependencies on `internal/` to stay reusable.
