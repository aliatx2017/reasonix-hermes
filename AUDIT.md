# Reasonix Codebase Audit (Historical — June 2025)

> **Note:** This is a historical audit from June 2025. All 6 bugs documented
> below have been fixed in subsequent releases. See `docs/CHANGELOG-HERMES.md`
> for the most recent audit. The architecture tree below reflects the codebase
> as of June 2025 (~20 packages). The current codebase has 57 packages (see
> `docs/SPEC.md` §2 for the full layout).

**Date:** 2025-06-11  
**Status:** Historical — all bugs resolved  
**Scope:** All Go source under `cmd/`, `internal/`, `pkg/`, `bot/`  
**Files:** 272 non-test `.go` + 293 test `.go` = 565 total  
**Lines:** ~64K production + ~50K test ≈ 114K total  
**Build:** `go vet ./...` clean (at time of audit)  

---

## Executive Summary

Reasonix is a well-structured, production-grade Go project. The architecture follows Go conventions: `cmd/` for entrypoints, `internal/` for encapsulated business logic, `pkg/` for shared public libraries, `bot/` for the standalone bot binary. No circular imports. Test coverage is above average for Go projects (293 test files, 258 under `internal/` alone). All tests pass with `-race`.

**Key findings:**

1. **Two confirmed bugs** — `MemoryStore.mu` race on concurrent access; `sqliteStorage.Save` does full DELETE+INSERT under a transaction without row-level locking, risking data loss under concurrent writes.
2. **One data-corruption risk** — `MemoryStore` file persistence (`memories.json`) has no atomic write; crash mid-write = zero-length file.
3. **One security gap** — `http.DefaultClient` used in `pkg/mcpbridge/main.go:325` and `cmd/reasonix-hooks/main.go:153` with no timeout on transport; also no TLS certificate verification customization.
4. **One injection vector** — `sqliteStorage.Search` builds LIKE patterns from user input; parameterized queries prevent SQL injection, but LIKE wildcards (`%`, `_`) in user queries are not escaped, allowing wildcard injection.
5. **One resource leak** — `sqliteStorage` `db.Close()` is never called in `main()`; the SQLite connection leaks on shutdown.
6. **Architecture smells** — `internal/bot/gateway.go` (592 lines, 16K) and `internal/control/controller.go` (2,722 lines) are god objects. `internal/cli/` has 49 source files with no cohesive boundary.

---

## Architecture Overview

```
cmd/
  reasonix/          → main CLI entrypoint (dispatches to internal/cli)
  reasonix-hooks/   → git hook manager
  reasonix-plugin-example/ → plugin SDK example
  e2ebench/         → end-to-end mutation benchmark

internal/
  agent/       → LLM agent loop, tool dispatch, conversation management
  boot/        → session bootstrap, controller assembly
  bot/         → multi-platform bot gateway (Discord, Feishu, QQ, WeChat)
  cli/         → TUI (Bubble Tea), command routing, 49 source files
  config/      → TOML config loading, MCP server config, model selection
  control/     → controller: orchestrates agent runs, approvals, checkpoints
  checkpoint/  → session state snapshots for resume
  event/       → typed event stream (Sink interface, Event types)
  evidence/    → readiness audit receipts
  fileref/     → file reference resolution
  fileutil/    → file utility functions (encoding sub-package)
  frontmatter/ → YAML frontmatter parser
  hook/        → git hook lifecycle (pre-commit, post-commit, etc.)
  i18n/        → English/Chinese message catalogs
  instruction/ → system prompt composition
  jobs/        → background job runner
  lsp/         → LSP client integration
  memory/      → hierarchical doc-memory + auto-memory (remember/forget)
  mcpdiag/     → MCP server diagnostics
  netclient/   → HTTP client factory
  nilutil/     → nil-safe interface helpers
  notify/      → cross-platform desktop notifications
  outputstyle/ → output style enum (concise, verbose, etc.)
  permission/  → policy engine: allow/ask/deny rules, Gate, glob matching
  plugin/      → MCP plugin transport (stdio, SSE), lifecycle
  proc/        → process management (kill groups, signal handling)
  provider/    → LLM provider abstraction (openai, anthropic sub-packages)
  sandbox/     → OS-level sandbox (macOS Seatbelt)
  serve/       → HTTP/WebSocket server mode
  skill/       → skill registry and loader
  sysproxy/    → system proxy detection
  tool/        → tool registry (builtin/ sub-package with 26 source files)

pkg/
  mcpbridge/     → standalone MCP bridge server (reasonix-as-MCP-tool)
  memoryserver/  → Hindsight memory MCP server (retain/recall/reflect)
  mcputil/       → MCP JSON-RPC server framework
  httputil/      → Bearer auth middleware

bot/
  main.go → standalone Discord/bot entrypoint
```

