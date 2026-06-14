---
name: native-mcp
description: "MCP client patterns — connect to MCP servers, register their tools, and call them via stdio or HTTP transport. Covers session lifecycle, tool discovery, and error recovery."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [mcp, client, tools, stdio, http, plugin]
---

# Native MCP Client Patterns

Patterns for connecting to MCP servers and using their tools from a Go agent. Reasonix-Hermes implements this in `internal/plugin/` — the MCP host manages server subprocesses and tool registration.

## When to Use

- Adding MCP server support to a tool registry
- Debugging MCP connection issues (server won't start, tools not appearing)
- Understanding how Reasonix discovers and registers MCP tools
- Building a custom MCP client

## Connection Lifecycle

### 1. Server Startup (stdio)

```go
cmd := exec.Command(serverCommand, serverArgs...)
cmd.Env = append(os.Environ(), serverEnv...)
stdin, _ := cmd.StdinPipe()
stdout, _ := cmd.StdoutPipe()
cmd.Start()
```

### 2. Initialize Handshake

```json
// → Request
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}

// ← Response
{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{...}}}
```

### 3. Tool Discovery

```json
// → Request
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}

// ← Response
{"jsonrpc":"2.0","id":2,"result":{"tools":[
  {"name":"tool_name","description":"...","inputSchema":{...}}
]}}
```

### 4. Tool Call

```json
// → Request
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"tool_name","arguments":{...}}}

// ← Response
{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"..."}]}}
```

## Transport Modes

### stdio

- Subprocess launched by Reasonix
- JSON-RPC over stdin/stdout
- Server lifecycle tied to session context
- Best for local tools (filesystem, git, language servers)

### HTTP / SSE

- Remote server at a URL
- Bearer token auth via `Authorization` header
- Stateless or SSE-streaming
- Best for shared services (databases, APIs, cloud tools)

## Error Recovery

- **Server crash**: Restart subprocess, re-initialize, re-register tools
- **Tool timeout**: Cancel via context, server stays alive
- **Protocol error**: Log warning, skip tool, continue session
- **Connection lost**: Retry with backoff (HTTP), restart subprocess (stdio)

## Registration in Reasonix

MCP tools are registered into the tool registry alongside built-in tools:

```
config → [[plugins]] entries → plugin.Host → tool.Registry
```

The `plugin.Host` manages server subprocesses, handles initialization, and adapts remote tools to the `tool.Tool` interface. Hot-added servers (via `/mcp add`) register mid-session.

## Key Files

| Component | Path |
|-----------|------|
| MCP plugin host | `internal/plugin/` |
| Tool registry | `internal/tool/registry.go` |
| Tool interface | `internal/tool/tool.go` |
| MCP types | `pkg/mcputil/` |
| Config entries | `[[plugins]]` in `reasonix.toml` |

## Related

- Project skill: `go-mcp-server` — building MCP servers in Go
- Tool: `install_source` — install MCP servers
- `reasonix.example.toml` — MCP server configuration examples
- `internal/plugin/` — MCP client implementation
