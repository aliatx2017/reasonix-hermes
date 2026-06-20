
---

## 🐛 BUGS & CRITICAL ISSUES

### 1. **Argument Parsing Bug in** `cmd/reasonix-mcpbridge/main.go` **(Line 551)**

**Severity:** HIGH | **Type:** Logic Error

```go
if len(os.Args) > 2 && os.Args[1] == "--port" {
    port = os.Args[2]
}
```

**Problem:** This checks `os.Args[1] == "--port"` but doesn't verify `os.Args[1] == "--http"` is false. When called with `--http --port 9090`, the condition passes but then line 565 also checks for `--http` again. **The port flag is silently ignored when** `--http` **is NOT present** because the main `if` at line 565 only handles HTTP mode.

**Fix:** Restructure flag parsing to handle all combinations:

```go
func main() {
    port := "9090"
    httpMode := false
    
    for i := 1; i < len(os.Args); i++ {
        if os.Args[i] == "--http" {
            httpMode = true
        } else if os.Args[i] == "--port" && i+1 < len(os.Args) {
            port = os.Args[i+1]
            i++
        }
    }
    // ... rest of logic
}
```

---

### 2. **Unbound Context in** `orchestrateTask()` **(Line 252)**

**Severity:** MEDIUM | **Type:** Resource Leak

```go
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
defer cancel()
```

The context is created but the caller's `ctx` parameter (line 245 signature) is ignored. If the MCP bridge is shut down mid-orchestration, the goroutines won't be cancelled cleanly.

**Fix:**

```go
func (b *Bridge) orchestrateTask(ctx context.Context, task string) (string, error) {
    orchestrateCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
    defer cancel()
    // ... rest
}
```

---

### 3. **Path Traversal Vulnerability in** `findSkillFile()` **(Line 474)**

**Severity:** MEDIUM | **Type:** Security

```go
if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
    return "", fmt.Errorf("invalid skill name %q: path separators not allowed", name)
}
```

This check is **after** the name is used as a path component. While it does validate, it should be bulletproof. Consider using `filepath.Clean()` and validating it's still within the skills directory:

```go
func (b *Bridge) findSkillFile(name string) (string, error) {
    if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
        return "", fmt.Errorf("invalid skill name")
    }
    // Verify resolved path is within skillsDir
    for _, skillsDir := range b.skillDirs() {
        path := filepath.Join(skillsDir, name+".md")
        realPath, _ := filepath.Abs(path)
        realDir, _ := filepath.Abs(skillsDir)
        if !strings.HasPrefix(realPath, realDir+string(os.PathSeparator)) {
            continue // skip this dir
        }
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }
    return "", fmt.Errorf("skill not found")
}
```

---

### 4. **Silent Failure in** `cmd/reasonix-hooks/main.go` **(Line 52)**

**Severity:** LOW | **Type:** Error Handling

```go
data, err := io.ReadAll(os.Stdin)
if err != nil {
    fmt.Fprintf(os.Stderr, "[reasonix-hooks] read stdin: %v\n", err)
    os.Exit(1)  // ✅ Good
}

// ... later ...
body, _ := json.Marshal(req)  // ❌ Silently ignores marshal error
if err := postJSON(url, key, timeout, body); err != nil {
    fmt.Fprintf(os.Stderr, "[reasonix-hooks] retain: %v\n", err)
}
```

JSON marshal shouldn't fail with static payloads, but explicit error handling is better:

```go
body, err := json.Marshal(req)
if err != nil {
    fmt.Fprintf(os.Stderr, "[reasonix-hooks] marshal: %v\n", err)
    return
}
```

---

### 5. **Missing HTTP Status Validation in** `reasonix_run()` **(Line 348)**

**Severity:** MEDIUM | **Type:** Error Handling

In `callDeepSeek()`, the response body is read with a 4 MB limit but **no validation that the body is actually valid JSON before unmarshalling**. A 502 Bad Gateway might return HTML:

```go
body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
if err != nil {
    return "", fmt.Errorf("read API response: %w", err)
}

if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("DeepSeek API status %d: %s", resp.StatusCode, string(body))
}

var apiResp struct{ /* ... */ }
if err := json.Unmarshal(body, &apiResp); err != nil {
    // At least this is caught, but message is unclear
    return "", fmt.Errorf("parse API response: %w", err)
}
```

