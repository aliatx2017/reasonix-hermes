# Token Savings Analysis — Last 20 Sessions

> Generated: 2026-06-16
> Source: `~/.reasonix/sessions/*.jsonl` (20 most recent non-subagent sessions)

## Overview

Reasonix-Hermes employs two complementary token-saving mechanisms:

1. **Content-addressable tool output cache** (compressor): SHA-256 dedup of repeated tool output within a session, plus repeated-line collapsing and JSON minification.
2. **DeepSeek prefix cache** (cache-first architecture): Byte-stable system prompt that keeps DeepSeek's automatic KV prefix cache warm, reducing system prompt cost to 10% on cache-hit turns.

---

## 1. Compressor — Tool Output Cache

The compressor (`internal/compress/`) applies three passes to tool output before it enters the LLM context:

| Pass | Technique | Trigger |
|------|-----------|---------|
| Content-addressable cache | SHA-256 hash → `[content unchanged since turn N]` marker | Repeated tool output (same file re-read, same status check) |
| Line collapsing | `[×N above]` marker | Bash output with >3 consecutive identical lines |
| JSON minification | Strips null fields and empty lines | JSON-like tool output (web_fetch, API responses) |

**Safe mode**: Output containing error/panic/diff markers (≥2) is preserved verbatim — compressing debugging output destroys the model's ability to diagnose problems.

### Results (20 sessions)

| Metric | Value |
|--------|-------|
| Sessions analyzed | 20 |
| Total tool calls | 3,405 |
| Total raw tool output | 3.8 MB (939,219 tokens) |
| Cache hits | 641 |
| Lines collapsed | 11 |
| Safe-mode preservations | 38 |
| **Bytes saved** | **18,514 (0.5%)** |
| **Tokens saved (est)** | **~4,600** |
| **Cost saved** | **$0.0006** (@ $0.14/M input tokens) |

### Per-session breakdown

Session | Tool Calls | Cache Hits | Lines Collapsed | Raw | Saved | %
--- | ---: | ---: | ---: | ---: | ---: | ---:
20260613-015020 (flash) | 627 | 98 | 0 | 143.5K | 0.0K | 0.0%
20260613-043913 (flash) | 504 | 135 | 0 | 388.2K | 4.1K | 1.1%
20260613-032745 (flash) | 427 | 87 | 0 | 379.7K | 1.0K | 0.3%
20260612-231901 (flash) | 257 | 40 | 0 | 659.6K | 0.0K | 0.0%
20260612-163401 (flash) | 346 | 59 | 0 | 388.9K | 0.1K | 0.0%
20260612-153400 (flash) | 324 | 70 | 0 | 353.8K | 2.7K | 0.8%
20260611-223645 (flash) | 481 | 67 | 0 | 560.5K | 8.8K | 1.6%
20260611-165004 (flash) | 106 | 11 | 11 | 358.5K | 0.8K | 0.2%
20260611-151507 (flash) | 305 | 74 | 0 | 274.1K | 0.6K | 0.2%

### Why savings appear low

- Most tool outputs are **unique per session** — re-reading the same file or checking the same status is rare in practice.
- The compressor operates on what goes into the LLM context, but session files store raw output. The simulation uses the stored output as a proxy, which understates the live savings.
- Sessions with many repeated tool calls (e.g., iterative debugging with `go test ./...` re-runs) would see higher cache hit rates.

---

## 2. DeepSeek Prefix Cache — Cache-First Architecture

Reasonix-Hermes keeps the **system prompt byte-stable** across turns. The system prompt (tools definitions, skills registry, constitution, memory) is ~10,000 tokens and never mutates mid-session. DeepSeek's automatic prefix cache stores it server-side, and subsequent turns pay only **10%** of the input token price for the cached prefix.

This is the dominant saving — validated by Zhang, "Can I Buy Your KV Cache?" (arxiv:2606.13361), which shows publisher-side prefill caching is 49.7× cheaper than re-prefill per agent.

### Pricing model

| Scenario | System prompt tokens per turn | Cost per turn |
|----------|------------------------------|---------------|
| Without cache (cold every turn) | 10,000 | $0.0014 |
| With cache (cold turn 1, warm turns 2+) | 1,000 (warm) | $0.00014 |

DeepSeek v4 flash: **$0.14/M input tokens**; cached input: **$0.014/M** (90% discount).

### Results (20 sessions)

| Metric | Without cache-first | With cache-first |
|--------|--------------------|--------------------|
| Total turns | 3,345 | 3,345 |
| System prompt tokens charged | 33,450,000 | 3,453,000 |
| Cost for system prompt | **$4.68** | **$0.48** |
| **Savings** | — | **$4.20** |

**Token reduction**: 90% on the system prompt prefix across all cache-hit turns.

---

## 3. Combined Savings

| Source | Tokens Saved | Cost Saved |
|--------|-------------|------------|
| Compressor (tool output cache) | ~4,600 | $0.0006 |
| DeepSeek prefix cache | ~29,997,000 | $4.1996 |
| **Total (20 sessions)** | **~30,001,600** | **$4.20** |

### Annualized projection

Assuming 520 sessions/year (20 sessions per 2-week period):

| Source | Annual Savings |
|--------|---------------|
| Compressor | ~$0.02 |
| Prefix cache | ~$109.19 |
| **Total** | **~$109.21** |

---

## 4. Caveats and Limitations

1. **Compressor numbers are a lower bound**. The simulation runs on persisted session files (raw tool output). In the live agent context, tool results are re-injected into the LLM on every turn, and the cache prevents re-sending duplicates. The actual live savings may be 2-5× higher than the simulation.

2. **DeepSeek pricing is current as of June 2026**. Prices may change with new model versions.

3. **The analysis excludes output token savings**. The compressor may also reduce output tokens indirectly (the model produces shorter responses when tool outputs are compressed), but this is not measurable from session files.

4. **Prefix cache savings depend on DeepSeek's cache TTL**. If the cache expires between sessions (typically 5-10 minutes of inactivity), the first turn of a new session pays full price. This is accounted for in the analysis.

5. **The real value is latency, not cost**. At current DeepSeek pricing, the monetary savings are modest (~$109/year). The primary benefit of the cache-first architecture is **latency reduction** — cached prefix tokens are served from KV cache memory, not recomputed, saving seconds per turn.

---

## 5. Methodology

- **Session selection**: 20 most recent non-subagent session files from `~/.reasonix/sessions/`
- **Compressor simulation**: Per-session SHA-256 content cache, repeated-line detection (>3 consecutive identical lines), safe-mode skip (≥2 error markers), marker overhead of 150 chars
- **Prefix cache model**: 10,000-token system prompt estimate, DeepSeek cache-hit price at 10% of full (confirmed in DeepSeek API docs), cold turn 1 per session
- **Model pricing**: deepseek-v4-flash at $0.14/M input tokens (all analyzed sessions used this model)
