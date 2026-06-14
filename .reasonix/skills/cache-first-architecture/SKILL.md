---
name: cache-first-architecture
description: "Reasonix-Hermes cache-first agent architecture — immutable prefix, append-only log, volatile scratch, tool repair, telemetry. Core design principle for maximizing DeepSeek prefix cache hit rates."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [deepseek, cache, cost-optimization, reasonix, architecture, constitution]
---

# Cache-First Architecture — Reasonix-Hermes

This project's constitution encodes cache-first design: "Cache-first system prompt — byte-stable prefix across turns for DeepSeek prefix cache." This skill explains the architecture behind it so you can preserve cache hit rates when touching the controller, agent, or boot assembly.

## Trigger Conditions

Load when:
- Implementing or debugging prompt cache performance
- Modifying the controller's Compose or message pipeline
- Adding cost tracking for LLM API calls
- Diagnosing high token costs or low cache hit rates
- Touching the system prompt assembly in `internal/boot/` or `internal/control/`

## 4 Pillars

### Pillar 1: Cache-First Context

Three regions with strict invariants:

```
IMMUTABLE PREFIX   — system prompt + tool schemas (frozen after first turn)
APPEND-ONLY LOG    — message history (never reorder/rewrite/edit)
VOLATILE SCRATCH   — reasoning_content, memory notes (ride turn tail, never prefix)
```

CRITICAL: Memory blocks (Hindsight, project memory) MUST NOT be appended to system prompt mid-session. Every byte change in system prompt breaks the entire prefix cache. The controller's `Compose` method drains `pendingMemory` into the turn tail (a user message), never the prefix. Do not alter this invariant.

### Pillar 2: Tool-Call Repair

The agent should handle common LLM output issues:
1. **Truncation** — repair unbalanced JSON from max_tokens cutoff
2. **Storm** — suppress duplicate (tool, args) calls within a turn
3. **Flatten** — detect deep schemas, present compactly

### Pillar 3: Cost Control

- Flash-first default (deepseek-v4-flash), auto-escalate to pro on complexity
- Auxiliary calls (compression, vision) route to auxiliary provider if configured (`[agent.auxiliary]`)
- Auto-compaction triggers at configured `compact_ratio` / `compact_force_ratio` thresholds
- Compressor (`internal/compress/`) squashes repeated tool output

### Pillar 4: Telemetry

- Per-turn cache hit/miss tracking via `sessCacheHit` / `sessCacheMiss` atomic counters on Agent
- Cache economy gauge: `hit / (hit + miss)` percentage displayed in desktop StatusBar
- Session-level aggregation persisted in `.sessionstats` sidecar
- Cost tracking via provider pricing + token usage

## Implementation in Reasonix-Hermes

| Component | Location | Role |
|-----------|----------|------|
| System prompt assembly | `internal/boot/` | Freezes prefix after first turn; only Compose mutates via turn tail |
| Controller.Compose | `internal/control/controller.go` | Drains `pendingMemory` into turn tail, preserves prefix |
| Agent counters | `internal/agent/` | `sessCacheHit`, `sessCacheMiss`, `sessCost` — telemetry |
| Compressor | `internal/compress/` | SHA-256 content dedup, repeated-line collapse, JSON minification |
| Session stats | `.sessionstats` sidecar | Persisted aggregate stats (tokens/cost/turns) |
| Desktop cache gauge | `StatusBar` hermes chip | Live hit-rate display |

## DeepSeek Pricing

| Model | Cache Hit | Cache Miss | Output |
|-------|-----------|------------|--------|
| v4-flash | $0.02/M | $1/M | $2/M |
| v4-pro | $0.025/M | $3/M | $6/M |

50-120x difference hit vs miss. Cache-first is not optional.

## Pitfalls

- **Memory blocks in system prefix** — single biggest cache killer. Every tick changes content → cache miss. Route through turn tail via `pendingMemory`.
- **History reordering** — any edit/reorder of message history breaks append-only invariant. DeepSeek caches sequential prefixes.
- **Tool schema regeneration** — re-generating tool JSON every call produces different bytes → cache miss. Freeze after registration.
- **Mid-session prompt mutation** — constitution rule: "Never mutate the system prompt mid-session — ride the turn tail instead."

## Related

- Constitution rule: "Cache-first system prompt — byte-stable prefix across turns"
- Constitution rule: "Never mutate the system prompt mid-session — ride the turn tail instead"
- `internal/control/controller.go` — `Compose`, `pendingMemory` drain
- `internal/compress/` — token compressor
- `REASONIX.md` — "Cache-stable prefix" research backing (Zhang, 2606.13361)