**Dependency direction** is clean: `cmd/` → `internal/` → `pkg/` (no upward imports). No circular dependencies detected. `internal/bot/` depends on `internal/boot`, `internal/control`, `internal/event` — appropriate. `pkg/` packages are independent and reusable.

---

## Package-by-Package Analysis

### `cmd/reasonix/main.go`
- **Purpose:** CLI entrypoint, thin wrapper over `internal/cli`.
- **API surface:** Single `main()` that calls `cli.Run()`.
- **Cohesive:** Yes. Exits with `os.Exit(cli.Run(...))`.
- **Tests:** None at package level (all testing in `internal/cli`).
- **Issues:** None.

### `cmd/reasonix-hooks/main.go` (5,795 chars)
- **Purpose:** Manages git hooks (install, uninstall, run).
- **API surface:** `main()`, three subcommands.
- **Cohesive:** Yes. Single responsibility.
- **Tests:** Present (`main_test.go`). Tests hook installation lifecycle.
- **Issues:** Uses `http.DefaultClient` at line 153 for callback — no transport timeout.

### `cmd/reasonix-plugin-example/main.go` (13K chars)
- **Purpose:** Reference MCP plugin implementation.
- **Tests:** None (example code).
- **Issues:** None significant (example code).

### `cmd/e2ebench/main.go` (13.7K chars)
- **Purpose:** End-to-end mutation benchmark.
- **Tests:** None (benchmark tool).
- **Issues:** Multiple `os.Exit(1)` calls — acceptable for CLI tool.

### `internal/boot/boot.go` (1,155 lines)
- **Purpose:** Session bootstrap: config loading, controller assembly, model resolution.
- **API surface:** `Build(ctx, Options) (*control.Controller, error)` — single entry point.
- **Cohesive:** Mostly yes, but the file is large. Builds controller, memory, permissions, tools, provider all in one function.
- **Tests:** 5 test files. Good coverage of config resolution.
- **Issues:** `Build()` is a mega-function that assembles 10+ subsystems. Hard to test in isolation.

### `internal/control/controller.go` (2,722 lines)
- **Purpose:** Core agent controller — run loop, tool execution, approval flow, checkpointing.
- **API surface:** `Controller` struct with `Run()`, `Approve()`, `AnswerQuestion()`, `Cancel()`, `EnableInteractiveApproval()`.
- **Cohesive:** **No.** God object. 2,700 lines handling: run loop, streaming, tool dispatch, approval, checkpointing, compaction, sub-agent spawning. Should be decomposed.
- **Tests:** 19 test files including e2e approval tests. Reasonable coverage.
- **Issues:** `promptMu sync.Mutex` and `mu sync.Mutex` — two mutexes with unclear scoping. `Run()` method is ~300 lines.

### `internal/agent/agent.go` (1,323 lines)
- **Purpose:** LLM agent loop — conversation management, tool calling, streaming.
- **API surface:** `Agent` struct with `Run(ctx, input, sink)`.
- **Cohesive:** Mostly yes. Handles LLM interaction pattern.
- **Tests:** 36 test files. Extensive coverage.
- **Issues:** File is large but logically cohesive.