Better: Check Content-Type header:

```go
if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
    return "", fmt.Errorf("unexpected content type %q", ct)
}
```

---

## ⚠️ GAPS & ARCHITECTURAL ISSUES

### 6. **No Timeout on** `runReasonix()` **Process Cleanup**

**Severity:** MEDIUM | **Type:** Resource Management

```go
cmd := exec.CommandContext(ctx, "reasonix", args...)
// ... 
out, err := cmd.CombinedOutput()
```

**CombinedOutput() waits for the process**, but if the process ignores signals (e.g., stuck in a system call), the context timeout may not force termination. Add explicit signal handling:

```go
cmd := exec.CommandContext(ctx, "reasonix", args...)
// ... on context timeout, send SIGKILL
go func() {
    <-ctx.Done()
    if cmd.Process != nil {
        cmd.Process.Kill()
    }
}()
out, err := cmd.CombinedOutput()
```

---

### 7. **No Rate Limiting on MCP Bridge HTTP Server**

**Severity:** MEDIUM | **Type:** DoS Risk

```go
// Line 566
log.Fatal(srv.ServeHTTP(":"+port, "MCP_API_KEY"))
```

The HTTP server in `pkg/mcputil/` is not shown, but typical Go HTTP servers lack rate limiting. Without it:

- A malicious client can spawn unlimited goroutines
- Memory exhaustion is trivial
- No protection against slowloris attacks

**Recommendation:** Add middleware:

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(rate.Limit(100), 10) // 100 req/s, burst 10
// Use in HTTP handler
if !limiter.Allow() {
    http.Error(w, "rate limited", http.StatusTooManyRequests)
    return
}
```

---

### 8. **Concurrent Map Access in** `orchestrateTask()` **(Line 275)**

**Severity:** LOW-MEDIUM | **Type:** Race Condition

```go
results := make([]stepResult, len(steps))
var wg sync.WaitGroup
sem := make(chan struct{}, maxConcurrent)

