# Headroom — LLM Context Optimization Proxy

> v0.26.0 · The Context Optimization Layer for LLM Applications

Headroom sits between your agent and the LLM API, transparently compressing
conversation context before it reaches the provider. Compression + caching +
prefix-freeze reduces token usage by **20–92%** depending on session length
and tool output patterns.

## Quick Start

```bash
# Install
pip install headroom-ai[proxy]

# Start the proxy pointing at DeepSeek
headroom proxy \
  --backend anyllm --anyllm-provider openai \
  --openai-api-url https://api.deepseek.com \
  --port 8787

# Wire reasonix
# Add to reasonix.toml:
```

```toml
[[providers]]
name        = "deepseek-headroom"
kind        = "openai"
base_url    = "http://localhost:8787/v1"
model       = "deepseek-v4-pro"
api_key_env = "DEEPSEEK_API_KEY"
```

```bash
# Use it
./bin/reasonix run --model deepseek-headroom "your task"
```

## How It Works

```
reasonix → headroom proxy → DeepSeek API
              │
              ├─ Compress: dedup repeated tool output, minify JSON, collapse lines
              ├─ Cache:   SHA-256 content cache — repeated output → compact refs
              ├─ Freeze:  lock prefix bytes across turns (DeepSeek prefix cache)
              └─ Route:   smart model routing (ContentRouter)
```

### Compression pipeline

1. **Dedup** — repeated lines in bash/tool output collapsed to references (up to 58%)
2. **JSON minification** — null stripping, empty line removal (up to 45%)
3. **SHA-256 content cache** — identical tool outputs → hash reference (up to 92%)
4. **Safe mode** — preserves errors, stack traces, diffs (≥2 error markers → verbatim)

### Prefix cache freeze

DeepSeek charges 97.5% less for cache-hit tokens ($0.025/M vs $1.00/M). Headroom's
`cache` mode freezes prior turns so the byte-identical prefix stays warm across
the session. Combined with reasonix's own cache-stable system prompt, most turns
hit the cache at ~95%+.

## Features

| Feature | Flag | Default | What it does |
|---------|------|---------|--------------|
| Token compression | (on) | on | SHA-256 cache, dedup, minification |
| Cache mode | `--mode cache` | token | Freeze turns for prefix cache |
| Token mode | `--mode token` | ✓ | Aggressive compression, prioritize savings |
| Rate limiting | (on) | on | Prevents 429 errors |
| Memory | `--memory` | off | Learns patterns, saves to MEMORY.md |
| Code-aware | `--code-aware` | off | AST-aware compression (needs `[code]` extra) |
| Telemetry | `--no-telemetry` | on | Anonymous aggregate stats |
| Stateless | `--stateless` | off | No filesystem writes (container mode) |
| Embedding server | `--embedding-server` | off | ONNX embedder sidecar (saves 600MB RSS) |

## CLI Reference

### `headroom proxy` — main command

```
headroom proxy [OPTIONS]

Options:
  --host TEXT               Bind host (default: 127.0.0.1)
  -p, --port INTEGER        Port (default: 8787)
  --mode [token|cache]      Optimization mode (default: token)
  --no-optimize             Passthrough — no compression
  --backend TEXT             API backend: anthropic|bedrock|openrouter|anyllm
  --anyllm-provider TEXT     Provider for anyllm: openai|mistral|groq|ollama
  --openai-api-url TEXT      Upstream OpenAI-compatible API URL
  --anthropic-api-url TEXT   Upstream Anthropic API URL
  --memory                   Enable pattern learning → MEMORY.md
  --learn                    Learn from tool call failures
  --min-evidence INTEGER     Min observations before persisting (default: 5)
  --no-telemetry             Disable anonymous telemetry
  --stateless                No filesystem writes (container/read-only)
  --workers INTEGER          Uvicorn workers (default: 1)
  --limit-concurrency INTEGER Max concurrent connections (default: 1000)
```

### `headroom mcp` — MCP server

```bash
headroom mcp serve     # Start MCP stdio server
```

Registered as `[[plugins]]` in reasonix.toml:
```toml
[[plugins]]
name    = "headroom"
command = ".venv-tools/bin/headroom"
args    = ["mcp", "serve"]
```

Provides three tools:
- `headroom_stats` — session compression stats
- `headroom_compress` — compress content on demand
- `headroom_retrieve` — retrieve original by hash

### Other commands

| Command | Purpose |
|---------|---------|
| `headroom memory list` | List stored memories |
| `headroom memory stats` | Memory statistics |
| `headroom perf` | Analyze proxy performance from logs |
| `headroom init` | Install durable integrations for supported agents |
| `headroom learn` | Learn from past tool call failures |
| `headroom agent-savings` | Render token savings for Codex/Claude/Cursor |
| `headroom capture` | Capture network traffic for analysis |
| `headroom wrap` | Wrap CLI tools through headroom |
| `headroom unwrap` | Undo wrapping |

## Stats & Monitoring

### HTTP endpoints

| Endpoint | Returns |
|----------|---------|
| `GET /health` | Liveness + readiness + version |
| `GET /stats` | Full session stats (JSON) |
| `GET /stats-history` | Durable compression history |
| `GET /metrics` | Prometheus metrics |
| `GET /livez` | Process liveness |
| `GET /readyz` | Traffic readiness |

### Key stats

```bash
curl -s http://localhost:8787/stats | python3 -m json.tool
```

Key fields:
- `summary.api_requests` — total API calls through the proxy
- `summary.compression.total_tokens_before` — tokens before compression
- `summary.compression.total_tokens_saved` — tokens removed
- `summary.compression.avg_compression_pct` — average compression %
- `summary.cost.total_saved_usd` — estimated USD saved
- `summary.cost.savings_pct` — cost reduction %
- `agent_usage.agents[0].savings_percent` — per-agent savings
- `agent_usage.agents[0].models` — per-model request counts