### `internal/config/config.go` (1,817 lines)
- **Purpose:** TOML config loading, model selection, MCP server config.
- **API surface:** `Config` struct, `Load()`, `Default()`, `Save()`.
- **Cohesive:** No. Config struct has grown to 1,800 lines with multiple concerns.
- **Tests:** Good coverage.
- **Issues:** `ccswitch.go` uses `exec.Command` for SQLite queries — functional but unusual.

### `internal/bot/gateway.go` (592 lines, 16K)
- **Purpose:** Multi-platform bot gateway — session management, command routing, message rendering.
- **API surface:** `BotGateway` struct with `Start()`, `HandleMessage()`, `HandleCallback()`, `getOrCreateSession()`.
- **Cohesive:** Reasonable, but god-object tendencies.
- **Tests:** 6 test files.
- **Issues:** `getOrCreateSession()` builds a new controller on every new chat — no session eviction or TTL. Controllers accumulate in `gw.controllers` map indefinitely → memory leak in long-running bots.

### `internal/bot/discord/discord.go` (10.5K chars)
- **Purpose:** Discord adapter implementing `bot.Adapter`.
- **API surface:** `Adapter` interface methods: `Send()`, `Platform()`, `Start()`, `Stop()`.
- **Tests:** None. Discord adapter is untested.
- **Issues:** None critical, but no tests.

### `internal/bot/feishu/`, `internal/bot/qq/`, `internal/bot/weixin/`
- **Purpose:** Platform adapters for Feishu, QQ, WeChat.
- **Tests:** None for any adapter.
- **Issues:** All three lack test coverage entirely.

### `internal/memory/` (7 source, 8 test files)
- **Purpose:** Hierarchical doc-memory (REASONIX.md, AGENTS.md) + auto-memory store (remember/forget).
- **API surface:** `Set`, `Load()`, `Store`, `Compose()`, `WriteDoc()`, `AppendDoc()`.
- **Cohesive:** Yes. Clean separation between doc-memory and auto-memory.
- **Tests:** Good coverage of discovery, save, delete, compose.
- **Issues:** `Store.Save()` and `Store.Delete()` are not safe for concurrent access — no locking on file writes.

### `internal/event/event.go` (262 lines)
- **Purpose:** Typed event stream — `Sink` interface, `Event` struct with 15+ kinds.
- **API surface:** `Sink` interface (`Emit(Event)`), `FuncSink`, `Discard`.
- **Cohesive:** Yes. Clean event model.
- **Tests:** Minimal — 1 test file.
- **Issues:** None architectural.

### `internal/permission/` (2 source, 3 test files)
- **Purpose:** Policy engine — rule evaluation, glob matching, Gate with Approver.
- **API surface:** `Policy`, `Gate`, `New()`, `ParseRule()`, `Decide()`, `Check()`.
- **Cohesive:** Yes. Well-separated pure logic from I/O.
- **Tests:** Good coverage of rule parsing, glob matching, precedence.
- **Issues:** `matchGlob` implements custom glob matching — linear time with backtracking (exponential worst case on adversarial patterns like `a*a*a*a*b`). Acceptable for tool names but worth noting.

### `internal/notify/` (5 source, 1 test file)
- **Purpose:** Cross-platform desktop notifications (macOS, Linux, Windows).
- **API surface:** `Send(Notification)`.
- **Cohesive:** Yes. Platform-specific files via build tags.
- **Tests:** Darwin sender untested (requires OS).
- **Issues:** `sender_darwin.go` uses `osascript` via `exec.Command` — no input sanitization on `Title`/`Body`. Shell injection possible if notification content contains `"`. **Severity: Low** (notifications come from internal state, not external input, but still a defensive programming gap).

### `internal/sandbox/` (4 source, 2 test files)
- **Purpose:** OS-level process sandboxing (macOS Seatbelt).
- **API surface:** `Spec` struct, `Command()`, `Available()`.
- **Cohesive:** Yes.
- **Tests:** Tests verify Seatbelt profile generation on macOS.
- **Issues:** Only macOS is implemented. Linux/Windows get no sandboxing (documented as intentional fallback).

