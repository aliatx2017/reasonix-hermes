# Reasonix Hermes — Action Plan

Consolidated action plan derived from three audits (June–July 2026) and deep architectural analysis. Covers security, concurrency, architecture, operations, and technical debt.

**Codebase snapshot**: 222K lines Go, 766 files, 84 packages, 3,642 tests (all pass), race-clean, zero vulnerabilities (govulncheck).

---

## Table of Contents

1. [Risk Register](#1-risk-register)
2. [Gap Analysis](#2-gap-analysis)
3. [Action Items — Immediate (P0)](#3-action-items--immediate-p0)
4. [Action Items — Short-Term (P1)](#4-action-items--short-term-p1)
5. [Action Items — Medium-Term (P2)](#5-action-items--medium-term-p2)
6. [Action Items — Long-Term (P3)](#6-action-items--long-term-p3)
7. [Architectural Recommendations](#7-architectural-recommendations)
8. [Operational Recommendations](#8-operational-recommendations)
9. [Dependency Map](#9-dependency-map)
10. [Metrics & Success Criteria](#10-metrics--success-criteria)

---

## 1. Risk Register

### 1.1 Security Risks

| ID | Risk | Severity | Likelihood | Impact | Status |
|----|------|----------|------------|--------|--------|
| SR-01 | WebSocket CheckOrigin allows any origin (CSRF) | CRITICAL | High | Cross-site WebSocket hijacking | Open |
| SR-02 | HTTP servers without timeouts (slowloris DoS) | CRITICAL | High | Connection exhaustion, denial of service | Open |
| SR-03 | `http.DefaultClient` without timeout in 4 call sites | HIGH | Medium | Hung API calls block forever | Open |
| SR-04 | Path traversal in memory `doc.go:resolvePath` | HIGH | Medium | Arbitrary file read outside workspace | Open |
| SR-05 | Path traversal in MCP bridge `findSkillFile` | MEDIUM | Medium | Skill file reads outside allowed dirs | Partial (validates `..` but no `filepath.Rel` check) |
| SR-06 | Shell command injection via `eval.go:296` `sh -c` | HIGH | Low | Command injection if eval defs are user-controlled | Open |
| SR-07 | Non-constant-time token comparison in Feishu adapter | MEDIUM | Low | Timing side-channel on verification token | Open |
| SR-08 | Raw `err.Error()` leaked in HTTP responses | MEDIUM | Medium | Internal path/API key leakage to clients | Open |
| SR-09 | Servers bind 0.0.0.0 by default (feishu, collab) | MEDIUM | Medium | Network-exposed endpoints without reverse proxy | Open |
| SR-10 | No authentication on collab WebSocket | HIGH | Low | Unauthenticated session steering | Open |
| SR-11 | No TLS option for MCP HTTP server | MEDIUM | Low | Plaintext bearer tokens on LAN | Open |
| SR-12 | Unbounded `io.ReadAll` in 6 call sites | LOW | Low | OOM on oversized responses | Open |
| SR-13 | File permissions 0o644 on session/config data | LOW | Low | World-readable sensitive files | Open |
| SR-14 | MCP bridge `doctorCheck` reveals API key length | LOW | Low | Length oracle for brute-force | Open |
| SR-15 | No request size limit on MCP HTTP handler | MEDIUM | Medium | Multi-GB body DoS | Open |
| SR-16 | Committed `reasonix.toml` with hardcoded paths | MEDIUM | High | Reveals directory structure, breaks portability | Open |

### 1.2 Concurrency Risks

| ID | Risk | Severity | Likelihood | Impact | Status |
|----|------|----------|------------|--------|--------|
| CR-01 | `context.Background()` in 6+ request paths | CRITICAL | High | Detached goroutines survive shutdown | Open |
| CR-02 | `readFileWithTimeout` goroutine leak under disk pressure | CRITICAL | Low | Slot exhaustion, cascading failures | Open |
| CR-03 | `rateLimit.cleanupLoop` permanent goroutine leak | CRITICAL | High | Permanent leak per server instance | Open |
| CR-04 | Heartbeat `close(done)` without goroutine join | HIGH | Medium | Panic on restart (double close) | Open |
| CR-05 | Fire-and-forget session cleanup goroutines | HIGH | Medium | Partial cleanup on app shutdown | Open |
| CR-06 | `os.Exit(0)` in mcpbridge signal handler bypasses defers | HIGH | Medium | Buffer flush/cleanup skipped | Open |
| CR-07 | BotGateway dispatch goroutines without WaitGroup | HIGH | Medium | No graceful shutdown, unsynchronized reads | Open |
| CR-08 | `lazySpawn.run()` doesn't check `ctx.Err()` before failure | HIGH | Low | Spurious failure recording on cancellation | Open |
| CR-09 | Memory server periodic tidy runs forever (no context) | MEDIUM | High | Goroutine outlives shutdown | Open |
| CR-10 | `App.ctx` unsynchronized read/write | MEDIUM | Low | Race if method called before startup | Open |
| CR-11 | Compressor TOCTOU on `turn` atomic between locks | MEDIUM | Low | Stale turn numbers in cache entries | Open |
| CR-12 | Mesh `initializedPeers` global `sync.Map` never evicts | MEDIUM | Medium | Stale entries accumulate | Open |
| CR-13 | Plugin Host `c.prompts` may be read without lock | MEDIUM | Low | Data race on prompt data | Open |
| CR-14 | Bot gateway `/approve` handler race condition | MEDIUM | Medium | Nil dereference if controller closes mid-approve | Open |
| CR-15 | Compressor `SetTurn` data race (no synchronization) | MEDIUM | Medium | Concurrent read/write on turn field | Open |
| CR-16 | Marketplace `MergeFromLobeHub` modifies without sync | MEDIUM | Low | Data race on entries | Open |
| CR-17 | Collab Hub `Start()` returns nil on bind failure | MEDIUM | Medium | Silent server start failure | Open |

### 1.3 Architectural Risks

| ID | Risk | Severity | Likelihood | Impact | Status |
|----|------|----------|------------|--------|--------|
| AR-01 | `boot.Build()` monolith (1560 lines, 30+ imports) | MEDIUM | High | Fragile to change, hard to test | Open |
| AR-02 | Controller god object (3400+ lines) | MEDIUM | High | High cognitive load, merge conflicts | Mitigated (embedded managers) |
| AR-03 | Model switching rebuilds entire Controller | MEDIUM | Medium | MCP servers restart, latency spike | Partial (`SharedHost` for desktop) |
| AR-04 | Bot gateway spawns full Controller per session | HIGH | Low (no public bots yet) | OOM under high traffic | Open |
| AR-05 | Session as mutable shared state | MEDIUM | Low | Potential races on message slice | Mitigated (sync.Mutex) |
| AR-06 | Memory server file backend is default | MEDIUM | Medium | Write amplification, crash data loss | Open |
| AR-07 | Compressor cache unbounded growth | MEDIUM | Medium | Memory leak in long sessions | Open |
| AR-08 | Scheduler cron parser O(minutes) scan | LOW | Low | Seconds-long parse for rare expressions | Open |
| AR-09 | Mesh re-handshake on every operation | MEDIUM | Medium | 2-3x request overhead | Open |
| AR-10 | Hooks binary always exits 0 | MEDIUM | High | All hook errors masked | Open |
| AR-11 | MCP bridge `plan_task` bypasses Reasonix engine | LOW | High | No tools/permissions/memory | By design |
| AR-12 | Config struct monolith (3400+ lines, 30+ fields) | LOW | Medium | Can't unit-test sections independently | Open |

### 1.4 Operational Risks

| ID | Risk | Severity | Likelihood | Impact | Status |
|----|------|----------|------------|--------|--------|
| OR-01 | CI tests only 3 of 55 packages in Hermes workflow | HIGH | High | 95% of code untested in Hermes CI | Open |
| OR-02 | `govulncheck` is `continue-on-error: true` | MEDIUM | Medium | Vulnerabilities can merge | Open |
| OR-03 | No coverage threshold enforcement | MEDIUM | Medium | Coverage regression undetected | Open |
| OR-04 | Dockerfile hardcoded `GOARCH=amd64` | MEDIUM | Medium | No ARM64 support (Apple Silicon, Graviton) | Open |
| OR-05 | No pprof endpoints in production | MEDIUM | Medium | Can't diagnose CPU/memory issues live | Open |
| OR-06 | No PGO (Profile-Guided Optimization) | LOW | High | Missing 2-7% CPU improvement | Open |
| OR-07 | Action version drift across CI workflows | LOW | Medium | Inconsistent CI behavior | Open |
| OR-08 | Only 5 linters enabled (missing gosec, gocritic) | MEDIUM | Medium | Security/quality issues undetected | Open |
| OR-09 | Helm chart config_version mismatch (1 vs 2) | MEDIUM | Low | Deployed instances use stale schema | Open |
| OR-10 | No HEALTHCHECK in Dockerfile | LOW | Medium | Orchestrators can't determine liveness | Open |
| OR-11 | docker-compose uses `latest` tag, no version pin | LOW | Medium | Breaking changes on pull | Open |
| OR-12 | Memory server `CreatedAt` overwritten by Tidy | MEDIUM | High | All memories show today's date | Open |
| OR-13 | Hooks `session` vs `session_id` parameter mismatch | HIGH | High | Reflect returns all memories, never scoped | Open |

---

## 2. Gap Analysis

### 2.1 Security Gaps

| Gap | Current State | Target State | Effort |
|-----|--------------|--------------|--------|
| WebSocket origin validation | `return true` (allow all) | Configurable allowlist | 1h |
| HTTP server timeouts | 3 servers missing all timeouts | All servers have ReadHeader/Read/Write/Idle timeouts | 2h |
| Timeout on HTTP clients | 4 call sites use `http.DefaultClient` | All use `netclient.DefaultClient()` | 30m |
| Path traversal guards | `resolvePath` allows escape via `../../` | `filepath.Rel` + prefix check on all resolvers | 1h |
| Constant-time comparisons | 1 call site uses `==` for token | All token comparisons use `subtle.ConstantTimeCompare` | 15m |
| Error response sanitization | Raw `err.Error()` in 3 HTTP handlers | Generic message + structured internal logging | 30m |
| Bind address defaults | 0.0.0.0 in 2 servers | 127.0.0.1 default with configurable override | 30m |
| Collab WebSocket auth | No authentication | Token-based auth on upgrade | 2h |
| MCP request size limits | No limit on `io.ReadAll(r.Body)` | `http.MaxBytesReader(w, r.Body, 10<<20)` | 15m |
| Committed secrets/paths | `reasonix.toml` with hardcoded paths | Gitignore `reasonix.toml`, commit only example | 30m |

### 2.2 Testing Gaps

| Gap | Current State | Target State | Effort |
|-----|--------------|--------------|--------|
| Test coverage | 66.5% | 80%+ | 2 weeks |
| `internal/collab/` tests | No tests | Unit + race tests for WebSocket hub | 1 day |
| `internal/scheduler/` tests | Minimal | Cron parser exhaustive tests | 4h |
| Fuzz tests | Zero | JSON/URL/config/SQL parsers fuzzed | 1 day |
| Hermes CI scope | 3/55 packages | `go test -race ./...` | 30m |
| Coverage enforcement | Uploaded but not gated | 70% threshold gate in CI | 30m |

### 2.3 Observability Gaps

| Gap | Current State | Target State | Effort |
|-----|--------------|--------------|--------|
| Mixed logging | `log.Printf` in 4 packages | All `slog` with structured fields | 2h |
| Runtime profiling | No pprof endpoints | Optional pprof behind `--pprof` flag | 1h |
| Build optimization | No PGO file | `default.pgo` from representative workload | 2h |
| API documentation | No OpenAPI spec | `docs/mesh-api.md`, `docs/collab-protocol.md` | 1 day |

### 2.4 Architectural Gaps

| Gap | Current State | Target State | Effort |
|-----|--------------|--------------|--------|
| `boot.Build` decomposition | 1560-line single function | `Builder` struct with phase methods | 1 day |
| Error taxonomy | 4 sentinel errors for 84 packages | Domain-specific error types per subsystem | 1 week |
| Config section parsers | Monolithic `Config` struct | Per-section `Parse*` functions, unit-testable | 2 days |
| Compressor cache eviction | Unbounded `map` | LRU with configurable max size | 2h |
| Mesh peer caching | Re-handshake every operation | `initialized` bool with 5-min TTL on `Peer` | 2h |
| Session immutability | Mutable `[]Message` shared state | Copy-on-write for frontend reads | 4h |
| Bot session pooling | Full Controller per user | Shared provider/plugin host pool | 2 days |

---

## 3. Action Items — Immediate (P0)

Ship-blockers for any new deployment. All items are security or data-integrity critical.

### P0-01: Fix WebSocket CheckOrigin (SR-01)
- **File**: `internal/collab/collab.go:305`
- **Action**: Replace `return true` with origin allowlist validation
- **Mitigation**: Configurable via `[collab].allowed_origins` in config. Default to `["http://localhost:*", "http://127.0.0.1:*"]`
- **Effort**: 1 hour
- **Risk if skipped**: Cross-site WebSocket hijacking

### P0-02: Add HTTP server timeouts (SR-02, SR-03)
- **Files**: `internal/serve/serve.go:277`, `internal/bot/line/line.go:100`, `internal/bot/feishu/feishu.go:709`
- **Action**: Add `ReadHeaderTimeout: 10s`, `ReadTimeout: 30s`, `WriteTimeout: 30s`, `IdleTimeout: 120s` to all `http.Server{}` instances. The non-graceful `Run()` in serve.go should mirror `RunGraceful()`'s timeout config.
- **Mitigation**: Extract a `newConfiguredServer(addr, handler)` helper to enforce timeouts consistently
- **Effort**: 2 hours
- **Risk if skipped**: Slowloris DoS exhausts server file descriptors

### P0-03: Replace `http.DefaultClient` with `netclient.DefaultClient()` (SR-03)
- **Files**: `desktop/write_mode.go:241`, `desktop/bot_connection_app.go:684`, `cmd/reasonix-pr-review/main.go:105,127`, `internal/bot/qq/gateway.go:385`
- **Action**: Swap `http.DefaultClient.Do(req)` to `netclient.DefaultClient().Do(req)` in all 4 call sites
- **Mitigation**: Project already has `netclient.DefaultClient()` with 10s dial, 30s header, 120s overall
- **Effort**: 30 minutes
- **Risk if skipped**: Hung API calls block goroutines forever

### P0-04: Fix path traversal in memory doc resolver (SR-04)
- **File**: `internal/memory/doc.go:250-260`
- **Action**: After resolving path, verify result is within allowed root using `filepath.Rel` + prefix check
- **Mitigation**:
  ```go
  resolved := filepath.Clean(filepath.Join(baseDir, p))
  rel, err := filepath.Rel(baseDir, resolved)
  if err != nil || strings.HasPrefix(rel, "..") {
      return ""
  }
  ```
- **Effort**: 1 hour
- **Risk if skipped**: Arbitrary file read via `../../etc/passwd`

### P0-05: Fix `context.Background()` in request paths (CR-01)
- **Files**: `internal/control/controller.go:787,1628`, `internal/cli/chat_tui.go:3678`, `desktop/bot_connection_app.go:106,150,256,673`, `desktop/workspace_changes.go:143,232`, `internal/bot/feishu/feishu.go:716`
- **Action**: Replace `context.Background()` with parent context (controller lifecycle, app context, or timeout-bounded context) in all 6 subsystems
- **Mitigation**: Each subsystem has an appropriate parent: `c.ctx` for controller, `a.bootContext()` for desktop, `ctx` parameter for bot
- **Effort**: 3 hours
- **Risk if skipped**: Detached goroutines survive shutdown, resource leaks

### P0-06: Fix rateLimit cleanupLoop goroutine leak (CR-03)
- **File**: `internal/serve/auth.go:82`
- **Action**: Add `done chan struct{}` to `rateLimit`, close it in a `Stop()` method, select on `<-rl.done` in the cleanup loop
- **Mitigation**: Wire `Stop()` to server shutdown
- **Effort**: 1 hour
- **Risk if skipped**: Permanent goroutine leak per server instance

### P0-07: Fix hooks parameter mismatch (OR-13)
- **File**: `cmd/reasonix-hooks/main.go:119`
- **Action**: Change `"session"` to `"session_id"` in `doReflect` handler
- **Mitigation**: Aligns with memory server's `hindsight_reflect` expected parameter
- **Effort**: 5 minutes
- **Risk if skipped**: Reflect always returns all memories instead of session-scoped

### P0-08: Fix memory server CreatedAt overwrite (OR-12)
- **File**: `cmd/reasonix-memoryserver/main.go:286-292`
- **Action**: Add `LastDecayAt time.Time` field to entry. Decay based on `now - LastDecayAt`. Preserve `CreatedAt` for display
- **Mitigation**: Migration: set `LastDecayAt = CreatedAt` for existing entries on first load
- **Effort**: 1 hour
- **Risk if skipped**: All memories show today's date after one tidy pass

### P0-09: Fix hooks binary always exits 0 (AR-10)
- **File**: `cmd/reasonix-hooks/main.go`
- **Action**: Use `os.Exit(1)` on error paths instead of `os.Exit(0)`
- **Mitigation**: Calling process can now detect hook failures
- **Effort**: 15 minutes
- **Risk if skipped**: All hook errors silently masked

### P0-10: Expand Hermes CI to all packages (OR-01)
- **File**: `.github/workflows/ci-hermes.yml`
- **Action**: Change `go test ./internal/config/ ./internal/agent/ ./internal/provider/` to `go test -count=1 -race ./...`
- **Mitigation**: Catches regressions across all 83 packages, not just 3
- **Effort**: 30 minutes
- **Risk if skipped**: 95% of code untested in Hermes CI pipeline

---

## 4. Action Items — Short-Term (P1)

Next sprint. Security hardening, concurrency fixes, and critical quality improvements.

### P1-01: Add collab WebSocket authentication (SR-10)
- **Action**: Implement token-based auth on WebSocket upgrade (query param or first-message auth). Validate against configurable token in `[collab]` config section
- **Effort**: 2 hours

### P1-02: Add `io.LimitReader` to unprotected `io.ReadAll` sites (SR-12)
- **Files**: `internal/bot/qq/gateway.go:158,390`, `internal/cli/upgrade.go:243,293,316`, `internal/jobs/jobs.go:640`, `internal/memory/doc.go:127`
- **Action**: Wrap with `io.LimitReader(resp.Body, 10<<20)` (10MB)
- **Effort**: 30 minutes

### P1-03: Tighten file permissions (SR-13)
- **Files**: `desktop/window_state.go:66`, `desktop/workspace.go:46,120,149`, `desktop/write_mode.go:122,127,155`, `desktop/hermes_dashboard.go:917`
- **Action**: Change `0o644` to `0o600` for files containing user data/session state
- **Effort**: 30 minutes

### P1-04: Sanitize HTTP error responses (SR-08)
- **File**: `internal/serve/serve.go:397,409,454`
- **Action**: Log full error via `slog.Error`, return generic `"internal server error"` to client
- **Effort**: 30 minutes

### P1-05: Fix constant-time token comparison (SR-07)
- **File**: `internal/bot/feishu/feishu.go:406-408`
- **Action**: Replace `==` with `subtle.ConstantTimeCompare`
- **Effort**: 15 minutes

### P1-06: Default bind to 127.0.0.1 (SR-09)
- **Files**: `internal/bot/feishu/feishu.go:710`, `internal/collab/collab.go:101`
- **Action**: Change `:port` to `127.0.0.1:port`. Add `--bind` flag for explicit override
- **Effort**: 30 minutes

### P1-07: Add MCP HTTP request size limit (SR-15)
- **File**: `pkg/mcputil/server.go:92`
- **Action**: Add `http.MaxBytesReader(w, r.Body, 10<<20)` before `ReadAll`
- **Effort**: 15 minutes

### P1-08: Fix heartbeat Stop() double-close panic (CR-04)
- **File**: `desktop/heartbeat.go:132`
- **Action**: Add `sync.WaitGroup` to join loop goroutine in `Stop()`. Recreate `done` channel in `Start()`
- **Effort**: 1 hour

### P1-09: Track delayed session cleanup goroutines (CR-05)
- **Files**: `desktop/tabs.go:4117`, `desktop/app.go:1083,1094,1642`
- **Action**: Add `sync.WaitGroup` on `App`, wait during `beforeClose`
- **Effort**: 1 hour

### P1-10: Fix mcpbridge signal handler (CR-06)
- **File**: `cmd/reasonix-mcpbridge/main.go:583`
- **Action**: Replace `os.Exit(0)` with context cancellation to signal `ServeStdio()` to return cleanly
- **Effort**: 1 hour

### P1-11: Fix BotGateway goroutine management (CR-07)
- **File**: `internal/bot/gateway.go:232,237,224-225,242-249`
- **Action**: Add `sync.WaitGroup` for `dispatchLoop`/`evictLoop` goroutines. Guard `gw.adapters` reads with mutex
- **Effort**: 2 hours

### P1-12: Fix bot gateway `/approve` race condition (CR-14)
- **File**: `internal/bot/gateway.go:592-605`
- **Action**: Copy controller pointer while holding lock, or call `Approve()` under lock
- **Effort**: 30 minutes

### P1-13: Fix compressor `SetTurn` data race (CR-15)
- **File**: `internal/compress/compress.go`
- **Action**: Protect `turn` field with `atomic.Int32` (already partially done — ensure consistency)
- **Effort**: 30 minutes

### P1-14: Add compressor cache eviction (AR-07)
- **File**: `internal/compress/compress.go`
- **Action**: Enforce `maxCache` limit in `Compress()`. Evict oldest entries when limit reached
- **Effort**: 2 hours

### P1-15: Fix error wrapping `%s` → `%w` (E5)
- **File**: `internal/control/errmsg.go:20,23`
- **Action**: Replace `%s` + `err.Error()` with `%w` + `err` to preserve error chains
- **Effort**: 15 minutes

### P1-16: Fix byte truncation → rune truncation (E7)
- **Files**: `desktop/bot_connection_app.go:429`, `internal/plugin/plugin.go:1080`, `internal/publish/publish.go`, `internal/orchestrate/orchestrate.go`, `internal/compress/compress.go:414`
- **Action**: Use `clampRunes()` helper (already exists in `errmsg.go:80`) or `[]rune` truncation
- **Effort**: 30 minutes

### P1-17: Fix type assertion without comma-ok (E8)
- **File**: `internal/cli/chat_tui.go:695`
- **Action**: Add comma-ok pattern: `cm, ok := next.(chatTUI); if !ok { return m, nil }`
- **Effort**: 5 minutes

### P1-18: Add memory server `SearchDense` lock downgrade (1.3)
- **File**: `cmd/reasonix-memoryserver/main.go:199-246`
- **Action**: Change `ms.mu.Lock()` to `ms.mu.RLock()` in `SearchDense` (read-only operation)
- **Effort**: 15 minutes

### P1-19: Remove committed `reasonix.toml` with hardcoded paths (SR-16)
- **Action**: Gitignore `reasonix.toml`. Move machine-specific rules to `~/.config/reasonix/config.toml`. Commit only `reasonix.example.toml`
- **Effort**: 30 minutes

### P1-20: Fix MCP bridge argument parsing bug
- **File**: `cmd/reasonix-mcpbridge/main.go:551`
- **Action**: Restructure flag parsing to handle `--http` and `--port` independently
- **Effort**: 30 minutes

---

## 5. Action Items — Medium-Term (P2)

Technical debt, performance, and architectural improvements. Target: 2-4 week horizon.

### P2-01: Decompose `boot.Build()` into Builder struct
- **File**: `internal/boot/boot.go`
- **Action**: Extract into `Builder` struct with phase methods: `buildConfig()`, `buildProvider()`, `buildToolSurface()`, `buildPlugins()`, `buildSubagents()`, `buildMesh()`, `assemble()`
- **Mitigation**: Preserves single assembly point while improving readability and testability
- **Effort**: 1 day
- **Dependency**: None

### P2-02: Implement bot session pooling
- **File**: `internal/bot/gateway.go`
- **Action**: Share provider connections and plugin hosts across bot sessions using `SharedHost` pattern from desktop. Pool controllers with configurable max.
- **Mitigation**: Prevents OOM under high traffic (100+ concurrent IM sessions)
- **Effort**: 2 days
- **Dependency**: AR-03 (SharedHost promotion)

### P2-03: Default memory server backend to SQLite
- **File**: `cmd/reasonix-memoryserver/main.go:689`
- **Action**: Change `--backend` flag default from `file` to `sqlite`. Make file backend explicitly opt-in
- **Mitigation**: SQLite handles concurrent access, crash recovery, and write amplification natively
- **Effort**: 30 minutes
- **Dependency**: None

### P2-04: Add pprof endpoint to production binaries
- **Action**: Add `--pprof <addr>` flag. When set, start pprof HTTP server. Import `_ "net/http/pprof"` behind the flag
- **Effort**: 1 hour
- **Dependency**: None

### P2-05: Increase test coverage to 80%+
- **Focus areas**: `internal/collab/` (zero tests), `internal/scheduler/` (minimal), `cmd/reasonix-memoryserver/` (integration tests needed)
- **Effort**: 2 weeks
- **Dependency**: P0-10 (CI scope)

### P2-06: Add fuzz tests for parsers
- **Targets**: JSON unmarshal in `pkg/mcputil/server.go`, URL parsing in `internal/tool/builtin/webfetch.go`, config parsing in `internal/config/`, SQL query building in `cmd/reasonix-memoryserver/sqlite_storage.go`, cron parser in `internal/scheduler/`
- **Effort**: 1 day
- **Dependency**: None

### P2-07: Cache mesh peer initialization
- **File**: `internal/mesh/mesh.go:233-278`
- **Action**: Add `initialized bool` + `initializedAt time.Time` to `Peer` struct. Skip re-handshake if initialized within 5 minutes. Clear on connection error
- **Mitigation**: Reduces 2-3x request overhead per mesh operation
- **Effort**: 2 hours
- **Dependency**: CR-12 (move `initializedPeers` into Mesh struct)

### P2-08: Add write deadlines to collab WebSocket
- **File**: `internal/collab/collab.go`
- **Action**: Set `peer.conn.SetWriteDeadline(time.Now().Add(5*time.Second))` before each `WriteJSON` call. Remove slow/dead clients on deadline error
- **Effort**: 1 hour
- **Dependency**: None

### P2-09: Separate recall read-path from write-path in memory server
- **File**: `cmd/reasonix-memoryserver/main.go:343-416`
- **Action**: Make `Recall` read-only (no `AccessCount` mutation, no `Importance` boost, no `save()`). Batch access mutations in periodic `Tidy()` pass
- **Mitigation**: Eliminates write amplification on every search query
- **Effort**: 4 hours
- **Dependency**: None

### P2-10: Migrate remaining `log.Printf` to `slog`
- **Files**: `pkg/mcputil/server.go:141,145,149`, `cmd/reasonix-mcpbridge/main.go:587`, `desktop/heartbeat.go:83,121,193,211,227,234`, `cmd/reasonix-memoryserver/embedding.go:106`
- **Action**: Replace with `slog.Info`/`slog.Warn` with structured key-value pairs
- **Effort**: 2 hours
- **Dependency**: None

### P2-11: Enable `gosec` and `gocritic` linters
- **File**: `.golangci.yml`
- **Action**: Add `gosec` (security audit) and `gocritic` (code quality) to enabled linters. Fix flagged issues
- **Effort**: 4 hours (including fixing new findings)
- **Dependency**: None

### P2-12: Make `govulncheck` blocking in CI
- **File**: `.github/workflows/ci.yml:170`
- **Action**: Set `continue-on-error: false`
- **Effort**: 5 minutes
- **Dependency**: None

### P2-13: Add coverage threshold gate in CI
- **File**: `.github/workflows/ci.yml`
- **Action**: Add step that parses coverage output and fails if below 70%
- **Effort**: 30 minutes
- **Dependency**: P2-05

### P2-14: Multi-arch Docker build
- **File**: `Dockerfile:14-20`
- **Action**: Use `FROM --platform=$BUILDPLATFORM` with `$TARGETOS`/`$TARGETARCH` args. Test arm64 builds
- **Effort**: 2 hours
- **Dependency**: None

### P2-15: Add HEALTHCHECK to Dockerfile
- **File**: `Dockerfile`
- **Action**: Add `HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/usr/local/bin/reasonix", "--version"]`
- **Effort**: 15 minutes
- **Dependency**: None

### P2-16: Fix Helm chart issues
- **Files**: `deploy/helm/`
- **Action**: Fix `config_version` (1→2), add security context (`runAsNonRoot`, `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`), fix conditional collab port rendering
- **Effort**: 2 hours
- **Dependency**: None

### P2-17: Replace panic with error returns in non-main code
- **Files**: `internal/serve/auth.go:188,196,527`, `desktop/app.go:222`, `desktop/updater_app.go:149`
- **Action**: Return error instead of panicking on `crypto/rand.Read` failure. Remove `os.Exit(0)` from non-main package
- **Effort**: 1 hour
- **Dependency**: None

### P2-18: Fix multi-error wrapping `%v` → `%w` (Go 1.20+)
- **Files**: `internal/installsource/apply.go:225,238,244`, `internal/tool/builtin/movefile.go:122,128`, `internal/plugin/plugin.go:942`
- **Action**: Use `%w` for both errors: `fmt.Errorf("%w; rollback failed: %w", err, rbErr)`
- **Effort**: 30 minutes
- **Dependency**: None

### P2-19: Add Collab Hub bind-error propagation (CR-17)
- **File**: `internal/collab/collab.go`
- **Action**: Use `net.Listen()` first, then pass listener to `Serve()`. Return bind error to caller
- **Effort**: 1 hour
- **Dependency**: None

### P2-20: Fix Netclient connection pooling (1.13)
- **File**: `internal/netclient/netclient.go:56-61`
- **Action**: Use `sync.Once` to create shared `*http.Client` instead of creating new one per call
- **Effort**: 30 minutes
- **Dependency**: None

---

## 6. Action Items — Long-Term (P3)

Enrichment, optimization, and forward-looking improvements. Target: 1-3 month horizon.

### P3-01: Extract per-section config parsers
- **Action**: Break `Config` struct into composable section types with `Parse*` functions. Enable independent unit testing
- **Effort**: 2 days

### P3-02: Introduce domain-specific error types
- **Action**: Replace 60+ string-matched errors in tests with sentinel errors. Add error types per subsystem (e.g., `PermissionDeniedError`, `ProviderTimeoutError`)
- **Effort**: 1 week

### P3-03: Session copy-on-write for frontends
- **Action**: Snapshot message slice before handing to save/display paths. Prevent mutation during concurrent reads
- **Effort**: 4 hours

### P3-04: Enable PGO for production builds
- **Action**: Collect CPU profile from representative workload (1000-turn session). Save as `default.pgo`. Rebuild. Verify 2-7% improvement
- **Effort**: 2 hours

### P3-05: Analytical cron parser
- **Action**: Replace minute-by-minute scan with O(1)-per-field analytical computation. Or adopt `github.com/robfig/cron/v3`
- **Effort**: 4 hours

### P3-06: Add inverted index to memory server
- **Action**: Use SQLite FTS5 for full-text keyword search. Use ANN for vector search instead of brute-force cosine. Add tag index
- **Effort**: 2 days

### P3-07: Promote `SharedHost` to first-class pattern
- **Action**: Allow model switching to hot-swap only the Provider while preserving session, tool registry, and plugin host. Eliminate MCP server restart on model switch
- **Effort**: 2 days

### P3-08: Add TLS option to MCP HTTP server
- **Action**: Add `ServeHTTPS(addr, certFile, keyFile)` method to `pkg/mcputil`. Document reverse proxy alternative
- **Effort**: 2 hours

### P3-09: Add CORS middleware to MCP HTTP server
- **Action**: Configurable CORS headers for browser-based MCP clients
- **Effort**: 1 hour

### P3-10: Use faster hash for compressor dedup
- **Action**: Replace SHA-256 with xxhash or FNV-1a for content-addressable caching. SHA-256 is cryptographic overkill for dedup
- **Effort**: 1 hour

### P3-11: Add structured output formats to MCP bridge
- **Action**: Support JSON/YAML output via `format` parameter on tool calls
- **Effort**: 2 hours

### P3-12: Add circuit breaker for DeepSeek API in bridge
- **Action**: Track failure count per provider. Open circuit after 5 consecutive failures. Half-open after 30s timeout
- **Effort**: 4 hours

### P3-13: Add OpenAPI spec for mesh/collab APIs
- **Action**: Document JSON-RPC endpoints, WebSocket protocol, request/response schemas in `docs/mesh-api.md` and `docs/collab-protocol.md`
- **Effort**: 1 day

### P3-14: Move test-seam function vars to struct fields
- **Files**: 11 package-level mutable vars across `movefile.go`, `attachments.go`, `refs.go`, `transcript.go`, `chat_tui.go`, `updater.go`, `balance.go`, `cli.go`, `credentials.go`, `plan.go`
- **Action**: Move to struct fields or constructor parameters for testability
- **Effort**: 1 day

### P3-15: Standardize CI action versions
- **Action**: Align all workflows to latest action versions (`actions/checkout@v6`, `setup-go@v6`, etc.)
- **Effort**: 1 hour

### P3-16: Add `readOnlyRootFilesystem` to Docker
- **Action**: Add `--read-only` flag to Docker runtime or Kubernetes `securityContext`
- **Effort**: 30 minutes

### P3-17: Add config validation on startup
- **Action**: Validate TOML values (types, ranges, unknown keys) on load. Warn on invalid, error on critical
- **Effort**: 4 hours

### P3-18: Defer learn pattern detection to read-time
- **File**: `internal/learn/learn.go:119`
- **Action**: Move `detectPatterns()` from `Observe()` to `Patterns()`/`BuildReflectionPrompt()`. Reduces O(n²) to O(n)
- **Effort**: 1 hour

---

## 7. Architectural Recommendations

### 7.1 Decompose boot.Build (AR-01)

**Current**: Single 1560-line function that imports 30+ packages.

**Proposed**: Extract into `Builder` struct with phase methods.

```go
type Builder struct {
    cfg     *config.Config
    opts    Options
    reg     *tool.Registry
    host    *plugin.Host
    // ...
}

func Build(ctx context.Context, opts Options) (*control.Controller, error) {
    b := &Builder{opts: opts}
    if err := b.loadConfig(); err != nil { return nil, err }
    if err := b.buildProvider(); err != nil { return nil, err }
    if err := b.buildToolSurface(); err != nil { return nil, err }
    if err := b.buildPlugins(ctx); err != nil { return nil, err }
    if err := b.buildSubagents(); err != nil { return nil, err }
    if err := b.buildMesh(); err != nil { return nil, err }
    return b.assemble()
}
```

**Benefit**: Each phase is independently testable. Merge conflicts on `boot.go` become smaller.

### 7.2 Promote SharedHost for Model Switching (AR-03)

**Current**: Model switch → full Controller rebuild → all MCP servers restart.

**Proposed**: `SharedHost` (already used by desktop) becomes the default pattern. Model switch only rebuilds Provider + Agent, preserving plugin host, tool registry, and session state.

**Benefit**: Sub-second model switching. No MCP server reconnections.

### 7.3 Bot Session Pooling (AR-04)

**Current**: Each IM user gets a full `boot.Build() → Controller` with its own provider connection and MCP subprocess tree.

**Proposed**: Pool of shared Provider connections and a single plugin Host. Sessions share these resources with per-session tool registries and permissions.

**Benefit**: O(1) resource usage per platform instead of O(N) per user.

### 7.4 Strengthen Error Taxonomy

**Current**: 4 sentinel errors for 84 packages. Most errors are inline `fmt.Errorf`.

**Proposed**: Domain-specific error types per subsystem:

```go
// internal/provider/
type StreamError struct { Provider, Model string; Cause error }
type RateLimitError struct { RetryAfter time.Duration }

// internal/permission/
type DeniedError struct { Tool, Rule string }
type ApprovalRequiredError struct { Tool string; Subjects []string }

// internal/plugin/
type SpawnError struct { Server string; Cause error }
type TimeoutError struct { Server string; Duration time.Duration }
```

**Benefit**: Callers use `errors.As()` instead of string matching. Tests become robust to message changes.

### 7.5 Separate Recall Read-Path from Write-Path (Memory Server)

**Current**: Every `Recall` query mutates data (increments `AccessCount`, boosts `Importance`, extends `ExpiresAt`, writes entire JSON file).

**Proposed**: `Recall` is purely read-only. Access tracking is batched in-memory and flushed during periodic `Tidy()`.

**Benefit**: Eliminates write amplification. Enables `RLock` on recall path. 10x throughput improvement under concurrent access.

---

## 8. Operational Recommendations

### 8.1 CI Pipeline Hardening

| Item | Current | Target | Effort |
|------|---------|--------|--------|
| Hermes CI test scope | 3 packages | All 83 packages | 30 min |
| govulncheck | informational | blocking | 5 min |
| Coverage gate | none | 70% minimum | 30 min |
| Linters | 5 (errcheck, govet, ineffassign, staticcheck, unused) | +gosec, +gocritic, +revive | 4h |
| Action versions | mixed v4/v5/v6 | standardized v6 | 1h |

### 8.2 Deployment Hardening

| Item | Current | Target | Effort |
|------|---------|--------|--------|
| Docker arch | amd64 only | amd64 + arm64 | 2h |
| Docker healthcheck | none | `HEALTHCHECK CMD` | 15 min |
| Helm security context | none | `runAsNonRoot`, `readOnlyRootFilesystem` | 2h |
| Helm config version | 1 | 2 (matches codebase) | 5 min |
| docker-compose version | `latest` | pinned tag | 5 min |

### 8.3 Monitoring Recommendations

| Signal | Tool | Implementation |
|--------|------|----------------|
| CPU/memory profiling | pprof | `--pprof` flag on all binaries |
| Agent telemetry | agentlog | Already implemented (6 event types) |
| Build optimization | PGO | Collect `default.pgo` from representative workload |
| Goroutine leaks | goleak | Already in test dependencies — extend usage |

---

## 9. Dependency Map

Items with dependencies on other items:

```
P0-10 (CI scope) ←── P2-05 (coverage 80%) ←── P2-13 (coverage gate)
P0-05 (context.Background) ←── P2-17 (panic→error in auth.go)
AR-03 (SharedHost) ←── P2-02 (bot session pooling)
CR-12 (mesh initializedPeers) ←── P2-07 (mesh peer caching)
P2-11 (gosec linter) ←── many security items auto-detected
P2-05 (test coverage) ←── P2-06 (fuzz tests)
```

Independent items (can be done in any order):
- All P0 items (except P0-10 → P2-05 chain)
- P1-01 through P1-20 (all independent)
- P2-03 (SQLite default), P2-04 (pprof), P2-10 (slog), P2-14 (multi-arch), P2-15 (healthcheck)

---

## 10. Metrics & Success Criteria

### 10.1 Security

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Critical security findings | 2 | 0 | Audit review |
| High security findings | 3 | 0 | Audit review |
| `gosec` clean | not enabled | 0 findings | CI lint |
| `govulncheck` clean | informational | blocking, 0 vulns | CI check |

### 10.2 Quality

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Test coverage | 66.5% | 80%+ | `go test -coverprofile` |
| Race detector | clean | clean | `go test -race` |
| Packages tested in Hermes CI | 3 | 83 | CI job |
| `staticcheck` findings | 1 | 0 | CI lint |
| Fuzz test targets | 0 | 5+ | `go test -fuzz` |

### 10.3 Concurrency

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Critical concurrency findings | 3 | 0 | Audit review |
| Goroutine leaks | 3 known | 0 | `goleak` in tests |
| `context.Background()` in request paths | 6 subsystems | 0 | Code review |

### 10.4 Operations

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Docker architectures | 1 (amd64) | 2 (amd64 + arm64) | Docker manifest |
| CI workflows with latest actions | ~60% | 100% | Workflow audit |
| Linters enabled | 5 | 8+ | `.golangci.yml` |
| Helm security controls | 0 | 4 (runAsNonRoot, readOnly, noPrivEsc, resources) | Chart review |

---

## Appendix: Source Audits

This action plan consolidates findings from:

1. **`AUDIT.md`** — June 2026 Go codebase audit (58 findings, grade B+)
2. **`AUDIT-2026-07-15.md`** — July 2026 deep architecture audit (40+ findings)
3. **`docs/reasonix-audit-06202026.md`** — June 2026 MCP bridge focused audit (25 findings)
4. **Architectural analysis** — June 2026 deep codebase exploration (3 parallel agents)
