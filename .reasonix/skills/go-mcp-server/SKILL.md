---
name: go-mcp-server
description: "Build Go MCP (Model Context Protocol) servers — stdio/HTTP transport, JSON-RPC 2.0, concurrency, security, graceful shutdown. Covers patterns from Reasonix-Hermes bridge and memory server implementations."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [go, mcp, server, json-rpc, stdio, http, patterns]
---

# Go MCP Server Patterns

Patterns for building MCP servers in Go, drawn from Reasonix-Hermes's own `cmd/reasonix-mcpbridge/` and `cmd/reasonix-memoryserver/`.

## When to Use

- Building a new MCP server in Go
- Adding tools to an existing Go MCP server
- Debugging MCP transport issues (stdio vs HTTP vs SSE)
- Designing MCP server architecture for production use

## Architecture

### Transport Layer

Three transport options, choose based on deployment:

| Transport | When | Implementation |
|-----------|------|---------------|
| **stdio** | Local subprocess (Claude Code, Reasonix plugin) | `os.Stdin`/`os.Stdout` JSON-RPC |
| **HTTP** | Remote server, REST-friendly | `net/http` handler, Bearer auth |
| **SSE** | Streaming events from server | `text/event-stream`, long-lived connection |

### JSON-RPC 2.0

All transports use the same wire format:

```go
// Request
type Request struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      json.RawMessage `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

// Response
type Response struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      json.RawMessage `json:"id"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *Error          `json:"error,omitempty"`
}
```

### Tool Registration

```go
type Tool struct {
    Name        string     `json:"name"`
    Description string     `json:"description"`
    InputSchema Schema     `json:"inputSchema"`
}

type Server struct {
    tools map[string]ToolHandler
    mu    sync.RWMutex
}

type ToolHandler func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
```

## Concurrency

- **stdio**: Single reader goroutine + single writer goroutine + mutex. JSON-RPC responses must be ordered.
- **HTTP**: Standard `net/http` concurrency — each request in its own goroutine.
- **Batch safety**: If supporting JSON-RPC batches, process sequentially or with result ordering.

## Security

- **Bearer auth** for HTTP transport (see `pkg/httputil/auth.go`)
- **Input validation** — validate tool arguments against InputSchema before execution
- **Timeout** — every tool handler should respect `ctx.Done()`
- **No secrets in logs** — sanitize error messages, never log API keys

## Graceful Shutdown

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigCh
    server.Shutdown(context.Background())
}()

server.Serve()
```

## Testing

- **httptest** for HTTP transport — mock server, test tool calls end-to-end
- **Table-driven tests** for tool handlers — each tool should have success + error cases
- **Race detector** — `go test -race` for concurrency testing

## Reference Implementations

| Component | Path | Notes |
|-----------|------|-------|
| MCP bridge server | `cmd/reasonix-mcpbridge/` | 6 tools, HTTP transport, Bearer auth |
| Memory server | `cmd/reasonix-memoryserver/` | 3 tools, HTTP mode, SQLite backend |
| Shared auth | `pkg/httputil/auth.go` | Bearer token middleware |
| Shared MCP types | `pkg/mcputil/` | JSON-RPC types and server helpers |
| MCP client | `internal/plugin/` | stdio/HTTP/SSE client |

## Related

- Project skill: `native-mcp` — MCP client patterns
- Tool: `install_source` — install MCP servers from URL/source
- `docs/SPEC.md` — MCP transport spec