### `internal/checkpoint/` (1 source, 1 test)
- **Purpose:** Session state snapshots for resume.
- **API surface:** Small. Save/load checkpoint.
- **Cohesive:** Yes.
- **Issues:** None.

### `internal/plugin/` (9 source, 12 test)
- **Purpose:** MCP plugin transport (stdio, SSE), lifecycle management.
- **API surface:** `Plugin` struct, `Start()`, `Stop()`, `Transport`.
- **Cohesive:** Yes. Well-structured transport abstraction.
- **Tests:** Good coverage.
- **Issues:** `transport_stdio.go:57` and `:282` use `exec.CommandContext` — properly uses context for cancellation.

### `internal/provider/` (8 source, 13 test)
- **Purpose:** LLM provider abstraction (OpenAI, Anthropic sub-packages).
- **API surface:** `Provider` interface, streaming, retry logic.
- **Cohesive:** Yes.
- **Tests:** Good coverage including streaming tests.

### `internal/tool/` (26 source, 36 test)
- **Purpose:** Built-in tool registry and implementations (bash, file ops, grep, glob, etc.).
- **API surface:** `Registry`, `Tool` interface, 20+ built-in tools.
- **Cohesive:** Yes. Each tool in own file.
- **Tests:** Extensive coverage per tool.

### `internal/cli/` (49 source, 36 test)
- **Purpose:** Bubble Tea TUI, command routing, `/` command handling.
- **API surface:** `Run()`, TUI model, command dispatcher.
- **Cohesive:** **No.** 49 source files is too large for a single package. Should be decomposed into sub-packages (tui, commands, mcp_manager, etc.).
- **Tests:** 36 test files — reasonable but can't cover 49 source files adequately.
- **Issues:** `mcp_manager_actions.go:399-415` uses `exec.Command("sh", "-lc", editor+" "+shellQuote(path))` — shell injection risk if `editor` contains shell metacharacters. `shellQuote` may not cover all cases.

### `internal/proc/` (4 source, 4 test)
- **Purpose:** Process management — kill groups, signal handling.
- **Cohesive:** Yes.
- **Issues:** None.

### `pkg/mcpbridge/main.go` (18.3K chars)
- **Purpose:** Standalone MCP bridge — exposes Reasonix as an MCP tool server.
- **API surface:** `Bridge` struct, MCP tool handlers, doctor check, skill listing, orchestration.
- **Cohesive:** Reasonable but large for a single file.
- **Tests:** 937-line test file. Good coverage.
- **Issues:**
  1. **`http.DefaultClient`** at line 325 — no timeout on transport. Should use custom `http.Client` with timeout.
  2. API key passed via `Authorization: Bearer` header — acceptable but no key rotation support.
  3. No response body size limit on `io.ReadAll` at line 331 — memory exhaustion from malicious API response.

### `pkg/memoryserver/main.go` (17.8K chars) + `sqlite_storage.go` (6.8K chars)
- **Purpose:** Hindsight memory MCP server (retain/recall/reflect with SQLite backend).
- **API surface:** `MemoryStore`, `MemoryEntry`, `memoryHandler`, MCP tool handlers.
- **Cohesive:** Yes.
- **Tests:** 1,568-line test file. Extensive.
- **Issues:**
  1. **Race condition on `MemoryStore.mu`**: `MemoryStore` has `sync.RWMutex` at line 70 but `Tidy()` iterates entries while other methods may mutate. The `Search()` method in `sqliteStorage` is safe (delegated to SQL), but the JSON-backed store has no transaction isolation between Load/Save.
  2. **`sqliteStorage.Save()` DELETE+INSERT**: Full table wipe + re-insert under transaction. Under concurrent access, rows inserted between Load and Save are lost. Should use UPSERT.
  3. **`sqliteStorage.db.Close()` never called** in `main()` — connection leak on shutdown.
  4. **LIKE wildcard injection** in `sqliteStorage.Search()`: `%` and `_` in user queries are not escaped before building LIKE patterns. `LIKE '%" + query + "%'` lets users inject wildcards.
  5. **No atomic file write** for `memories.json`: `os.WriteFile` at `main.go:272` is not atomic (no write-to-temp-then-rename). Crash during write = zero-length file = data loss.
  6. **`io.ReadAll`** on HTTP response body (test code line 838, 850) with no size limit.

