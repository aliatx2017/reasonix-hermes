---
name: performance-profiler
description: Identify performance bottlenecks, suggest optimizations, and measure impact with benchmarks.
runAs: inline
allowedTools:
  - read_file
  - grep
  - glob
  - bash
  - edit_file
---

# Performance Profiler

Identify and fix performance bottlenecks with a measurement-first approach.

## Golden Rule

**Never optimize without measuring first.** Always establish a baseline, then measure the impact of each change.

## Process

### 1. Measure (Baseline)
- Run existing benchmarks: `go test -bench=. -benchmem ./...`
- Or create a simple benchmark for the target code.
- Record: ops/sec, ns/op, B/op, allocs/op.

### 2. Profile (Find Bottlenecks)
- **CPU**: `go test -bench=. -cpuprofile=cpu.prof` → `go tool pprof cpu.prof`
- **Memory**: `go test -bench=. -memprofile=mem.prof` → `go tool pprof mem.prof`
- **Block**: look for mutex contention, channel blocking.
- **Trace**: `go test -trace=trace.out` → `go tool trace trace.out`

### 3. Diagnose

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| High B/op | Excessive allocations | Reuse buffers (`sync.Pool`), avoid `[]byte` to `string` conversions |
| High allocs/op | Many small allocations | Pre-allocate slices, use value types |
| High ns/op | Expensive operations in hot path | Cache results, move work out of loops |
| Goroutine leak | Unclosed channels, missing context cancel | `defer cancel()`, buffered channels |
| Lock contention | Coarse-grained locks | `sync.RWMutex`, sharded locks, lock-free structures |

### 4. Optimize
- Apply **one optimization** at a time.
- Re-run the benchmark after each change.
- If an optimization doesn't show measurable improvement, revert it.

### 5. Document
- Record the before/after metrics in a comment or commit message.
- If the optimization is non-obvious, add a comment explaining why it works.

## Go-Specific Quick Wins
- `make(T, 0, capacity)` over `make(T, 0)` when the final size is known.
- `strings.Builder` over `+=` for string concatenation in loops.
- `sync.Pool` for frequently allocated short-lived objects.
- Avoid `defer` in very hot loops (it has a small overhead).
- Pass large structs by pointer.
- Use `[]byte` over `string` when doing lots of transformations.