for i, step := range steps {
    wg.Add(1)
    go func(idx int, stepDesc string) {
        // ...
        results[idx] = stepResult{...}  // ✅ Safe (different indices)
    }(i, step)
}
```

Actually **this is safe** because each goroutine writes to a unique index. But if `results` were a map, it would crash. Consider adding a comment:

```go
// Safe: each goroutine writes to a unique index
results[idx] = stepResult{...}
```

---

### 9. **Missing Connection Pool for DeepSeek API**

**Severity:** LOW | **Type:** Performance

```go
resp, err := netclient.DefaultClient().Do(req)
```

The `netclient.DefaultClient()` is called per-request. If it doesn't reuse TCP connections, each call spawns a new connection. **This is fine for low-frequency calls but wasteful at scale.**

**Recommendation:** Verify `netclient.DefaultClient()` uses `http.DefaultClient` or a shared client with keep-alive:

```go
var httpClient = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:       100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:    90 * time.Second,
    },
}
```

---

### 10. **No Versioning Metadata in MCP Bridge Tools**

**Severity:** LOW | **Type:** API Contract

The tools defined in `tools()` (lines 53–111) have no versioning or deprecation markers. If the schema changes, older MCP clients break silently.

**Recommendation:** Add version tags to tool descriptions:

```go
{
    Name:        "reasonix_run",
    Description: "Execute a one-shot task using Reasonix with DeepSeek. (v1, stable)",
    // ...
}
```

---

## 🚀 PERFORMANCE & EFFICIENCY ISSUES

### 11. **Inefficient Skill Discovery Loop (Line 510–529)**

**Severity:** LOW | **Type:** I/O Performance

```go
for _, skillsDir := range b.skillDirs() {
    entries, err := os.ReadDir(skillsDir)
    if err != nil {
        continue
    }
    for _, e := range entries {
        name := e.Name()
        if e.IsDir() {
            if _, err := os.Stat(filepath.Join(skillsDir, name, "SKILL.md")); err == nil {
                fmt.Fprintf(&out, "- %s\n", name)
            }
        }
        // ...
    }
}
```

**Issue:** For each directory, it calls `os.Stat()` separately. This is **N I/O syscalls per skill** when it could be 1.

**Fix:** Use `os.DirFS` or batch the checks:

```go
for _, e := range entries {
    if e.IsDir() {
        // Use e.Type() to check without stat
        if e.Type()&os.ModeDir != 0 {
            skillPath := filepath.Join(skillsDir, e.Name(), "SKILL.md")
            if data, err := os.ReadFile(skillPath); err == nil && len(data) > 0 {
                fmt.Fprintf(&out, "- %s\n", e.Name())
            }
        }
    }
}
```

---

### 12. **Memory Inefficiency: Large Response Buffering**

**Severity:** MEDIUM | **Type:** Memory Usage

```go
body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
// ... later ...
if err := json.Unmarshal(body, &apiResp); err != nil {
```

A 4 MB API response is entirely buffered into memory before parsing. For large skill files or orchestration results:

**Fix:** Stream JSON decoding:

```go
decoder := json.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024))
if err := decoder.Decode(&apiResp); err != nil {
```

---

### 13. **Quadratic String Building in** `parseSteps()` **(Line 382–404)**

**Severity:** LOW | **Type:** String Performance

```go
for _, line := range lines {
    // ...
    if currentStep.Len() > 0 && trimmed != "" {
        currentStep.WriteString(" ")
        currentStep.WriteString(trimmed)  // ✅ OK (using strings.Builder)
    }
}
```

Actually **this is fine** — `strings.Builder` avoids the O(n²) issue. The code is efficient.

---

## 💡 ENHANCEMENT & IMPROVEMENT OPPORTUNITIES

### 14. **Add Structured Logging Throughout**

Currently uses basic `log` package. Upgrade to `slog` for context and levels:

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

logger.Info("bridge started",
    "port", port,
    "workDir", b.workDir,
)
```

---

### 15. **Implement Health Check Endpoint**

Add `/health` endpoint to MCP bridge:

```go
func (h *handler) Health(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "status": "ok",
        "bridge_version": version,
        "uptime_seconds": time.Since(startTime).Seconds(),
    })
}
```

---

### 16. **Add Telemetry for Tool Invocations**

Track which tools are called, latency, and errors:

```go
type ToolMetrics struct {
    Name             string
    CallCount        int64
    TotalLatencyMs   int64
    ErrorCount       int64
    LastInvokedAt    time.Time
}

// In handle():
start := time.Now()
result, err := b.handle(name, args)
latency := time.Since(start).Milliseconds()
metrics[name].TotalLatencyMs += latency
if err != nil {
    metrics[name].ErrorCount++
}
```

---

### 17. **Graceful Shutdown Handler**

Add context cancellation and cleanup:

```go
func main() {
    // ... setup ...
    
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        sig := <-sigCh
        logger.Info("shutting down", "signal", sig)
        // Close DB connections, flush logs, etc.
        os.Exit(0)
    }()
    
    // Start server
}
```

---

### 18. **Caching Layer for Skill Discovery**

Skills rarely change during a session. Cache them:

```go
type Bridge struct {
    workDir      string
    apiBase      string
    skillCache   map[string]string  // name → body
    skillMutex   sync.RWMutex
    cacheExpiry  time.Time
}

func (b *Bridge) listSkills(ctx context.Context) (string, error) {
    b.skillMutex.RLock()
    if time.Now().Before(b.cacheExpiry) && b.skillCache != nil {
        defer b.skillMutex.RUnlock()
        // Return cached
    }
    b.skillMutex.RUnlock()
    
    // Rebuild cache
}
```

---

### 19. **Implement Retry Logic with Exponential Backoff**

DeepSeek API calls can be transient. Add resilience:

```go
func (b *Bridge) callDeepSeekWithRetry(ctx context.Context, ...) (string, error) {
    for attempt := 0; attempt < 3; attempt++ {
        result, err := b.callDeepSeek(ctx, ...)
        if err == nil {
            return result, nil
        }
        if attempt < 2 {
            backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return "", ctx.Err()
            }
        }
    }
    return "", fmt.Errorf("max retries exceeded")
}
```

---

### 20. **Add Request Validation Middleware**

Validate MCP request payloads before processing:

```go
func validateRequest(name string, args map[string]any) error {
    switch name {
    case "reasonix_run":
        if task, ok := args["task"].(string); !ok || task == "" {
            return fmt.Errorf("task is required and must be a string")
        }
        if model, ok := args["model"].(string); ok && model != "" {
            if !isValidModel(model) {
                return fmt.Errorf("unknown model: %s", model)
            }
        }
    }
    return nil
}
```

---

### 21. **Implement Circuit Breaker for DeepSeek API**

Prevent cascading failures if the API is down:

```go
type CircuitBreaker struct {
    state      string // "closed", "open", "half-open"
    failures   int
    lastError  time.Time
    threshold  int
    timeout    time.Duration
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == "open" {
        if time.Since(cb.lastError) > cb.timeout {
            cb.state = "half-open"
        } else {
            return fmt.Errorf("circuit breaker open")
        }
    }
    
    err := fn()
    if err != nil {
        cb.failures++
        if cb.failures >= cb.threshold {
            cb.state = "open"
            cb.lastError = time.Now()
        }
    } else {
        cb.failures = 0
        cb.state = "closed"
    }
    return err
}
```

---

### 22. **Add Configuration File Support**

Instead of only env vars, support `reasonix-bridge.toml`:

```toml
[server]
port = 9090
http = true
api_key = "${MCP_API_KEY}"

[deepseek]
base_url = "${DEEPSEEK_BASE_URL}"
model = "deepseek-v4-flash"
timeout_seconds = 60

[skills]
directories = [
    "~/.reasonix/skills",
    "./skills-hub/skills"
]
cache_ttl_seconds = 300
```

---

### 23. **Implement Dry-Run Mode**

Allow testing orchestration without execution:

```go
// In orchestrateTask
if args["dry_run"].(bool) {
    var sb strings.Builder
    sb.WriteString("# Orchestration Plan (Dry-Run)\n\n")
    for i, step := range steps {
        sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
    }
    return sb.String(), nil
}
```

---

### 24. **Add Structured Output Formats**

Support JSON, YAML, or Protobuf output:

```go
type OutputFormat string
const (
    OutputText OutputFormat = "text"
    OutputJSON OutputFormat = "json"
    OutputYAML OutputFormat = "yaml"
)

func (b *Bridge) formatOutput(result string, format OutputFormat) (string, error) {
    switch format {
    case OutputJSON:
        return json.Marshal(map[string]string{"result": result})
    case OutputYAML:
        // ...
    }
}
```

---

### 25. **Desktop App Opportunities**

From the README, the desktop enrichment is strong, but opportunities remain:

1. **Offline mode caching** — Cache recent skill, memory facts, and session transcripts
2. **Keyboard macro system** — Ctrl+Shift+R to rerun last task with variations
3. **Multi-window support** — Pin skills, memory, and goal progress in separate windows
4. **Live dashboard metrics** — Show API cost/token burn rate in real-time
5. **Session branching** — Visual "fork" UI to explore alternative paths
6. **Skill authoring UI** — Built-in editor for creating/testing new skills with hot-reload

---

## 🎯 SUMMARY TABLE

| Category | Issue | Severity | Fix Complexity |
| --- | --- | --- | --- |
| **Bugs** | Arg parsing in mcpbridge | HIGH | Low |
| **Bugs** | Path traversal in skill loading | MEDIUM | Medium |
| **Bugs** | Unbound context in orchestration | MEDIUM | Low |
| **Gaps** | No rate limiting on HTTP | MEDIUM | Medium |
| **Gaps** | No health checks | LOW | Low |
| **Perf** | Inefficient skill discovery I/O | LOW | Low |
| **Perf** | Large response buffering | MEDIUM | Medium |
| **Enhancement** | Add structured logging | MEDIUM | Medium |
| **Enhancement** | Add telemetry/metrics | MEDIUM | High |
| **Enhancement** | Circuit breaker for API | LOW | High |

---

## 🏆 Quick Wins (Start Here)

1. **Fix the** `--port` **argument parsing bug** (5 min)
2. **Add stricter path validation** (10 min)
3. **Use** `json.Decoder` **instead of** `ReadAll` **+** `Unmarshal` (10 min)
4. **Add** `/health` **endpoint** (15 min)
5. **Add signal handling for graceful shutdown** (20 min)

These five changes would significantly improve reliability and operational visibility.