# Session Evaluation & Comparison

> Eval-driven development toolkit — define test cases, run regressions, compare
> sessions, and track behavioral changes across config/model/prompt changes.

## Quick Reference

```bash
reasonix eval define <name>        # Define a new eval test case
reasonix eval check <name>         # Run a defined eval and check for regressions
reasonix eval report <name>        # Show detailed results for an eval
reasonix eval list                 # List all defined evals
reasonix eval clean <name>         # Remove eval artifacts
reasonix eval compare <a> <b>      # Compare two saved session transcripts
```

All commands also work as slash commands in the CLI TUI: `/eval define`, `/eval check`, etc.

## Define — Create an Eval Test Case

```bash
reasonix eval define my-bugfix-test
```

Creates `evals/my-bugfix-test/` with:
- `prompt.txt` — the input prompt to send to the agent
- `config.toml` — optional config overrides (model, tools, profile)

Edit `prompt.txt` with the task you want the agent to perform. Optionally edit
`config.toml` to pin a specific model or tool set.

The TUI `/eval define <name>` uses the workspace root as the evals directory.

## Check — Run an Eval

```bash
reasonix eval check my-bugfix-test
```

Runs the agent against `prompt.txt` using the specified config, captures the
result (turns, tools used, tokens, cost), and compares against a saved baseline
if one exists. Reports:

```
=== Eval: my-bugfix-test ===
Turns:       5
Tokens in:   42000
Tokens out:  31000
Cost:        0.0145 ¥
Pass@3:      ✓ (3/3 stable)
```

The pass@3 metric runs the test 3 times and checks for consistency — all 3
runs must produce the same tool-call pattern (not identical output, but the
same tool sequence). This catches flaky prompts and non-deterministic behavior.

If a baseline `.baseline.json` exists in the eval directory, `check` diffs
against it and flags any regressions in token usage or turn count.

## Report — Show Detailed Results

```bash
reasonix eval report my-bugfix-test
```

Prints the full results from the last `check` run:
- Turn-by-turn tool calls
- Token breakdown per turn
- Comparison against baseline (if any)
- Any tool-call differences across the 3 runs

## List — Show All Evals

```bash
reasonix eval list
```

Lists all defined evals with summary stats:
```
my-bugfix-test      (5 turns, pass@3=✓, last run 2026-06-16)
refactor-safety     (8 turns, pass@3=✗, last run 2026-06-15)
prompt-experiment   (no results yet)
```

## Clean — Remove Eval Artifacts

```bash
reasonix eval clean my-bugfix-test
```

Removes results and temporary files for the named eval but **keeps the
definition** (prompt.txt and config.toml). Does not delete the baseline.

To fully delete an eval, `rm -rf evals/<name>/`.

## Compare — Diff Two Sessions

```bash
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

## Eval Directory Structure

```
evals/
├── my-bugfix-test/
│   ├── prompt.txt          # The input prompt
│   ├── config.toml         # Optional config overrides
│   ├── .baseline.json      # Saved baseline for regression checks
│   └── results/            # Last check results
└── refactor-safety/
    ├── prompt.txt
    └── .baseline.json
```

The CLI uses `./evals/` relative to the current working directory. The TUI uses
the workspace root as the evals directory.
