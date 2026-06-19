---
name: cost-aware-llm-pipeline
description: "Cost optimization for LLM API usage — flash-first routing, auxiliary offloading, budget thresholds, and compressor leverage."
version: 2.0.0
author: Reasonix-Hermes (rewritten with writing-great-skills principles)
tags: [cost, optimization, llm, routing, budget]
---

# Cost-Aware LLM Pipeline

The flash-first pipeline. Route cheap, escalate only on complexity, offload to auxiliaries. The cache architecture (immutable prefix, append-only log, volatile scratch) is covered by `cache-first-architecture` — load that one for cache mechanics; this skill covers the routing layer above it.

## Flash-first routing

Default to the cheapest capable model. Escalate only when the task needs deeper reasoning:

| Task | Model | Why |
|------|-------|-----|
| Simple edits, grep, read | flash | Low complexity |
| Multi-file refactoring | pro | Needs reasoning depth |
| Plan mode | pro | Strategic thinking |
| Subagent review | pro | Code review depth |
| Subagent explore | flash | Read-only survey |
| Security review | pro | High-stakes analysis |

## Auxiliary offloading

Route cheap sub-tasks to auxiliary providers — saves main-model tokens without sacrificing quality:

```toml
[agent.auxiliary]
compression = "deepseek/deepseek-v4-flash"   # compaction summaries
vision = "gemini-3-flash-preview"            # image analysis
web_extract = "deepseek/deepseek-v4-flash"   # web page reading
```

What gets offloaded: compaction summarization, image understanding, HTML-to-text conversion. These are high-volume, low-complexity tasks — the main model shouldn't pay for them.

## Budget thresholds

```toml
[agent]
soft_compact_ratio  = 0.5   # warn at 50% context
compact_ratio       = 0.8   # compact at 80%
compact_force_ratio = 0.9   # force at 90%
max_steps = 0               # 0 = no cap, rely on compaction
planner_max_steps = 12      # cap planner exploration
```

Compaction is your budget enforcer. The compressor (`internal/compress/`) provides additional savings: SHA-256 content dedup, repeated-line collapse, JSON minification. Safe mode preserves errors and stack traces.

## Cost tracking

Per-session atomic counters (`sessTokensIn`, `sessTokensOut`, `sessCost`) on Agent. Persisted in `.sessionstats` sidecar. Desktop StatusBar shows live `cache%` and `cost` chips. CLI `/stats` shows session totals.

## Related

- `cache-first-architecture` — immutable prefix, append-only log, volatile scratch (the cache mechanics beneath this routing layer)
- `reasonix.example.toml` — `[agent.auxiliary]` and compaction settings
- `internal/compress/` — token compressor
- `internal/agent/` — session cost tracking
