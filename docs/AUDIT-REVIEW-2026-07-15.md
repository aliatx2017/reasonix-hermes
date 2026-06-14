# Audit of AI-Generated Deep Analysis — Reasonix Hermes

**Date:** 2026-07-15
**Methodology:** Each claim was verified against the actual code at `HEAD` (f37bc16) using grep, file reads, and line-by-line inspection. Claims about paths that no longer exist (`pkg/mcpbridge/`, `pkg/memoryserver/`) indicate the analysis ran against an outdated snapshot of the repo.

## Summary

| Verdict | Count | Notes |
|---------|-------|-------|
| **Already fixed** | 8 | Were real bugs in an older version; fixes are in the code now |
| **False/Misleading** | 6 | Paths don't exist, code was misread, or the claim is incorrect |
| **Real (partial/low)** | 3 | Real but low severity or partially addressed |
| **Real (needs fix)** | 1 | Confirmed bug present in current code |

---

## Claims Already Fixed (8)

These were real bugs in older versions of the codebase. The fixes are present in the current code — the analysis ran against stale references.

### BUG-2: SQLite DELETE→INSERT race
**Claim:** `sqlite_storage.go` deletes all rows then re-inserts, losing concurrent writes.
**Verdict:** Already fixed. Code at `cmd/reasonix-memoryserver/sqlite_storage.go:106` uses `INSERT OR REPLACE`.
Fix applied: June 12, 2026 (REASONIX.md: "P1-1: DELETE+INSERT → INSERT OR REPLACE upsert").

### BUG-3: LIKE wildcard injection
**Claim:** User can inject `%` or `_` into LIKE queries.
**Verdict:** Already fixed. `sqlite_storage.go:148-154` uses `LIKE ? ESCAPE '\'` with `escapeLikeWildcards()` escaping `%`, `_`, and `\`.
Fix applied: June 12, 2026 (REASONIX.md: "P2-2: LIKE wildcard escaping with ESCAPE '\' clause").

### BUG-4: HTTP client without timeout
**Claim:** `http.DefaultClient.Do(req)` without timeout.
**Verdict:** Already fixed. All paths use `netclient.DefaultClient()` which has a 30s timeout. The referenced `pkg/mcpbridge/` directory no longer exists (moved to `cmd/reasonix-mcpbridge/`).
Fix applied: June 12, 2026 (REASONIX.md: "P0-2: http.DefaultClient → netclient.DefaultClient() across 7 files").

### BUG-5: Bearer token timing side-channel
**Claim:** Token comparison leaks timing information.
**Verdict:** Already fixed. `pkg/httputil/auth.go:62` uses `subtle.ConstantTimeCompare([]byte(token), []byte(a.APIKey))`.

### BUG-7: SQLite connection never closed
**Claim:** ss.Close() never called on sqliteStorage.
**Verdict:** Already fixed. `cmd/reasonix-memoryserver/main.go:692` has `defer ss.Close()`.
Fix applied: June 12, 2026 (REASONIX.md: "P1-1: sqliteStorage.Close() deferred in memoryserver main").

### Unbounded HTTP response reads
**Claim:** `io.ReadAll(resp.Body)` without limit on response bodies.
**Verdict:** Already fixed. 22 instances of `io.LimitReader` exist across the codebase (web_fetch, balance checks, MCP transport, mesh, install_source, provider fetch_models, bot adapters).
Fix applied: June 12, 2026 (REASONIX.md: "P2-3: io.LimitReader on 5 unbounded HTTP response reads").

### Graceful shutdown missing
**Claim:** No SIGTERM/SIGINT handlers in bot/main.go and memoryserver.
**Verdict:** Already fixed. `cmd/reasonix-memoryserver/main.go` has `signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)` and graceful shutdown. REASONIX.md notes: "Graceful shutdown: memoryserver handles SIGINT/SIGTERM in HTTP mode. Bot already had it."

### Linux sandbox missing
**Claim:** Only macOS Seatbelt; Linux/Windows run unsandboxed.
**Verdict:** Already implemented. `internal/sandbox/seatbelt_other.go` provides bubblewrap (bwrap) integration for Linux. REASONIX.md documents: "Linux sandbox: bubblewrap (bwrap) integration — matches macOS Seatbelt profile (read-only root, writable workspace + toolchain caches, network isolation). Graceful fallback when bwrap missing."

---

## False or Misleading Claims (6)

These claims are erroneous — either referencing nonexistent files, misreading existing code, or making incorrect assertions.

### BUG-1: MemoryStore file write race
**Claim:** `os.WriteFile("memories.json", ...)` is non-atomic; could lose all data.
**Verdict:** False. `cmd/reasonix-memoryserver/main.go:163-167` writes to `.tmp` then calls `os.Rename(tmp, "memories.json")` — the standard atomic write pattern on POSIX. The claim misread the code or was generated from an older version that didn't use tmp+rename.

### Discord adapter: zero test coverage
**Claim:** `internal/bot/discord/` has "1 file, 0 tests."
**Verdict:** False. `internal/bot/discord/discord_test.go` exists and contains **57 tests** covering adapter creation, message handling, mention stripping, keyboard rendering, interaction parsing, and channel/DM filtering.

### Notification shell injection
**Claim:** `osascript` command constructed with raw string interpolation; `$(...) ` RCE possible.
**Verdict:** False. `internal/notify/sender_darwin.go:26-28` defines `appleScriptString()` which escapes `\` and `"`. The notification body and title are properly sanitized before interpolation.

