# Reasonix Hermes — Deep Evaluation (June 10, 2026)

> CTO/Architect review of fork health, code cohesiveness, gaps, bugs, refactor
> opportunities, and mesh integration potential.

---

## Executive Summary

| Dimension | Score | Notes |
|-----------|-------|-------|
| Build health | ✅ A | `go build ./...` clean, zero warnings |
| Test coverage (custom pkg/) | ✅ 85.5% | Race-free, 108 tests pass |
| Upstream sync | ✅ Current | v1.5.0 merged (e5e8f02), clean |
| Architecture coherence | ⚠️ B- | Bot wired correctly; MCP servers orphaned from control loop |
| Code duplication | ⚠️ C+ | JSON-RPC scaffolding duplicated across 2 packages |
| Production readiness | ⚠️ C | No systemd units, no health monitoring, no graceful restart |
| Documentation | ✅ A- | AGENTS.md, REASONIX.md, research docs, impl plan all solid |
| Mesh integration potential | 🟢 HIGH | Natural fit on 3 axes |

**Verdict**: Well-structured fork with solid test coverage and clean upstream
tracking. Custom packages work but are architecturally isolated from upstream
engine — they shell out to `reasonix` CLI or call DeepSeek directly instead of
using `control.Controller`. This is the #1 design debt.

---

## 1. Architecture Analysis

### 1.1 What Exists

```
Your code (3,862 LOC across 4 files):
  pkg/mcpbridge/main.go      (717 LOC) — MCP server, 5 tools
  pkg/memoryserver/main.go   (463 LOC) — MCP server, 3 tools
  bot/main.go                (131 LOC) — Discord bot entry point
  pkg/httputil/auth.go       (76 LOC)  — Bearer auth middleware

Tests:
  pkg/mcpbridge/main_test.go     (936 LOC) — 48 tests
  pkg/memoryserver/main_test.go  (1400 LOC) — comprehensive
  pkg/httputil/auth_test.go      (139 LOC)
```

### 1.2 Architectural Tension

**Bot (bot/main.go)** — correctly wired. Uses `internal/bot.Gateway` +
`internal/bot/discord.Adapter`. Follows upstream architectural invariant: all
frontends go through `control.Controller`. ✅ Good.

**MCP Bridge (pkg/mcpbridge/)** — shells out to `reasonix` CLI binary via
`exec.Command`. Doesn't import `internal/control` or `internal/agent`. Result:

- No access to streaming events (tool progress, thinking tokens)
- 5-minute timeout is the only backpressure mechanism
- `plan_task` and `orchestrate_task` bypass agent entirely, calling DeepSeek raw
- No checkpoint/undo support
- No permission gating
- Can't leverage prefix-cache warmth (each `reasonix run` is a cold session)

**Memory Server (pkg/memoryserver/)** — standalone JSON file store. Doesn't integrate
with upstream's `internal/memory` package or `memory.Queue`. Parallel evolution
that will drift.

### 1.3 Cohesiveness Rating

```
Bot ←→ Upstream engine:     TIGHT (uses Gateway, Adapter, Config)
MCP Bridge ←→ Engine:       LOOSE (exec.Command("reasonix", ...))
Memory Server ←→ Engine:    NONE (independent store, no hooks)
httputil ←→ both servers:   SHARED (good DRY)
```

---

## 2. Bugs & Issues Found

### 2.1 JSON-RPC ID Type Mismatch (P1 — Correctness)

```go
// Both servers:
type jsonRPCRequest struct {
    ID int `json:"id"`
}
```

MCP spec allows `id` to be string, number, or null. Claude Code sends string
IDs (`"msg_01..."`) → unmarshal silently gives `ID: 0` → response has wrong ID
→ client drops it. **Real interop bug.**

**Fix**: Use `json.RawMessage` for ID field, echo back verbatim.

### 2.2 Memory Store Race Condition (P2 — Data Integrity)

```go
func (ms *MemoryStore) Recall(...) []MemoryEntry {
    ms.mu.RLock()
    // ... build results ...
    ms.mu.RUnlock()          // ← released here

    // Increment access counts asynchronously
    if len(results) > 0 {
        go func() {
            ms.mu.Lock()     // ← re-acquired in goroutine
```