### `pkg/mcputil/server.go` (7.6K chars)
- **Purpose:** MCP JSON-RPC server framework (stdio + HTTP transports).
- **API surface:** `Server` struct, `HandleMessage()`, `ServeStdio()`, `ServeHTTP()`.
- **Cohesive:** Yes.
- **Tests:** Via consumers (mcpbridge, memoryserver).
- **Issues:** None significant.

### `pkg/httputil/auth.go` (2.8K chars)
- **Purpose:** Bearer token auth middleware.
- **Cohesive:** Yes.
- **Issues:** Constant-time comparison not used for token check. Uses `subtle.ConstantTimeCompare` — actually checked: it does NOT use it. Uses `strings.TrimPrefix` + `==`. **Timing side-channel** on API key validation.

---

## Bugs Found

### BUG-1: MemoryStore file write is not atomic
- **File:** `pkg/memoryserver/main.go:272`
- **Severity:** Medium
- **Description:** `os.WriteFile` for `memories.json` is not atomic. A crash or power loss between truncate and write produces a zero-length file, losing all memories. Should write to a temp file then rename.
- **Fix:** Use `os.WriteFile` to a `.tmp` file, then `os.Rename` to the final path.

### BUG-2: sqliteStorage.Save() loses concurrent writes
- **File:** `pkg/memoryserver/sqlite_storage.go:91-128`
- **Severity:** Medium
- **Description:** `Save()` does `DELETE FROM memories` then re-inserts all entries. If another goroutine inserts between the DELETE and INSERT, that data is lost. Under the current architecture (single MCP server), this is unlikely but architecturally wrong.
- **Fix:** Use UPSERT (INSERT OR REPLACE) or row-level updates instead of full table wipe.

### BUG-3: LIKE wildcard injection in sqliteStorage.Search()
- **File:** `pkg/memoryserver/sqlite_storage.go:140-147`
- **Severity:** Low
- **Description:** User-provided query strings are embedded in LIKE patterns without escaping `%` and `_`. A query like `100%` matches unintended patterns.
- **Fix:** Escape LIKE wildcards before building the pattern, or use FTS5 full-text search instead of LIKE.

### BUG-4: http.DefaultClient with no timeout
- **File:** `pkg/mcpbridge/main.go:325`, `cmd/reasonix-hooks/main.go:153`
- **Severity:** Medium
- **Description:** `http.DefaultClient.Do(req)` has no transport timeout. While the request context has a 60s deadline in mcpbridge, the transport itself has no idle connection timeout, connection pool limits, or TLS handshake timeout. A hung server could leak connections.
- **Fix:** Create a custom `http.Client` with `Transport` configured for timeouts.

### BUG-5: Timing side-channel in Bearer token validation
- **File:** `pkg/httputil/auth.go`
- **Severity:** Low
- **Description:** Token comparison uses `==` (or `strings.TrimPrefix` + `==`) instead of `crypto/subtle.ConstantTimeCompare`. Allows timing-based key extraction.
- **Fix:** Replace with `subtle.ConstantTimeCompare`.

### BUG-6: BotGateway session memory leak
- **File:** `internal/bot/gateway.go:516-527`
- **Severity:** Medium
- **Description:** `getOrCreateSession()` adds entries to `gw.controllers` map but no code ever removes them. Long-running bots accumulate stale sessions indefinitely.
- **Fix:** Add TTL-based eviction or explicit `/quit` command cleanup.

### BUG-7: sqliteStorage.db.Close() never called
- **File:** `pkg/memoryserver/main.go:545-601`
- **Severity:** Low
- **Description:** The `*sql.DB` opened in `newSQLiteStorage()` is never closed. The `sqliteStorage` has a `Close()` method but `main()` never calls it (not even via defer).
- **Fix:** `defer ss.Close()` after creation in `main()`.

---

## Refactor Opportunities

