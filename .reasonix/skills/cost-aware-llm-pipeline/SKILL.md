---
name: cost-aware-llm-pipeline
description: "Cost optimization patterns for LLM API usage — model routing by task complexity, auxiliary model offloading, budget enforcement, and prompt caching strategies."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [cost, optimization, llm, routing, caching, budget, token-economy]
---

# Cost-Aware LLM Pipeline

## When to Use

- Configuring model routing for cost optimization
- Setting up auxiliary models for offloading cheap tasks
- Diagnosing unexpectedly high API costs
- Tuning compaction and token economy settings

## Model Routing Strategy

### Flash-First Default

Use the cheapest capable model by default, escalate only when needed:

```
default_model = "deepseek"  # → deepseek-v4-flash (cheap)
Escalate to pro on: plan mode, complex multi-file edits, subagent review
```

### Auxiliary Model Offloading

Route cheap tasks to auxiliary providers to save main-model tokens:

```toml
[agent.auxiliary]
compression = "deepseek/deepseek-v4-flash"   # compaction summaries
vision = "gemini-3-flash-preview"            # image analysis
web_extract = "deepseek/deepseek-v4-flash"   # web page reading
```

**What gets offloaded:**
- **Compression** — Summarization of old turns during compaction
- **Vision** — Image understanding (screenshots, diagrams, photos)
- **Web extract** — Converting raw HTML to structured text

### Task Complexity Routing

| Task | Model | Why |
|------|-------|-----|
| Simple edits, grep, read | flash | Low complexity |
| Refactoring, multi-file | pro | Needs reasoning |
| Plan mode | pro | Strategic thinking |
| Subagent (review) | pro | Code review depth |
| Subagent (explore) | flash | Read-only survey |
| Security review | pro | High-stakes analysis |

## Cache Economics

DeepSeek prefix cache: **50-120x cheaper on cache hit vs. miss.**

```
Cache hit:  $0.02/M tokens
Cache miss: $1.00/M tokens  (50x more)
```

### Preserving Cache

1. **Immutable prefix** — System prompt + tool schemas frozen after first turn
2. **Append-only history** — Never reorder/rewrite/edit message history
3. **Volatile scratch** — Memory notes ride turn tail, never prefix
4. **Tool schema freeze** — Tool descriptions don't change mid-session

Cache hit rate target: >95%. Current implementation achieves ~99.8%.

## Budget Enforcement

### Compaction Thresholds

```toml
[agent]
soft_compact_ratio  = 0.5   # warn at 50% context
compact_ratio       = 0.8   # compact at 80%
compact_force_ratio = 0.9   # force at 90%
```

### Token Economy Settings

```toml
[agent]
max_steps = 0              # 0 = no cap (relies on compaction)
planner_max_steps = 12     # cap planner exploration
```

### Compressor

`internal/compress/` reduces repeated tool output:
- **SHA-256 content cache** — repeated output → compact reference
- **Repeated-line collapse** — bash output dedup
- **JSON minification** — strip nulls, collapse whitespace
- **Safe mode** — preserves errors, stack traces, diffs

## Cost Tracking

### Per-Session

- `sessTokensIn` / `sessTokensOut` — atomic counters on Agent
- `sessCost` — accumulated from provider pricing × token usage
- Persisted in `.sessionstats` sidecar

### Per-Turn

- `sessCacheHit` / `sessCacheMiss` — cache effectiveness
- StatusBar `cache%` chip in desktop
- CLI `/stats` shows session totals

## Price Reference

| Model | Cache Hit | Input | Output |
|-------|-----------|-------|--------|
| deepseek-v4-flash | $0.02 | $1 | $2 |
| deepseek-v4-pro | $0.025 | $3 | $6 |
| ollamacloud-flash | varies | varies | varies |

All prices per 1M tokens in USD.

## Related

- Project skill: `cache-first-architecture` — 4-pillar design
- `reasonix.example.toml` — `[agent.auxiliary]` and compaction settings
- `internal/compress/` — token compressor
- `internal/agent/` — session cost tracking