### http.DefaultClient
**Claim:** `cmd/reasonix-hooks/main.go:153` uses `http.DefaultClient`.
**Verdict:** False. `cmd/reasonix-hooks/main.go:155` uses `netclient.DefaultClient().Do(req)`.

### SQLite Save() DELETE→INSERT
**Claim:** Deletes all rows, then re-inserts. (Duplicate of BUG-2.)
**Verdict:** False (same as BUG-2 — already INSERT OR REPLACE).

### Shell injection in editor — full vulnerability
**Claim:** `exec.Command("sh", "-lc", editor + " " + path)` allows full RCE.
**Verdict:** Partial. The `path` argument IS quoted via `shellQuote()` (wraps in single quotes). The editor value from `$VISUAL`/`$EDITOR` is NOT quoted, so if set to a malicious value it could execute commands. However, this requires the user's own environment variables to be compromised — low real-world risk.

---

## Real Issues (4)

### 1. BotGateway session memory leak — REAL (High)
**Code:** `internal/bot/gateway.go:888-929` (`getOrCreateSession`)
**Evidence:** The `gw.controllers` map grows without bound. There is no background goroutine, no TTL-based eviction, no cleanup logic. Each session state holds a full Controller (~1-5MB). A long-running bot with many users will OOM.
**Note:** REASONIX.md claims "P0-1: BotGateway session eviction (idle timeout + background goroutine)" was applied on June 12, 2026, but the fix is NOT present in the actual code. It may have been lost in an upstream merge or never actually committed.
**Fix:** Add a background eviction goroutine that runs every 5 minutes, removes sessions idle for >30 minutes, and calls `ctrl.Close()` + `gw.mu.Lock()` before deleting from the map.

### 2. Shell injection in editor launch — PARTIALLY REAL (Low)
**Code:** `internal/cli/mcp_manager_actions.go:399,405`
**Evidence:** `exec.Command("sh", "-lc", editor+" "+shellQuote(path))` — the path is quoted but the `editor` variable (from `$VISUAL`/`$EDITOR`) is not. An attacker with control over the user's environment could inject commands.
**Risk:** Very low — requires compromising the user's own environment variables.
**Fix:** Use `exec.Command(editor, path)` without shell, or quote both editor and path with `shellQuote`.

### 3. Controller size — 3,682 lines (Medium)
**Code:** `internal/control/controller.go`
**Evidence:** 3,682 lines mixing approval, checkpointing, compaction, sub-agents, tool dispatch, and session management. The analysis claimed 2,722 lines — growth since then.
**Risk:** Maintenance burden, merge conflict surface, cognitive load.
**Recommended:** Not urgent. Decompose incrementally when touching related code.

### 4. No SQLite FTS5 — Low priority
**Code:** `cmd/reasonix-memoryserver/sqlite_storage.go`
**Evidence:** Search uses `LIKE` with `ESCAPE` — O(n) scans. FTS5 would give O(log n) and better relevance scoring.
**Risk:** Only matters at 100K+ entries. Not a current problem.
**Action:** Defer until memoryserver handles production-scale datasets.

---

## Recommended Priority — Concrete Steps

### Session 1: Fix BUG-6 (session memory leak)
1. Add `idleTimeout time.Duration` field to `BotGateway` (default 30 minutes)
2. Start a background goroutine in `BotGateway.Start()` that evicts idle sessions:
   ```go
   go func() {
       ticker := time.NewTicker(5 * time.Minute)
       defer ticker.Stop()
       for {
           select {
           case <-gw.ctx.Done():
               return
           case <-ticker.C:
               gw.evictIdleSessions(gw.idleTimeout)
           }
       }
   }()
   ```
3. Add `evictIdleSessions(maxAge time.Duration)` that iterates gw.controllers, deletes entries where `time.Since(state.lastActive) > maxAge`, calling `state.ctrl.Close()` before deletion. Hold `gw.mu` during the iteration.
4. Add tests: `TestBotGateway_SessionEviction`, `TestBotGateway_SessionReuseResetsTimeout`
5. Update REASONIX.md to note the fix is actually applied this time.

### Session 2: Editor shell injection hardening
1. Replace `exec.Command("sh", "-lc", editor+" "+shellQuote(path))` with `exec.Command(editor, path)` — the editor binary is resolved from `$PATH` by `exec.LookPath` in the calling code.
2. If shell must be used (for editors like `vim -c`), quote both `editor` and `path` with `shellQuote()`.
3. Verify with test: `TestMCPEditorLaunch_NoShellInjection`.

### Future (non-urgent)
- Decompose controller into sub-packages when making unrelated changes
- Consider FTS5 migration when memoryserver exceeds 50K entries
- Add missing tests for `internal/bot/slack/` adapter (untested new code)
- Verify glob matching doesn't have exponential worst-case in `internal/permission/policy.go`