Between `RUnlock()` and goroutine's `Lock()`, another `Retain()` could mutate
`ms.entries`. The goroutine iterates stale indices. Not a crash (IDs are
matched), but access counts could miss newly-retained entries with matching IDs.

More critically: goroutine has no error reporting path. If `save()` fails,
nobody knows.

### 2.3 Memory ID Generation Not Unique (P2 — Correctness)

```go
ID: fmt.Sprintf("mem-%d-%d", len(ms.entries)+1, time.Now().Unix())
```

If two `Retain()` calls arrive in same second AND entries slice hasn't grown yet
(can't happen due to mutex, but...) — actually safe due to sequential increment
under lock. However, `len(ms.entries)+1` means IDs are non-monotonic after
deletes (no delete func exists yet, but planned). Use UUID or monotonic counter.

### 2.4 MCP Bridge orchestrateTask Unbounded Parallelism (P2 — Resource)

```go
sem := make(chan struct{}, 3)
```

Hardcoded to 3. If DeepSeek returns 10 steps, 3 `reasonix run` processes
spawn simultaneously. Each is a full agent session that can spawn bash, write
files, etc. No resource isolation.

### 2.5 No Content Validation on Retain (P3 — Data Quality)

```go
content, _ := call.Arguments["content"].(string)
```

Empty string accepted. No max length. Could store megabytes per entry.
`memories.json` grows unbounded.

### 2.6 Recall Matching Too Broad (P3 — UX)

```go
match := query == "" ||
    strings.Contains(strings.ToLower(e.Content), lower) ||
    (e.SessionID == sessionID)
```

If `sessionID` is provided, ALL entries from that session match regardless of
query. Probably not intended — should be `sessionID != "" && e.SessionID == sessionID`.

---

## 3. Code Duplication & DRY Violations

### 3.1 JSON-RPC Scaffolding (HIGH)

Both `pkg/mcpbridge/` and `pkg/memoryserver/` independently define:
- `jsonRPCRequest` struct
- `jsonRPCResponse` struct  
- `jsonRPCError` struct
- `handleMessage()` dispatcher
- `successResp()` / `errorResp()` helpers
- `ServeStdio()` readline loop
- `ServeHTTP()` with signal handling

**Refactor**: Extract `pkg/mcputil/` or `pkg/mcpserver/` shared package.
~150 LOC saved. Single place to fix ID type bug.

### 3.2 Auth Middleware Pattern

Already DRY via `pkg/httputil/`. Good. But `LoadAPIKey` could also validate
key format (minimum length, prefix check).

---

## 4. Missing Features & Gaps

### 4.1 No Tests for Bot Integration

`bot/main.go` has zero test coverage. Gateway integration tested via upstream's
`internal/bot/gateway_test.go`, but our standalone entry point (flag parsing,
config loading, signal handling) is untested.

### 4.2 No Graceful Shutdown for MCP Servers

`ServeStdio()` loops until EOF — no context cancellation, no cleanup. If memory
server is mid-write when killed, `memories.json` could be corrupted (no atomic
write).

### 4.3 No Memory Pruning/TTL

REASONIX.md mentions P3 TTL plan. Currently unbounded growth. 1000 sessions ×
10 memories = 10K entries linear-scanned on every recall.

### 4.4 No Health Probes for Bridge

Bridge has `/health` endpoint but no liveness probe for stdio mode. If reasonix
binary hangs, 5-minute timeout is only recovery.

### 4.5 Skills Hub Not Wired to Upstream

`skills-hub/registry.json` declares 16 skills with `runAs` semantics, but
there's no code that loads this registry into upstream's `internal/skill/`
resolver. Skills are markdown files — not auto-loaded.

### 4.6 CI Doesn't Test Custom Packages Explicitly

`ci-hermes.yml` runs `go test -race ./...` which covers everything, but doesn't
have coverage gates, benchmark regression, or golangci-lint.

---

## 5. Refactor Opportunities

### 5.1 MCP Server Framework (HIGH IMPACT)

Extract shared MCP server scaffold:

```go
// pkg/mcputil/server.go
type Server struct {
    Name    string
    Version string
    Tools   []Tool
}

func (s *Server) ServeStdio() error { ... }
func (s *Server) ServeHTTP(addr string, auth *httputil.AuthMiddleware) error { ... }
```

Both bridge and memory server become ~200 LOC each (just tool handlers).

### 5.2 Bridge → In-Process Agent (HIGH IMPACT)

Replace `exec.Command("reasonix", ...)` with:

```go
import "reasonix/internal/control"

ctrl := control.New(control.Options{...})
ctrl.Send(ctx, task)
```

Benefits:
- Prefix-cache reuse across calls (warm session)
- Streaming events back to MCP client
- Permission gating
- Checkpoint support
- No subprocess overhead

Blocker: `internal/` packages can't be imported by `pkg/`. Two approaches:
1. Move bridge into `internal/mcpbridge/` (loses standalone binary)
2. Expose thin API in `pkg/controlapi/` that wraps internal (better)

### 5.3 Memory Server → SQLite Backend (MEDIUM)

Replace `memories.json` with SQLite:
- FTS5 for recall queries (keyword search in microseconds vs linear scan)
- Bounded storage with VACUUM
- Concurrent-safe without goroutine hacks
- TTL via `WHERE created_at > ?`

~200 LOC with `modernc.org/sqlite` (no CGO).

### 5.4 Unified Config Loading

Bridge reads env vars directly. Memory server hardcodes `~/.reasonix/hindsight-memory`.
Neither reads `reasonix.toml`. Upstream config package already supports
plugin/provider resolution — wire into it.

---

## 6. Security Observations

| Finding | Severity | Location |
|---------|----------|----------|
| Auth optional (empty key = no auth) | Low | Both servers |
| No rate limiting on tools/call | Medium | Both servers |
| orchestrate_task spawns unbounded processes | Medium | mcpbridge |
| No input sanitization on task strings | Low | mcpbridge (task → shell arg) |
| memories.json world-readable (0644) | Low | memoryserver |

The `task` string passed to `reasonix run` is safe (Go's `exec.Command` doesn't
invoke shell). But `plan_task`/`orchestrate_task` pass user input to DeepSeek
prompts — indirect injection vector if bridge is internet-facing.

---

## 7. Mesh Integration Opportunities

### 7.1 Reasonix as MCP Plugin for Hermes (IMMEDIATE)

Deploy `reasonix-bridge` on .12 (or .11) as systemd service. Configure in
Hermes `config.yaml`:

```yaml
mcp_servers:
  reasonix:
    command: /opt/reasonix/bin/reasonix-bridge
    transport: stdio
```

OR HTTP mode with auth:

```yaml
mcp_servers:
  reasonix:
    url: http://192.168.1.12:9090/mcp
    transport: http
    headers:
      Authorization: "Bearer ${MCP_API_KEY}"
```

**Value**: Hermes gains DeepSeek-optimized coding agent as delegatable tool.
Cache-first prefix means DeepSeek calls cost ~75% less than cold calls from
Hermes directly.

### 7.2 Hindsight Memory → Shared Redis Backend

Replace file-based `memories.json` with Redis on TrueNAS (:30059). Both Hermes
and Reasonix write to same memory store. Cross-agent knowledge sharing.

```
Hermes session → hindsight_retain("discovered: port 9090 is Reasonix bridge")
Reasonix session → hindsight_recall("ports") → sees Hermes's discovery
```

**Implementation**: Add Redis adapter to memoryserver alongside JSON. Select via
env var `MEMORY_BACKEND=redis` + `REDIS_URL`.

### 7.3 Reasonix Discord Bot → Same Channel as Hermes

Deploy `reasonix-bot` on .12 connected to same Discord server. Separate channel
(#reasonix-coding). Hermes delegates coding tasks:

```
User in #general → "refactor auth module in project X"
Hermes → sends task to Reasonix via MCP bridge
Reasonix → executes, streams progress to #reasonix-coding
Hermes → summarizes result back to user
```

### 7.4 Skills Hub → Hermes Skills Cross-Pollination

Reasonix skills-hub has 16 skills. Some map directly to Hermes workflow:
- `code-review` → could be Hermes `requesting-code-review` alternative
- `deep-research` → parallel research agent
- `council` → multi-perspective decision making
- `security-audit` → pre-deploy security gate

Auto-sync: cron job copies `skills-hub/skills/*.md` → Hermes skill directories
on mesh machines. One source, two consumers.

### 7.5 Kanban Worker with Reasonix Backend

Configure a Hermes kanban worker profile that delegates to Reasonix for
Go-specific tasks. Model tier: Reasonix uses DeepSeek (¥0.14/M cache hit) while
Hermes uses Claude for orchestration. Cost-optimal split.

```
Hermes (Claude/Opus) → orchestration, planning, mesh coordination
Reasonix (DeepSeek V4) → implementation, code changes, testing
```

### 7.6 OmniRoute Integration

Register DeepSeek via OmniRoute on TrueNAS (:20128). Reasonix points
`base_url` at OmniRoute instead of api.deepseek.com. Benefits:
- Request logging
- Fallback routing (if DeepSeek down, route to local model)
- Usage tracking across all mesh agents
- Cost aggregation

---

## 8. Prioritized Action Items

| # | Action | Priority | Effort | Impact |
|---|--------|----------|--------|--------|
| 1 | Fix JSON-RPC ID type (json.RawMessage) | P0 | 1hr | Unblocks Claude Code interop |
| 2 | Extract pkg/mcputil shared scaffold | P1 | 3hr | DRY, single bug-fix surface |
| 3 | Deploy reasonix-bridge on .12 as MCP plugin | P1 | 2hr | Mesh coding delegation |
| 4 | Add Redis backend to memoryserver | P2 | 4hr | Cross-agent memory sharing |
| 5 | Wire skills-hub into upstream skill resolver | P2 | 3hr | Skills actually load |
| 6 | Replace exec.Command with in-process control | P2 | 8hr | Warm cache, streaming, permissions |
| 7 | SQLite backend for memoryserver | P3 | 4hr | Scale + FTS5 search |
| 8 | Add bot/main.go tests | P3 | 2hr | Coverage completeness |
| 9 | Systemd units + health monitoring | P3 | 2hr | Production deployment |
| 10 | Rate limiting on HTTP transport | P3 | 1hr | Security hardening |

---

## 9. Comparison: Reasonix vs Hermes Agent

| Dimension | Reasonix | Hermes |
|-----------|----------|--------|
| Language | Go (single binary, zero deps) | Python (venv, many deps) |
| LLM default | DeepSeek V4 (¥0.14 cache hit) | Claude (via custom providers) |
| Cache strategy | Prefix cache (byte-stable system prompt) | Prompt cache (provider-managed) |
| Tools | 12 built-in + MCP plugins | 50+ built-in + MCP + skills |
| Memory | File-based per-project | Provider-injected + session DB |
| Multi-agent | Subagent with transcript continue | delegate_task + kanban workers |
| Desktop | Wails v2 (Go+React) | Ink TUI + platform gateways |
| Bot platforms | Discord, Feishu, Weixin, QQ | Discord, Telegram, Slack, SMS |
| Sandbox | macOS Seatbelt, network control | None (relies on OS permissions) |
| Differentiator | Cost (DeepSeek prefix cache) | Mesh (5 machines, cross-agent) |

**Complementary, not competing.** Reasonix excels at cheap, fast code changes.
Hermes excels at orchestration, mesh coordination, multi-platform delivery.
Together: Hermes plans, Reasonix executes.

---

## 10. Conclusion

Fork is in good shape. Clean build, solid test coverage, proper upstream
tracking. Primary architectural debt is MCP bridge's shell-out pattern — it
works but leaves performance and integration on the table. Memory server needs
backend upgrade before scaling.

Mesh integration is straightforward and high-value. Deploy bridge as MCP plugin
(2hr), add Redis memory backend (4hr), and you have a DeepSeek-powered coding
worker accessible from any Hermes session on any machine.

---

*Authored by Theshire, 2026-06-10. Next review after upstream v1.6.0 merge.*