### REF-1: Decompose `internal/control/controller.go` (2,722 lines)
- **Problem:** God object managing run loop, streaming, tool dispatch, approval flow, checkpointing, compaction, and sub-agent spawning in a single file.
- **Recommendation:** Split into: `controller.go` (orchestrator), `approval.go`, `checkpoint.go`, `compaction.go`, `subagent.go`. The `Run()` method alone is ~300 lines.

### REF-2: Decompose `internal/cli/` (49 source files)
- **Problem:** Single package with 49 source files covering TUI model, command routing, MCP management, git status, and more. No sub-packages.
- **Recommendation:** Split into `cli/tui/`, `cli/command/`, `cli/mcp/`, `cli/render/`.

### REF-3: Extract `internal/bot/render.go` rendering logic
- **Problem:** `renderSink` in `internal/bot/render.go` hard-codes Chinese strings for all platforms.
- **Recommendation:** Move display strings to `internal/i18n/` or a per-platform renderer interface.

### REF-4: Unify MemoryStore persistence backends
- **Problem:** `pkg/memoryserver/main.go` has a `MemoryStore` (JSON file) and `sqliteStorage` (SQLite), with different APIs (`Load/Save` vs `Search`). The `Storage` interface bridges them but `MemoryStore` duplicates search logic that `sqliteStorage.Search` already implements.
- **Recommendation:** Make `Storage` interface the primary API; `MemoryStore` should delegate to it entirely. Remove JSON-file search path when SQLite backend is used.

### REF-5: `pkg/mcpbridge/main.go` is 548 lines
- **Problem:** Single file handles doctor, skills, planning, orchestration, and tool dispatch.
- **Recommendation:** Extract `doctor.go`, `skills.go`, `orchestrator.go` in the same package.

---

## Enhancement Recommendations

### ENH-1: Add session TTL/eviction to BotGateway
- The `controllers` map grows without bound. Add a background goroutine that evicts sessions idle for >N minutes, or add a `/quit` command that removes the session.

### ENH-2: Use FTS5 for memory search
- Replace `LIKE '%query%'` in `sqliteStorage.Search()` with SQLite FTS5 full-text search. LIKE scans cannot use indexes; FTS5 provides proper relevance ranking and eliminates wildcard injection.

### ENH-3: Add test coverage for bot platform adapters
- `discord/`, `feishu/`, `qq/`, `weixin/` have zero test files. At minimum, add unit tests for the `Adapter` interface methods and message format conversion.

### ENH-4: Implement sandbox for Linux
- `internal/sandbox/` only implements macOS Seatbelt. Linux support (via `namespaces`/`seccomp`/`bubblewrap`) would close a significant security gap for the most common deployment platform.

### ENH-5: Add graceful shutdown to bot and memory servers
- Neither `bot/main.go` nor `pkg/memoryserver/main.go` handle SIGTERM/SIGINT for graceful shutdown. The bot gateway has no `Shutdown()` method.

### ENH-6: Rate-limit MCP bridge API calls
- `pkg/mcpbridge/main.go` calls the DeepSeek API with no rate limiting. A runaway agent could exhaust API quota. Add per-session rate limiting.

### ENH-7: Add context cancellation propagation in bot gateway
- `internal/bot/gateway.go` creates controllers with `context.Background()` and doesn't propagate cancellation from the parent context. Long-running sessions may leak goroutines.

---

## Security Review

### SEC-1: Shell injection in desktop notifications (Low)
- **File:** `internal/notify/sender_darwin.go:18`
- **Detail:** `exec.Command("osascript", "-e", script)` where `script` interpolates `m.Title` and `m.Body`. If either contains `"`, the osascript breaks. If they contain `$(...)`, command substitution may execute.
- **Mitigation:** Escape or strip shell metacharacters before interpolation.

### SEC-2: Shell injection in editor launch (Medium)
- **File:** `internal/cli/mcp_manager_actions.go:399-415`
- **Detail:** `exec.Command("sh", "-lc", editor+" "+shellQuote(path))` — `editor` comes from config/env and may contain shell metacharacters. `shellQuote` is a local function that may not handle all edge cases.
- **Mitigation:** Use `exec.Command(editor, path)` (no shell) instead of `sh -lc`.

