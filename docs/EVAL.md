# Session Evaluation & Comparison

> `reasonix eval compare <session-a> <session-b>` — compare two agent sessions
> structurally for eval-driven development.

## Why Compare Sessions?

When you change your config, system prompt, model, or tool set, you want to know
whether the agent behaves **better** or **worse**. Session comparison gives you a
structured diff — turns, tool usage, token consumption, cost — so you can quantify
regressions and improvements.

## Quick Start

```bash
# Compare two saved session transcripts
reasonix eval compare .reasonix/sessions/session-01.json .reasonix/sessions/session-02.json
```

Output:
```
=== Session Comparison ===
A: .../sessions/session-01.json
B: .../sessions/session-02.json

--- Stats ---
Turns:      6 vs 8  (Δ +2)
Tokens in:  45000 vs 52000  (Δ +7000)
Tokens out: 32000 vs 38000  (Δ +6000)
Cost:       0.0180 vs 0.0210  (Δ +0.0030 ¥)

--- Tool Usage ---
  bash                 3 vs  5  (Δ +2)
  edit_file           12 vs 15  (Δ +3)
  read_file            8 vs  7  (Δ -1)
  write_file           4 vs  6  (Δ +2)

--- Turn-by-Turn ---
  Turn  1 ✓
  Turn  2 ✓
  Turn  3 ✗
        only in B: bash
  Turn  4 ✓
  Turn  5 ✓
  Turn  6 ✓
  Turn  7 ✗ (unpaired — only in B)
        only in B: read_file
  Turn  8 ✗ (unpaired — only in B)
        only in B: edit_file

  Matched: 5/8 turns

--- Similarity ---
  Jaccard (tool seq): 0.72
```

## What It Compares

The comparison tool loads saved session transcripts (JSON files in your
`.reasonix/sessions/` directory) and their `.sessionstats` sidecar files, then
produces:

| Dimension | What | Metric |
|-----------|------|--------|
| **Stats** | Aggregate session metrics | Tokens in/out, turns, cost, currency |
| **Tools** | Per-tool call counts | Count A vs B, delta |
| **Turns** | Paired turn-by-turn diff | Which tools called, matched/mismatched |
| **Similarity** | Structural overlap | Jaccard index of tool-call sequences (0–1) |

## Interpreting Results

- **Similarity > 0.8**: Sessions are structurally similar — same tool patterns.
- **Similarity 0.5–0.8**: Moderate drift — the agent is taking different approaches.
- **Similarity < 0.5**: Significant behavioral change — likely a config/system prompt
  change has altered the agent's strategy.

A decreasing similarity score after a model/planner change **isn't necessarily bad** —
the agent may be finding more efficient paths. Look at the token and cost deltas to
judge whether the change is an improvement.

## Desktop

The Wails desktop app exposes `CompareSessions(pathA, pathB)` as a binding. A
comparison panel in the Hermes Dashboard lets you pick two sessions and view the
structured diff inline.

## How It Works

`internal/eval/` — loads session transcripts, parses agent turns (user→assistant
pairs), extracts tool calls, and computes:

1. **Stats diff**: Simple numeric comparison of session metadata
2. **Tool diff**: Multi-set difference of tool call counts
3. **Turn diff**: Paired matching by index, showing which tools differed
4. **Jaccard similarity**: Set overlap of the ordered tool-call sequence

Sessions are compared **structurally** (tool choices, not output content). This
is intentionally different from comparing the final result — it tells you whether
the agent is *thinking differently*, not whether it got the right answer.
