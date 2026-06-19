---
name: cache-first-architecture
description: "Cache-first agent architecture — immutable prefix, append-only log, volatile scratch. Preserving DeepSeek prefix cache hit rates across turns."
version: 2.0.0
author: Reasonix-Hermes (rewritten with writing-great-skills principles)
tags: [cache, cost-optimization, architecture, constitution]
---

# Cache-First Architecture

The constitution encodes it: "Cache-first system prompt — byte-stable prefix across turns for DeepSeek prefix cache." This skill explains the architecture so you preserve cache hit rates when touching the controller, agent, or boot assembly.

## Three regions

```
IMMUTABLE PREFIX   — system prompt + tool schemas (frozen after first turn)
APPEND-ONLY LOG    — message history (never reorder/rewrite/edit)
VOLATILE SCRATCH   — reasoning_content, memory notes (ride turn tail, never prefix)
```

**Critical invariant**: Memory blocks (Hindsight, project memory) MUST NOT be appended to the system prompt mid-session. Every byte change in the prefix breaks the entire cache. The controller's `Compose` drains `pendingMemory` into the turn tail (a user message), never the prefix. Do not alter this.

## Cache economics

DeepSeek prefix cache: **50-120× cheaper on hit vs. miss.**

| | Cache Hit | Cache Miss | Output |
|---|-----------|------------|--------|
| v4-flash | $0.02/M | $1/M | $2/M |
| v4-pro | $0.025/M | $3/M | $6/M |

Current implementation achieves ~99.8% cache hit rate.

## Tool-call repair

The agent handles common LLM output issues without breaking the cache:
1. **Truncation** — repair unbalanced JSON from max_tokens cutoff
2. **Storm** — suppress duplicate (tool, args) calls within a turn
3. **Flatten** — detect deep schemas, present compactly

## Pitfalls

- **Memory blocks in system prefix** — single biggest cache killer. Route through turn tail.
- **History reordering** — any edit/reorder breaks append-only invariant. DeepSeek caches sequential prefixes.
- **Tool schema regeneration** — re-generating tool JSON every call produces different bytes → cache miss.
- **Mid-session prompt mutation** — constitution says "never mutate mid-session — ride the turn tail."

## Related

- `cost-aware-llm-pipeline` — flash-first routing, auxiliary offloading, budget thresholds (the routing layer above this cache foundation)
- Constitution rule: "Cache-first system prompt"
- `internal/control/controller.go` — `Compose`, `pendingMemory` drain
- `internal/compress/` — token compressor
- `REASONIX.md` — "Cache-stable prefix" research backing (Zhang, 2606.13361)