### TUI status line

When the proxy is running, the CLI TUI shows `◈↓N%` in the status bar (tokens saved
percentage) and `◈$N` (estimated USD saved). Falls back silently if the proxy is down.

### Desktop dashboard

The Hermes dashboard has a Headroom widget showing:
- Compression ratio bar
- Tokens saved count
- Estimated cost savings
- Cache hit rate gauge

## Expected Savings

| Scenario | Typical savings | Why |
|----------|----------------|-----|
| Short session (< 5 turns) | 5–15% | Limited repeated content |
| Medium session (10–30 turns) | 20–40% | Tool outputs repeat, patterns emerge |
| Long session (50+ turns) | 40–60% | Deep dedup + cache hits compound |
| Repeated tool runs | 60–92% | SHA-256 cache hits on identical output |
| Bash-heavy workflows | 30–58% | Repeated-line collapsing |
| API responses (JSON) | 25–45% | Minification |

**Cache-mode savings** are additive: DeepSeek charges $0.025/M for cache hits vs
$1.00/M for new tokens — that's 97.5% savings on cache-hit tokens *before*
compression even runs. Headroom's `cache` mode maximizes cache-hit ratio by
freezing prior turns.

## How-Tos

### Start as a daemon (macOS)

```bash
# One-liner background (without any-llm-sdk)
headroom proxy --openai-api-url https://api.deepseek.com --no-telemetry &

# Or with launchd (survives logout) — plist maintained in the repo
# at .reasonix/headroom/com.headroom.proxy.plist (827 bytes, plutil-clean).
# Copy it into place and load it:
cp .reasonix/headroom/com.headroom.proxy.plist ~/Library/LaunchAgents/ && \
launchctl load ~/Library/LaunchAgents/com.headroom.proxy.plist

# Verify
curl -s http://localhost:8787/health | python3 -c \
  "import sys,json; print(json.load(sys.stdin)['status'])"
# → healthy
```

The plist contents for reference:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>com.headroom.proxy</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/alex.maksimchuk/projects/reasonix/.venv-tools/bin/headroom</string>
        <string>proxy</string>
        <string>--openai-api-url</string>
        <string>https://api.deepseek.com</string>
        <string>--no-telemetry</string>
    </array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
    <key>StandardOutPath</key><string>/Users/alex.maksimchuk/Library/Logs/headroom-proxy.log</string>
    <key>StandardErrorPath</key><string>/Users/alex.maksimchuk/Library/Logs/headroom-proxy.err</string>
</dict>
</plist>
```

Manage the daemon:

```bash
launchctl unload ~/Library/LaunchAgents/com.headroom.proxy.plist   # stop
launchctl load ~/Library/LaunchAgents/com.headroom.proxy.plist     # restart
```

> **Note:** The `--backend anyllm --anyllm-provider openai` flags from the upstream
> docs require `any-llm-sdk[all]` to be installed (pip). Without it the proxy silently
> crashes during startup. Omitting `--backend` defaults to anthropic passthrough while
> `--openai-api-url` still handles OpenAI-compatible routing — works fine for DeepSeek.

### Multiple agents through one proxy

```toml
# reasonix.toml — both models through the same proxy
[[providers]]
name        = "deepseek-headroom"
kind        = "openai"
base_url    = "http://localhost:8787/v1"
model       = "deepseek-v4-pro"
api_key_env = "DEEPSEEK_API_KEY"

[[providers]]
name        = "deepseek-flash-headroom"
kind        = "openai"
base_url    = "http://localhost:8787/v1"
model       = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
```

### Check if proxy is running

```bash
curl -s http://localhost:8787/health | python3 -c \
  "import sys,json; d=json.load(sys.stdin); print(d['status'])"
# → healthy
```

### View live savings

```bash
# Watch stats refresh every 5s
watch -n5 'curl -s http://localhost:8787/stats | python3 -c "
import sys,json; d=json.load(sys.stdin)[\"summary\"]
print(f\"Requests: {d[\"api_requests\"]} | Saved: {d[\"compression\"][\"total_tokens_saved_with_cli_filtering\"]} tok | \${d[\"cost\"][\"total_saved_usd\"]:.4f}\")
"'
```

### Reset stats

```bash
# Restart the proxy
kill $(lsof -ti:8787) && headroom proxy ...
```

## Architecture

```
┌──────────┐     ┌─────────────────┐     ┌──────────────┐
│ reasonix │────▶│ headroom proxy  │────▶│ DeepSeek API │
│ (client) │◀────│ :8787           │◀────│ (upstream)   │
└──────────┘     │                 │     └──────────────┘
                 │ • Compression   │
                 │ • SHA-256 cache │
                 │ • CCR           │
                 │ • Rate limiter  │
                 │ • Savings tracker│
                 └─────────────────┘
```

Headroom is a **transparent proxy** — it does not modify the API contract.
Any OpenAI-compatible client can point at `http://localhost:8787/v1` and
headroom optimizes the request before forwarding it upstream. The response
flows back unmodified.

### Key components

- **Compression engine** — Rust core (`headroom-core`) for fast dedup/caching
- **CCR** (Compress-Cache-Retrieve) — proactive context management
- **Smart Router** — routes requests to optimal upstream based on content
- **Savings tracker** — persistent savings history in `~/.headroom/proxy_savings.json`

## Related

- `docs/HOWTO-TOKEN-SAVING.md` — sqz compressor (in-process, not a proxy)
- `docs/TOKEN-SAVINGS-ANALYSIS.md` — token economics deep-dive
- `docs/HERMES-GUIDE.md` §16 — Hermes expansion pack reference
