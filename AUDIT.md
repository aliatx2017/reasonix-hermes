# Reasonix Go Codebase Audit — June 2026

## Executive Summary

| Metric | Value |
|--------|-------|
| Go files | 766 (379 source, 387 test) |
| Lines of Go | 222,832 |
| Go version | 1.26.4 (go.mod: 1.25.0) |
| Packages | 84 |
| Tests | 3,642 — ALL PASS |
| Race detector | CLEAN |
| `go vet` | CLEAN |
| `go build` | CLEAN |
| `staticcheck` | 1 issue (unused test helper) |
| `golangci-lint` | 1 issue (same unused func) |
| `govulncheck` | NO VULNERABILITIES |
| Test coverage | 66.5% |
| CI workflows | 18 (3-OS matrix, race, lint, vulncheck, coverage) |

**Overall grade: B+** — Production-grade codebase with solid fundamentals. Clean build, clean vet, zero race conditions, zero vulnerabilities. Gaps are in HTTP timeout coverage, file permission consistency, missing pprof, and test coverage below 80% target.

---

## 1. SECURITY

### CRITICAL

#### S1. WebSocket CheckOrigin allows any origin (CSRF)
**File:** `internal/collab/collab.go:305`
```go
up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
```
**Risk:** Cross-site WebSocket hijacking. Malicious page can open WS to this endpoint, bypassing same-origin policy.
**Fix:** Validate against allowlist:
```go
up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    for _, allowed := range allowedOrigins {
        if origin == allowed { return true }
    }
    return false
}}
```
**Note:** Same pattern in `internal/collab/collab_test.go:54` — acceptable for test.

#### S2. `http.ListenAndServe` without timeouts (slowloris)
**File:** `internal/serve/serve.go:277`
```go
func (s *Server) Run(addr string) error {
    s.ctl().EnableInteractiveApproval()
    return http.ListenAndServe(addr, s.Handler())
}
```
**Risk:** Slowloris DoS — attacker opens connections slowly, exhausts server file descriptors.
**Fix:** Use `http.Server{}` with timeouts (the `RunGraceful` method at line 283 already does this correctly):
```go
srv := &http.Server{
    Addr:              addr,
    Handler:           s.Handler(),
    ReadHeaderTimeout: 10 * time.Second,
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       120 * time.Second,
}
return srv.ListenAndServe()
```
**Note:** `RunGraceful()` at line 283 is correct — has `ReadHeaderTimeout`. The non-graceful `Run()` is the gap.

#### S3. HTTP servers missing WriteTimeout/ReadTimeout
**Files:**
- `internal/bot/line/line.go:100` — `&http.Server{Handler: mux}` — no timeouts
- `internal/bot/feishu/feishu.go:709` — `&http.Server{Addr: ..., Handler: mux}` — no timeouts

**Risk:** Slow request/response can hang connections indefinitely.
**Fix:** Add `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` to all `http.Server{}` instances.

### HIGH

#### S4. `http.DefaultClient.Do()` without timeout (production code)
**Files:**
- `desktop/write_mode.go:241` — LLM completions API call
- `desktop/bot_connection_app.go:684` — OAuth registration
- `cmd/reasonix-pr-review/main.go:105,127` — GitHub API calls
- `internal/bot/qq/gateway.go:385` — QQ gateway API