### SEC-3: Timing side-channel in API key validation (Low)
- **File:** `pkg/httputil/auth.go`
- **Detail:** Bearer token comparison uses string equality instead of constant-time comparison.
- **Mitigation:** Use `crypto/subtle.ConstantTimeCompare`.

### SEC-4: No TLS certificate pinning for LLM API calls (Info)
- **File:** `pkg/mcpbridge/main.go:325`, `internal/provider/`
- **Detail:** HTTP clients use default TLS verification. No certificate pinning. Standard for most deployments but worth noting for high-security environments.

### SEC-5: Sandbox only on macOS (Medium)
- **File:** `internal/sandbox/`
- **Detail:** Only macOS Seatbelt is implemented. On Linux and Windows, `Available()` returns false and commands run unsandboxed. The `bash` tool can execute arbitrary commands with full system access.
- **Mitigation:** Implement Linux sandboxing (namespace/seccomp) as ENH-4.

### SEC-6: Unbounded HTTP response body reads (Low)
- **File:** `pkg/mcpbridge/main.go:331`, `pkg/memoryserver/main_test.go:838,850`
- **Detail:** `io.ReadAll(resp.Body)` with no size limit. A malicious API server could send an enormous response.
- **Mitigation:** Use `io.LimitReader(resp.Body, maxResponseSize)`.

---

## Test Coverage Assessment

| Package | Source Files | Test Files | Coverage Assessment |
|---------|-------------|-----------|---------------------|
| `internal/agent` | 14 | 36 | Excellent |
| `internal/boot` | 1 | 5 | Good |
| `internal/bot` | 11 | 6 | Moderate (adapters untested) |
| `internal/bot/discord` | 1 | 0 | **None** |
| `internal/bot/feishu` | 1 | 0 | **None** |
| `internal/bot/qq` | 2 | 0 | **None** |
| `internal/bot/weixin` | 1 | 0 | **None** |
| `internal/cli` | 49 | 36 | Moderate (spread thin) |
| `internal/config` | ~8 | 5+ | Good |
| `internal/control` | 11 | 19 | Good |
| `internal/event` | 2 | 1 | Minimal |
| `internal/evidence` | 2 | 2 | Good |
| `internal/fileref` | 1 | 0 | **None** |
| `internal/frontmatter` | 1 | 2 | Good |
| `internal/hook` | 3 | 2 | Moderate |
| `internal/instruction` | 1 | 1 | Good |
| `internal/jobs` | 1 | 5 | Excellent |
| `internal/lsp` | 7 | 4 | Good |
| `internal/memory` | 7 | 8 | Good |
| `internal/notify` | 5 | 1 | Minimal |
| `internal/permission` | 2 | 3 | Good |
| `internal/plugin` | 9 | 12 | Excellent |
| `internal/proc` | 4 | 4 | Excellent |
| `internal/provider` | 8 | 13 | Excellent |
| `internal/sandbox` | 4 | 2 | Moderate |
| `internal/skill` | 4 | 3 | Good |
| `internal/tool` | 26 | 36 | Excellent |
| `pkg/mcpbridge` | 1 | 1 (937 lines) | Good |
| `pkg/memoryserver` | 2 | 1 (1,568 lines) | Good |
| `pkg/mcputil` | 1 | 0 | **None** |
| `pkg/httputil` | 1 | 0 | **None** |

**Overall:** 293 test files / 272 source files = 1.08 test-to-source ratio. Above average for Go projects. Key gaps:
- **Zero test coverage:** Discord, Feishu, QQ, WeChat adapters; `pkg/mcputil`; `pkg/httputil`; `internal/fileref`.
- **Spread thin:** `internal/cli` (49 source files, 36 test files — many source files untested).
- **Missing concurrency tests:** `MemoryStore` race conditions not tested despite `sync.RWMutex` presence.

---

*End of audit.*