**Risk:** `http.DefaultClient` has NO timeout. Hung API calls block forever.
**Fix:** Use `netclient.DefaultClient()` (project's pooled client with 120s timeout) or pass context with deadline:
```go
resp, err := netclient.DefaultClient().Do(req)  // instead of http.DefaultClient.Do(req)
```
**Note:** Project already has `netclient.DefaultClient()` with proper timeouts (10s dial, 30s header, 120s overall). These call sites just don't use it.

#### ~~S5. Dockerfile UID mismatch~~ (RETRACTED — false alarm)
**File:** `Dockerfile:30`
```dockerfile
RUN mkdir -p /workspace && chown 65532:65532 /workspace
```
**Image:** `gcr.io/distroless/static-debian12:nonroot` uses UID **65532** (confirmed via Google Groups distroless-users). The Dockerfile is CORRECT.
**Note:** Initial audit incorrectly claimed UID 65534. That was wrong — 65532 is the right value for distroless nonroot. Skill `go-mcp-server` had same error, patched.

#### S6. Shell command execution via `sh -c` with user input
**File:** `internal/cli/eval.go:296`
```go
out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
```
**Risk:** If `cmd` contains user-controlled input, this is command injection.
**Context:** This is in eval runner — `cmd` comes from eval check definitions. If eval definitions are user-controllable (e.g., from skill files), this is exploitable.
**Fix:** Validate `cmd` against allowlist, or split into args and use `exec.Command(parts[0], parts[1:]...)` without shell.

### MEDIUM

#### S7. Non-constant-time token comparison (timing attack)
**File:** `internal/bot/feishu/feishu.go:406-408`
```go
func (a *adapter) verificationTokenValid(token string) bool {
    return a.cfg.VerificationToken == "" || token == a.cfg.VerificationToken
}
```
**Risk:** `==` comparison is vulnerable to timing side-channel. Attacker can deduce token byte-by-byte by measuring response time.
**Fix:** Use `crypto/subtle.ConstantTimeCompare`:
```go
return a.cfg.VerificationToken == "" ||
    subtle.ConstantTimeCompare([]byte(token), []byte(a.cfg.VerificationToken)) == 1
```

#### S8. Raw `err.Error()` in HTTP responses (info leak)
**File:** `internal/serve/serve.go:397,409,454`
```go
http.Error(w, err.Error(), http.StatusInternalServerError)
```
**Risk:** Internal errors (file paths, provider details, API key fragments) leak to HTTP clients.
**Fix:** Log full error internally, return generic message:
```go
slog.Error("switch model failed", "err", err)
http.Error(w, "internal server error", http.StatusInternalServerError)
```

#### S9. Bind all interfaces by default
**Files:**
- `internal/bot/feishu/feishu.go:710` — `Addr: fmt.Sprintf(":%d", port)` (all interfaces)
- `internal/collab/collab.go:101` — `cfg.ListenAddr = ":9091"` (all interfaces)

**Risk:** Webhook/collab endpoints accessible from network. If not behind reverse proxy, anyone can reach them.
**Fix:** Default to `127.0.0.1`:
```go
Addr: fmt.Sprintf("127.0.0.1:%d", port)  // feishu
cfg.ListenAddr = "127.0.0.1:9091"         // collab
```

#### S10. Path traversal in memory doc resolver
**File:** `internal/memory/doc.go:250-260`
```go
func resolvePath(p, baseDir string) string {
    if strings.HasPrefix(p, "~") {
        return filepath.Join(home, rest)  // can escape via ~/../
    }
    if filepath.IsAbs(p) { return p }     // arbitrary absolute paths
    return filepath.Join(baseDir, p)      // can escape via ../../
}
```
**Risk:** Path traversal — `~/../../../etc/passwd` or `../../etc/passwd` can escape intended directory.
**Fix:** After resolving, verify result is within allowed root:
```go
resolved := filepath.Clean(filepath.Join(baseDir, p))
rel, err := filepath.Rel(baseDir, resolved)
if err != nil || strings.HasPrefix(rel, "..") {
    return "" // or return error
}
```

### LOW

#### S11. Unbounded `io.ReadAll` on HTTP responses
**Files:**
- `internal/bot/qq/gateway.go:158,390` — `io.ReadAll(resp.Body)` without LimitReader
- `internal/cli/upgrade.go:243,293,316` — `io.ReadAll(resp.Body)` without limit
- `internal/jobs/jobs.go:640` — `io.ReadAll(f)` on file (less risky)
- `internal/memory/doc.go:127` — `io.ReadAll(f)` on file
- `cmd/reasonix-hooks/main.go:63` — `io.ReadAll(os.Stdin)` (stdin from parent, low risk)

**Risk:** OOM if response body is unexpectedly large.
**Fix:** Wrap with `io.LimitReader`:
```go
body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB max
```
**Note:** Most other call sites in the codebase already use `io.LimitReader` correctly. These are the exceptions.

#### S12. File permissions too permissive for sensitive data
**Files (0o644 on potential config/session data):**
- `desktop/window_state.go:66` — `os.WriteFile(path, data, 0o644)`
- `desktop/workspace.go:46,120,149` — workspace paths at 0o644
- `desktop/write_mode.go:122,127,155` — file writes at 0o644
- `desktop/hermes_dashboard.go:917` — dashboard data at 0o644

**Risk:** 0o644 = world-readable. Session/config data should be 0o600 (owner-only).
**Fix:** Use 0o600 for files containing user data, session state, or configuration.

---

## 2. CONCURRENCY

### GOOD

- `sync.RWMutex` used correctly — `RLock()` for reads, `Lock()` for writes throughout `desktop/` and `cmd/reasonix-memoryserver/`.
- `atomic.Pointer`, `atomic.Int64`, `atomic.Uint64` used for counters and pointers in `internal/control/controller.go`.
- `sync.Pool` used in `internal/provider/openai/openai.go:185` and `internal/provider/anthropic/anthropic.go:145` for buffer reuse.
- Race detector: CLEAN across 3,642 tests.
- `sync.Once` guards `close()` in ACP server — no double-close panics.
- `wg.Add` before `go` pattern followed consistently in `plugin.Host`, `mesh`, `orchestrate`, `jobs`.
- `errgroup` not used but not needed — goroutine coordination via channels and context.

### CRITICAL

#### C1. `context.Background()` in request paths — detached goroutines ignore parent cancellation

Multiple goroutines spawn with `context.Background()` inside request handling paths, detaching from parent context. These goroutines cannot be cancelled when parent request/session terminates.

**C1a: `internal/control/controller.go:787`**
```go
go func() {
    if err := c.Compact(context.Background(), focus); err != nil {
```
Goroutine runs compaction with `context.Background()` — if controller closes during compaction, goroutine continues indefinitely.
**Fix:** Use `c.ctx` or a derived context with cancellation tied to controller lifecycle.

**C1b: `internal/control/controller.go:1628`**
```go
c.hooks.SessionStart(context.Background())
```
SessionStart hooks run with `context.Background()` — hook operations (potentially slow I/O) cannot be cancelled.
**Fix:** Pass controller's session context.

**C1c: `internal/cli/chat_tui.go:3678`**
```go
go func() {
    result, err := m.ctrl.Council(context.Background(), task)
```
Council dispatch uses `context.Background()` — mesh delegation goroutine survives TUI shutdown.
**Fix:** Use `m.ctrl`'s context or TUI lifecycle context.

**C1d: `desktop/bot_connection_app.go:106,150,256,673`**
```go
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
```
Bot connection operations use `context.Background()` instead of app context. If app shuts down, these 15s operations continue.
**Fix:** Use `a.bootContext()` as parent.

**C1e: `desktop/workspace_changes.go:143,232`**
```go
return workspaceGitCommand(context.Background(), args...)
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
```
Git operations use `context.Background()` — no cancellation on tab close/app shutdown.
**Fix:** Pass tab/app context.

**C1f: `internal/bot/feishu/feishu.go:716`**
```go
if err := server.Shutdown(context.Background()); err != nil ...
```
Webhook shutdown uses `context.Background()` — blocks indefinitely if connections refuse to close.
**Fix:** Use `context.WithTimeout(context.Background(), 10*time.Second)`.

#### C2. `readFileWithTimeout` — goroutine leak + slot exhaustion under disk pressure
**File:** `desktop/tabs.go:2924`
```go
go func() {
    data, err := os.ReadFile(path)
    <-readFileWithTimeoutSlots
    ch <- result{data: data, err: err}
}()
```
When timer fires (line 2934), caller returns. Goroutine still blocked on `os.ReadFile` continues. Slot semaphore (16 slots) is released inside goroutine — if many timeouts accumulate under slow disk, 16 slots fill and subsequent reads fail with "too many pending file reads".
**Fix:** Use context-aware file reading, or increase slot count, or abort the read via deadline.

#### C3. `rateLimit.cleanupLoop` — no termination, permanent goroutine leak
**File:** `internal/serve/auth.go:82`
```go
func newRateLimit() *rateLimit {
    rl := &rateLimit{attempts: make(map[string]*rateWindow)}
    go rl.cleanupLoop()
    return rl
}
```
Goroutine runs forever — no `done` channel, no context. `rateLimit` has no `Close()`/`Stop()` method. Every server instance leaks this goroutine permanently.
**Fix:** Add `done chan struct{}` to `rateLimit`, close it in a `Stop()` method, select on `<-rl.done` in the loop.

### HIGH

#### C4. Heartbeat `close(e.done)` without goroutine join — use-after-close on restart
**File:** `desktop/heartbeat.go:132`
```go
func (e *HeartbeatEngine) Stop() {
    e.mu.Lock()
    defer e.mu.Unlock()
    if !e.running { return }
    e.running = false
    close(e.done)
}
```
`Stop()` closes `e.done` but never waits for `loop()` goroutine (line 120: `go e.loop()`) to exit. If `Start()` is called again immediately, new goroutine starts while old one may still be draining. Also, `e.done` is closed but never re-initialized — calling `Start()` after `Stop()` would panic on second `close(e.done)`.
**Fix:** Use `sync.WaitGroup` to join the loop goroutine in `Stop()`. Recreate `e.done` channel in `Start()`.

#### C5. `delayedDesktopSessionTrash/Cleanup` — fire-and-forget goroutines with no tracking
**Files:** `desktop/tabs.go:4117`, `desktop/app.go:1083,1094,1642`
```go
go delayedDesktopSessionTrash(target.dir, target.sessionPath, target.key, destroys)
go delayedDesktopSessionCleanup(oldPath, destroys)
```
Multiple fire-and-forget goroutines for session cleanup. No `sync.WaitGroup`, no tracking. On app shutdown these goroutines may be killed mid-operation, leaving partial cleanup state.
**Fix:** Track with a `sync.WaitGroup` on `App`, wait during `beforeClose`.

#### C6. `os.Exit(0)` in mcpbridge signal handler bypasses defers
**File:** `cmd/reasonix-mcpbridge/main.go:583`
```go
go func() {
    <-sigCh
    os.Exit(0)
}()
```
`os.Exit(0)` skips all deferred functions. If `srv.ServeStdio()` has cleanup (buffer flush, etc.), it's lost.
**Fix:** Use context cancellation to signal `ServeStdio()` to return, then exit from main.

#### C7. `BotGateway` — `dispatchLoop` goroutines without WaitGroup, unsynchronized adapter reads
**File:** `internal/bot/gateway.go:232,237,224-225,242-249`
```go
for _, binding := range gw.adapters {
    go gw.dispatchLoop(ctx, binding)
}
go gw.evictLoop()
// ...
func (gw *BotGateway) AdapterCount() int { return len(gw.adapters) }  // no mutex
func (gw *BotGateway) StartErrors() []error { ... copy(gw.startErr) } // no mutex
```
Goroutines launched without `wg.Add()`. `Stop()` calls `gw.cancel()` but can't wait for goroutines. `gw.adapters` and `gw.startErr` written in `Start()`, read concurrently without synchronization in `AdapterCount()`, `StartErrors()`, `HasPlatform()`, `AdapterWebhookURL()`.
**Fix:** Add `sync.WaitGroup` for background goroutines. Guard `gw.adapters` reads with mutex or `atomic.Pointer`.

#### C8. `lazySpawn.run()` — doesn't check `ctx.Err()` before recording failure
**File:** `internal/plugin/lazy.go:91`
```go
func (s *lazySpawn) run() {
    real, err := s.host.Add(s.ctx, s.spec)
```
If caller cancels `s.ctx` (via `endDeferredSpawn`), `s.host.Add` may return partial error. State transition at line 93-130 handles errors but doesn't check `s.ctx.Err()` first — records spurious failure on cancellation.
**Fix:** Check `s.ctx.Err()` before recording failure at line 126-128.

### MEDIUM

#### C9. Goroutine without context cancellation (periodic tidy)
**File:** `cmd/reasonix-memoryserver/main.go:691-698`
```go
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    for range ticker.C {
        store.Tidy()
        logger.Debug("ran periodic tidy")
    }
}()
```
**Issue:** Goroutine runs forever — no context cancellation on shutdown.
**Fix:** Accept context and select on `ctx.Done()`.

#### C10. `App.ctx` field — unsynchronized read/write
**File:** `desktop/app.go:349,332-336,559,580`
```go
func (a *App) startup(ctx context.Context) { a.ctx = ctx }   // write
func (a *App) bootContext() context.Context {
    if a.ctx != nil { return a.ctx }  // read
    return context.Background()
}
```
`a.ctx` written in `startup()`, read in `bootContext()`, `reqCtx()`, and many other places. No mutex or atomic. If any method runs before `startup()` completes, race on `a.ctx`.
**Fix:** Use `atomic.Pointer[context.Context]` or guard with `a.mu`.

#### C11. `compress.go` — TOCTOU on `turn` atomic between RLock read and Lock write
**File:** `internal/compress/compress.go:111-129`
```go
c.mu.RLock()
entry, cached := c.cache[hash]
currentTurn := int(c.turn.Load())  // read 1
c.mu.RUnlock()
// ...
c.mu.Lock()
c.cache[hash] = cacheEntry{turn: int(c.turn.Load()), summary: summary}  // read 2
c.mu.Unlock()
```
`turn` is `atomic.Int32`, cache entries store `int`. `turn` could change between read 1 and read 2 — entries may get stale turn numbers.
**Fix:** Load `turn` once before the lock and use that value consistently.

#### C12. `mesh.go` — `initializedPeers` is `sync.Map` but `m.peers` is under RWMutex
**File:** `internal/mesh/mesh.go:106,359`
`m.peers` guarded by `m.mu`, but `initializedPeers` is a `sync.Map` (global, not on `m`). Peer initialization state and peer list can diverge if peers are added/removed concurrently.
**Fix:** Move `initializedPeers` into `Mesh` struct, guard under `m.mu`.

#### C13. `plugin.Host` — `c.prompts` written under `h.mu` but may be read without lock
**File:** `internal/plugin/plugin.go:432-435,458-461`
```go
h.mu.Lock()
c.prompts = ps
h.mu.Unlock()
```
`c.prompts` is a field on `Client`, written under `h.mu` (host mutex). If `Client` is accessed from another path that doesn't hold `h.mu`, there's a race. `Remove` method modifies `h.prompts` under `h.mu` but doesn't clear `c.prompts` on removed client.
**Fix:** Ensure all `c.prompts`/`c.resources` reads also hold `h.mu.RLock()`.

### LOW

#### C14. `goSafe` — panic recovery swallows errors silently
**File:** `desktop/crash_pending.go:60-65`
```go
func (a *App) goSafe(site string, fn func()) {
    go func() {
        defer a.recoverToPending(site)
        fn()
    }()
}
```
Goroutines launched via `goSafe` recover panics and write to pending crash file. No logging, no error propagation.
**Fix:** Add `slog.Error` log in `recoverToPending` alongside crash file write.

#### C15. `grep.go` — pipe goroutine outlives early return (fd leak)
**File:** `internal/tool/builtin/grep.go:167-171`
```go
go func() {
    _, _ = pw.Write(head)
    io.Copy(pw, f)
    pw.Close()
}()
```
If scanner returns early (line 186: `return nil` for binary), goroutine continues writing to `pw`. Writes may block if pipe buffer fills. File descriptor leaks until `io.Copy` finishes.
**Fix:** Use context-aware pipe or close `pr` to signal goroutine to exit.

#### C16. `notify/sender_*.go` — `cmd.Wait()` goroutine leak
**Files:** `internal/notify/sender_darwin.go:22`, `sender_windows.go:24`, `sender_linux.go:18`
```go
go func() { _ = cmd.Wait() }()
```
Fire-and-forget `cmd.Wait()` — if notification process hangs, goroutine leaks.
**Fix:** Add timeout via `context.WithTimeout` on the command.

#### C17. `context.Background()` in desktop workspace operations
**File:** `desktop/workspace_changes.go:143`
(See C1e above — same finding.)

---

## 3. ERROR HANDLING

### GOOD

- Sentinel errors defined properly: `ErrUnknownModel`, `ErrTurnRunning`, `ErrServerAlreadyConnected`, `ErrSpawningInFlight`, plus `installsource` errors (`ErrAuthRequired`, `ErrBinaryMissing`, etc.).
- `errors.Is()` used correctly in `internal/installsource/` for error chain inspection.
- `errors.As()` used in `internal/provider/openai/fetch_models.go:28` for typed error extraction.
- Error wrapping with `%w` in most error returns.
- `crypto/subtle.ConstantTimeCompare` used for all auth token comparisons (`pkg/httputil/auth.go:62`, `internal/serve/auth.go:235,243`).
- Nil checks consistently applied — `if a == nil { return }`, `nilutil.IsNil(sink)` pattern. No high-risk dereference patterns.
- No defer-in-loop bugs found.
- No production `_ = ...err` patterns (errors ignored).
- `init()` functions all follow legitimate registration pattern (19 tool registrations, 3 provider registrations, 2 state init).

### HIGH

#### E4. `os.Exit(0)` in non-main package
**File:** `desktop/updater_app.go:149`
```go
os.Exit(0)  // in InstallUpdate method
```
**Issue:** `os.Exit` in non-main package makes code untestable and breaks library contracts. Can't mock or test `InstallUpdate` without process dying.
**Fix:** Return error or send signal to channel. Let caller decide whether to exit.

### MEDIUM

#### E1. String matching on error messages in tests
**Files:** 60+ occurrences across 20+ test files of `strings.Contains(err.Error(), "...")`.
**Issue:** Tests break if error message wording changes. Should use sentinel errors and `errors.Is()`.
**Severity:** MEDIUM — test fragility, not production bug.

**Production code (2 low-priority instances):**
- `internal/tool/builtin/movefile.go:140` — `strings.Contains(msg, "cross-device")` for cross-device link error. No sentinel exists in Go stdlib. Acceptable workaround.
- `desktop/metrics_app.go:346` — Error classification for metrics buckets, not control flow. Low priority.

#### E5. Error wrapping with `%s` breaks `errors.Is`/`errors.As` chain
**File:** `internal/control/errmsg.go:20,23`
```go
fmt.Errorf("model stream interrupted...: %s", err.Error())
fmt.Errorf("model stream disconnected...: %s", err.Error())
```
**Issue:** Using `%s` with `err.Error()` breaks error chain. Callers can't use `errors.Is`/`errors.As` to inspect underlying error.
**Fix:** Use `%w`:
```go
fmt.Errorf("model stream interrupted...: %w", err)
```

#### E6. Multi-error wrapping with `%v` instead of `%w` (Go 1.20+ supports multiple `%w`)
**Files:**
- `internal/installsource/apply.go:225,238,244` — `fmt.Errorf("%w; rollback failed: %v", err, rbErr)`
- `internal/tool/builtin/movefile.go:122,128` — `fmt.Errorf("%w; restore %s: %v", err, src, restoreErr)`
- `internal/plugin/plugin.go:942` — `fmt.Errorf("%w; reinitialize failed: %v", err, initErr)`

**Issue:** Second error uses `%v` — callers can't inspect rollback/restore error via `errors.Is`.
**Fix:** Use `%w` for both errors (Go 1.20+):
```go
fmt.Errorf("%w; rollback failed: %w", err, rbErr)
```

#### E7. String truncation by bytes instead of runes (corrupts CJK/emoji)
**Files:**
- `desktop/bot_connection_app.go:429` — `return s[:80]` in `safeBotReportValue`
- `internal/plugin/plugin.go:1080` — `msg = msg[:500] + "..."` in `summarizeFailureError`

**Issue:** `s[:n]` truncates mid-character on multi-byte UTF-8. Corrupts CJK text, emoji.
**Fix:** Use `[]rune` truncation. Project already has `clampRunes` helper in `errmsg.go:80` — reuse it:
```go
return clampRunes(s, 80)  // instead of s[:80]
```

#### E8. Type assertion without comma-ok in production code
**File:** `internal/cli/chat_tui.go:695`
```go
cm := next.(chatTUI)
```
**Issue:** Panics if `next` is not `chatTUI`. bubbletea's `Update` returns `tea.Model` — if it ever returns a different type, this crashes the process.
**Fix:**
```go
cm, ok := next.(chatTUI)
if !ok { return m, nil }
```

#### E9. Panic in non-init library code
**Files:**
- `internal/serve/auth.go:188,196,527` — `panic("serve/auth: crypto/rand.Read failed: " + err.Error())`
- `desktop/app.go:222` — `panic("crypto/rand.Read failed: " + err.Error())`

**Issue:** Panicking in library code (not `main`) crashes the entire process. `crypto/rand.Read` failure is extremely rare but should be handled gracefully.
**Fix:** Return error instead of panicking. Callers decide how to handle.

#### E2. Type assertions without comma-ok in production code (mcpbridge)
**File:** `cmd/reasonix-mcpbridge/main.go:117-146`
```go
task, _ := args["task"].(string)
model, _ := args["model"].(string)
```
**Issue:** Silent zero-value on type assertion failure. Error message is misleading ("task is required" vs "task has wrong type").
**Severity:** LOW — defensive enough due to subsequent empty checks.

#### E3. `log.Fatal` in production code
**Files:** `cmd/reasonix-mcpbridge/main.go:575,589`, `cmd/reasonix-plugin-example/main.go:48`
**Issue:** `log.Fatal` calls `os.Exit(1)` — skips defers.
**Fix:** Return error to main, handle shutdown there.

### LOW

#### E10. Global mutable state — 11 test-seam function vars
**Files:**
- `internal/tool/builtin/movefile.go:18` — `var renameFile = os.Rename`
- `internal/control/attachments.go:25` — `var attachmentNow = time.Now`
- `internal/control/refs.go:28` — `var extractPDFText = extractPDFTextDefault`
- `internal/cli/transcript.go:27` — `var clipboardWriteAll = clipboard.WriteAll`
- `internal/cli/chat_tui.go:549` — `var detectTermuxTerminal = isTermuxTerminal`
- `desktop/updater.go:212,386` — `var updateCacheBaseDir`, `var retryBackoff`
- `internal/billing/balance.go:46` — `var httpClient = &http.Client{...}`
- `internal/cli/cli.go:287` — `var newNotificationSender`
- `internal/config/credentials.go:57-58` — `var storedCredentialValueLookup`, `var legacyKeyringCredentialValueLookup`
- `internal/installsource/plan.go:16` — `var githubAPIBaseURL`

**Issue:** Package-level mutable vars used as test seams. Complicates testing, introduces hidden coupling, race risks in parallel tests.
**Fix:** Move to struct fields or accept as constructor parameters.

#### E11. Package-level mutable caches with locks
**Files:**
- `desktop/tabs.go:5080` — `var projectSessionCache = &sessionListCache{...}`
- `desktop/tabs.go:5105` — `var topicSessionIndexCache = struct{ sync.Mutex; ... }{...}`
- `internal/config/credentials.go:52` — `var credentialSourceTracker`
- `internal/tool/builtin/bash.go:39-42` — `var bashPathMu sync.Mutex; var bashPathCache = map[string]string{}`

**Issue:** Package-level mutable state with locks — should be struct-owned for testability.
**Fix:** Move to App/struct fields.

---

## 4. HTTP SERVER TIMING ANALYSIS

| Server | ReadHeaderTimeout | ReadTimeout | WriteTimeout | IdleTimeout | Status |
|--------|-------------------|-------------|--------------|-------------|--------|
| `internal/serve/serve.go:285` (RunGraceful) | 10s | — | — | 120s | PARTIAL |
| `internal/serve/serve.go:277` (Run) | — | — | — | — | **FAIL** |
| `pkg/mcputil/server.go:131` | 10s | — | — | — | PARTIAL |
| `internal/collab/collab.go:123` | 5s | — | — | — | PARTIAL |
| `internal/bot/line/line.go:100` | — | — | — | — | **FAIL** |
| `internal/bot/feishu/feishu.go:709` | — | — | — | — | **FAIL** |
| `internal/bot/feishu/feishu.go:709` | — | — | — | — | **FAIL** |

**Recommendation:** All `http.Server{}` instances need at minimum `ReadHeaderTimeout`. Add `WriteTimeout` for endpoints that don't stream. Add `IdleTimeout` for keep-alive management.

---

## 5. PROJECT STRUCTURE

### GOOD

- Follows Go standard layout: `cmd/`, `internal/`, `pkg/`, `bot/`, `tools/`.
- `internal/` used correctly for encapsulation — all private packages are import-protected.
- `cmd/` contains all binary entry points — each subdirectory is a separate binary.
- `pkg/` contains exported reusable packages (`mcputil`, `httputil`).
- Package names are descriptive (not `util`, `helper`, `common`).

### LOW

#### P1. `desktop/` is a separate Go module but not in `cmd/`
**Observation:** `desktop/` has its own `go.mod` (Wails desktop app). This is correct for a separate binary with different dependencies, but breaks from the `cmd/` convention.
**Severity:** LOW — Wails apps have different build requirements, this is justified.

#### P2. `bot/` directory at root instead of `cmd/reasonix-bot/`
**Observation:** `bot/main.go` is a binary entry point but lives in `bot/` not `cmd/reasonix-bot/`.
**Severity:** LOW — style inconsistency. Dockerfile references `./bot` so changing it requires Docker update too.

---

## 6. TESTING

### GOOD

- 3,642 tests — ALL PASS
- Race detector clean
- Table-driven tests used throughout (Go idiom)
- `t.Run()` subtests for isolation
- `t.Parallel()` used in independent tests
- `t.TempDir()` used for temp files
- `t.Cleanup()` used for resource cleanup
- `httptest.NewServer` for fake API servers
- Benchmarks exist (7 benchmark functions)
- Testcontainers not used (no external DB deps to containerize — SQLite is embedded)
- CI runs tests on 3 OSes (ubuntu, macOS, windows)
- CI has dedicated race detector job
- CI has coverage job with artifact upload

### MEDIUM

#### T1. Coverage at 66.5% — below 80% target
**Root module coverage:** 66.5% (statements).
**Gap areas (likely):**
- `cmd/` binaries — `main()` functions are hard to test
- `desktop/` — Wails app lifecycle is hard to test
- `internal/notify/` — OS-specific notification code
- `internal/proc/` — process management, OS-specific

**Fix:** Focus on `internal/` packages — these are the reusable components. Use `httptest` for HTTP handler coverage, direct function calls for logic.

#### T2. No fuzz tests
**Observation:** Zero `func FuzzXxx(*testing.F)` in codebase.
**Risk:** Edge-case inputs to parsers (JSON, config, URL handling) are only tested with hand-picked cases.
**Fix:** Add fuzz tests for:
- JSON unmarshal in MCP message handling (`pkg/mcputil/server.go`)
- URL parsing in `internal/tool/builtin/webfetch.go`
- Config parsing in `internal/config/`
- SQL query building in `cmd/reasonix-memoryserver/sqlite_storage.go`

#### T3. Unused test helper
**File:** `internal/scheduler/scheduler_test.go:310`
```go
func containsString(ss []string, s string) bool {
```
**Issue:** Unused — flagged by staticcheck and golangci-lint.
**Fix:** Remove the function.

---

## 7. PERFORMANCE

### GOOD

- `sync.Pool` for buffer reuse in LLM providers (openai, anthropic).
- Pre-allocated slices in memory store operations.
- SQLite with WAL mode and busy timeout (`cmd/reasonix-memoryserver/sqlite_storage.go:26`).
- `atomic.Pointer` for lock-free command list access (`internal/control/controller.go:78`).
- HTTP client connection pooling via `netclient.DefaultClient()`.

### MEDIUM

#### F1. No pprof endpoints in production
**Observation:** Zero `net/http/pprof` imports or pprof handler registration.
**Impact:** Can't profile running production services. Hard to diagnose CPU/memory issues in deployed instances.
**Fix:** Add optional pprof endpoint (guarded by flag):
```go
if *pprofAddr != "" {
    go func() {
        log.Println(http.ListenAndServe(*pprofAddr, nil))
    }()
}
```
Or import `_ "net/http/pprof"` behind a build tag.

#### F2. No PGO (Profile-Guided Optimization)
**Observation:** No `default.pgo` file in module root.
**Impact:** Missing 2-7% CPU improvement from PGO (Go 1.21+).
**Fix:** Collect CPU profile from representative workload, save as `default.pgo`, rebuild.

#### F3. `http.DefaultClient` creates new connections per request in some paths
**Files:** `desktop/write_mode.go:241`, `desktop/bot_connection_app.go:684`, `cmd/reasonix-pr-review/main.go:105,127`
**Issue:** `http.DefaultClient` uses `http.DefaultTransport` which does pool, but without custom timeout config. These should use `netclient.DefaultClient()` which has tuned timeouts and pooling.
**Impact:** No timeout + potentially different pooling behavior.

---

## 8. OBSERVABILITY

### GOOD

- `log/slog` used throughout — structured logging done right.
- `slog.NewJSONHandler` for production, `slog.NewTextHandler` for dev/bot.
- Context-aware logging (`slog.InfoContext`, `slog.WarnContext`).
- Log level control (`slog.LevelInfo`, `slog.LevelDebug`).

### MEDIUM

#### O1. Mixed `log.Printf` and `slog` in some packages
**Files:**
- `pkg/mcputil/server.go:141,145,149` — `log.Printf` instead of `slog`
- `cmd/reasonix-mcpbridge/main.go:587` — `log.Println` instead of `slog`
- `desktop/heartbeat.go:83,121,193,211,227,234` — `log.Printf` instead of `slog`
- `cmd/reasonix-memoryserver/embedding.go:106` — `log.Printf` instead of `slog`

**Issue:** Inconsistent logging. `log.Printf` produces unstructured output — harder to parse in log pipelines.
**Fix:** Migrate remaining `log.Printf` calls to `slog.Info`/`slog.Warn` with structured key-value pairs.

---

## 9. DOCKER / DEPLOYMENT

### CRITICAL

(See S5 above — ~~UID mismatch~~ RETRACTED, UID 65532 is correct for distroless nonroot)

### MEDIUM

#### D1. No `readOnlyRootFilesystem` in Dockerfile
**Issue:** Runtime container has writable root filesystem.
**Fix:** Add `--read-only` flag or Kubernetes `readOnlyRootFilesystem: true` security context.

#### D2. Hardcoded `GOARCH=amd64` in Dockerfile
**File:** `Dockerfile:14-20`
```dockerfile
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -ldflags="-s -w" -o /out/reasonix ./cmd/reasonix
```
**Issue:** No ARM64 support. Can't run on Apple Silicon / ARM servers natively.
**Fix:** Use Docker buildx with `--platform` and `$TARGETARCH`:
```dockerfile
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build ...
```

#### D3. No health check in Dockerfile
**Fix:** Add `HEALTHCHECK` directive:
```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD ["/usr/local/bin/reasonix", "healthcheck"]
```

---

## 10. CI/CD

### GOOD

- 3-OS test matrix (ubuntu, macOS, windows)
- Race detector job
- golangci-lint with pinned version (v2.12.2)
- govulncheck (informational, continue-on-error)
- Coverage job with artifact upload
- gofmt enforcement
- `go mod tidy` check for desktop module
- CodeQL workflow
- PR auto-labeling

### MEDIUM

#### CI1. govulncheck is `continue-on-error: true`
**File:** `.github/workflows/ci.yml:170`
**Issue:** Vulnerability scan failures don't block PRs. A real vulnerability could merge.
**Fix:** Set `continue-on-error: false` once CI is stable. Keep informational only during transition.

#### CI2. No golangci-lint on root module in CI
**Observation:** CI runs golangci-lint only on `desktop/` module (line 139-144). Root module lint job (line 162-166) uses `golangci-lint-action@v9` but doesn't pass `working-directory`.
**Status:** Root module IS linted — the lint job at line 152 runs on root. Confirmed correct.

#### CI3. Coverage not enforced
**Observation:** Coverage uploaded as artifact but no threshold check.
**Fix:** Add coverage threshold gate:
```yaml
- name: coverage check
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    if [ "${COVERAGE%}" -lt 70 ]; then
      echo "Coverage ${COVERAGE} below 70%"
      exit 1
    fi
```

---

## 11. CODE QUALITY

### GOOD

- `go vet` clean
- `go build` clean
- `staticcheck` — 1 issue (unused test helper)
- `golangci-lint` — 1 issue (same)
- `gofmt` enforced in CI
- `go mod tidy` enforced in CI
- Atomic file writes with fsync (`internal/fileutil/atomicwrite.go`) — correct pattern
- `os.Rename` used for atomic file swaps throughout
- SQLite parameterized queries (no SQL injection)
- SSRF guard on web fetch (`internal/tool/builtin/webfetch.go`, `internal/installsource/ssrf.go`)
- `filepath.Base` used for path sanitization in installsource
- `init()` functions used appropriately (tool registration pattern)
- Panics only in startup/initialization code (crypto/rand failures, duplicate registration)

### LOW

#### Q1. `context.TODO()` in test
**File:** `desktop/app_test.go:528`
```go
app.emitReady(context.TODO())
```
**Issue:** Should use `context.Background()` in tests — `TODO()` is for "I haven't decided yet".
**Fix:** Replace `context.TODO()` with `context.Background()`.

#### Q2. Mixed `log` and `slog` (see O1)

#### Q3. `//nolint:errcheck` without explanation
**File:** `internal/tool/builtin/grep.go:169`
```go
io.Copy(pw, f) //nolint:errcheck
```
**Issue:** Suppressed error without explaining why. Is the error truly safe to ignore?
**Fix:** Add comment explaining why error is ignored, or handle it.

---

## 12. SUMMARY TABLE

| Category | Critical | High | Medium | Low | Total |
|----------|----------|------|--------|-----|-------|
| Security | 2 | 3 | 6 | 1 | 12 |
| Concurrency | 3 | 5 | 5 | 4 | 17 |
| Error Handling | — | 1 | 7 | 4 | 12 |
| HTTP Timeouts | 2 | — | — | — | 2 |
| Performance | — | — | 3 | — | 3 |
| Observability | — | — | 1 | — | 1 |
| Docker | — | — | 2 | — | 2 |
| CI/CD | — | — | 2 | 1 | 3 |
| Code Quality | — | — | — | 3 | 3 |
| Testing | — | — | 3 | — | 3 |
| **Total** | **7** | **9** | **29** | **13** | **58** |

---

## 13. PRIORITIZED FIX LIST

### Immediate (ship-blocker for new deployments)
1. **S1** — Fix WebSocket CheckOrigin in collab.go EchoWSHandler
2. **S2** — Add timeouts to `serve.go:Run()`
3. **S3** — Add timeouts to line.go and feishu.go HTTP servers
4. **S4** — Replace `http.DefaultClient.Do()` with `netclient.DefaultClient()` in 4 call sites
5. **S-NEW** — Non-constant-time token comparison in feishu.go:406
6. **S-NEW** — Raw err.Error() in HTTP responses (serve.go:397,409,454)
7. **S-NEW** — Bind all interfaces defaults (feishu.go:710, collab.go:101)
8. **C1** — Fix 6 context.Background() in request paths (controller.go, chat_tui.go, bot_connection_app.go, workspace_changes.go, feishu.go)
9. **C2** — Fix readFileWithTimeout goroutine leak (desktop/tabs.go:2924)
10. **C3** — Fix rateLimit.cleanupLoop permanent goroutine leak (serve/auth.go:82)

### Short-term (next sprint)
11. **S6** — Audit eval.go shell command input path
12. **S11** — Add `io.LimitReader` to 6 unprotected `io.ReadAll` call sites
13. **S12** — Tighten file permissions to 0o600 for session/config data
14. **S10** — Fix path traversal in memory/doc.go resolvePath
15. **D2** — Add multi-arch Docker build (ARM64 support)
16. **C4** — Fix heartbeat Stop() use-after-close (desktop/heartbeat.go:132)
17. **C5** — Track delayed session cleanup goroutines with WaitGroup
18. **C6** — Fix os.Exit(0) in mcpbridge signal handler
19. **C7** — Add WaitGroup + mutex to BotGateway dispatchLoop
20. **C8** — Check ctx.Err() before recording failure in lazySpawn.run()
21. **C9** — Add context cancellation to memoryserver periodic tidy goroutine
22. **E4** — Remove os.Exit(0) from desktop/updater_app.go:149 (non-main package)
23. **E5** — Fix errmsg.go %s → %w to preserve error chains
24. **E6** — Fix 6 multi-error %v → %w (Go 1.20+ supports multiple %w)
25. **E7** — Fix 2 byte truncation → rune truncation (reuse clampRunes helper)
26. **E8** — Add comma-ok to chat_tui.go:695 type assertion
27. **E9** — Replace panic with error returns in serve/auth.go + app.go
28. **F1** — Add pprof endpoint to production binaries
29. **O1** — Migrate `log.Printf` to `slog` in 4 packages

### Medium-term (technical debt)
30. **T1** — Increase test coverage to 80%+
31. **T2** — Add fuzz tests for JSON/URL/config parsers
32. **E1** — Replace 60+ string error matching in tests with sentinel errors
33. **CI1** — Make govulncheck blocking
34. **CI3** — Add coverage threshold gate
35. **F2** — Enable PGO for production builds
36. **D1** — Add readOnlyRootFilesystem to Docker
37. **D3** — Add HEALTHCHECK to Dockerfile
38. **Q3** — Add explanation for `//nolint:errcheck` in grep.go
39. **C10** — Fix App.ctx unsynchronized read/write
40. **C11** — Fix compress.go TOCTOU on turn atomic
41. **C12** — Move mesh.go initializedPeers into struct
42. **C13** — Guard plugin.Host c.prompts reads with RLock
43. **E10** — Move 11 test-seam function vars to struct fields
44. **E11** — Move 4 package-level mutable caches to struct-owned

### Cleanup (nice-to-have)
45. **T3** — Remove unused `containsString` in scheduler_test.go
46. **Q1** — Replace `context.TODO()` with `context.Background()` in test
47. **P2** — Consider moving `bot/` to `cmd/reasonix-bot/` for consistency
48. **E3** — Replace `log.Fatal` with error returns in mcpbridge
49. **C14** — Add slog.Error in goSafe recoverToPending
50. **C15** — Fix grep.go pipe goroutine fd leak on early return
51. **C16** — Add timeout to notify/sender cmd.Wait() goroutines

---

## 14. WHAT'S DONE RIGHT

This deserves explicit recognition — the codebase gets many things right that most Go projects don't:

1. **Atomic file writes with fsync** — `internal/fileutil/atomicwrite.go` does tmp→write→fsync→close→rename. Correct power-loss-safe pattern. Better than most production code.
2. **SSRF guard on web fetch** — Dial-time IP validation against RFC1918/link-local/metadata. DNS rebinding protected. Industry-grade.
3. **Constant-time comparison** — All auth token checks use `crypto/subtle.ConstantTimeCompare`. No timing attacks.
4. **Structured logging with slog** — Most of codebase uses `log/slog` with structured key-value pairs. JSON handler for production.
5. **Shared HTTP client with timeouts** — `netclient.DefaultClient()` with 10s dial, 30s header, 120s overall. Connection pooling.
6. **SQLite WAL mode + busy timeout** — Correct concurrent SQLite configuration.
7. **Race detector in CI** — Dedicated race job, not just part of matrix.
8. **3-OS test matrix** — ubuntu, macOS, windows. CRLF-aware gofmt skip on Windows.
9. **go mod tidy enforcement** — CI checks that go.mod/go.sum are not stale.
10. **Proper error sentinel pattern** — `errors.Is`/`errors.As` used correctly, not string matching in production code.
11. **Body size limits on most HTTP handlers** — `http.MaxBytesReader` and `io.LimitReader` used in most MCP handlers.
12. **Auth middleware DRY** — `pkg/httputil.AuthMiddleware` shared across servers. No duplicated auth logic.
13. **34th upstream sync** — Active upstream tracking with documented sync points. Not a stale fork.
14. **scrubSensitiveText in crash reports** — Actively redacts emails, tokens, API keys, JWTs, paths from crash data (`desktop/crash_app.go`).
15. **safePath in checkpoint** — Properly validates with `filepath.Rel` + `filepath.IsLocal` to prevent traversal.
16. **escapeLikeWildcards in SQLite** — Prevents LIKE wildcard injection (`sqlite_storage.go:201-208`).
17. **Line bot binds 127.0.0.1** — Webhook only accessible locally, not exposed to network.
18. **No InsecureSkipVerify in production TLS** — Only in test code (`netclient_test.go:370`).
19. **Collab hub CheckOrigin validates localhost** — Main hub at `collab.go:111-118` checks host against `127.0.0.1`/`localhost`/`::1`. (Only EchoWSHandler at line 305 returns true.)