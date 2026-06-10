---
name: debugger
description: Systematic debugging workflow: reproduce, isolate, fix, verify. Stack trace analysis and log correlation.
runAs: inline
allowedTools:
  - read_file
  - grep
  - glob
  - bash
  - edit_file
---

# Debugger

Systematic debugging workflow. Follow these steps in order — don't skip ahead.

## 1. Reproduce

- Understand the exact conditions that trigger the bug.
- Write a minimal reproduction case if possible.
- Document: what input, what expected output, what actually happens.

## 2. Isolate

- **Binary search**: comment out or disable half the system at a time to narrow the cause.
- **Log injection**: add temporary logging at key decision points to trace data flow.
- **Stack trace analysis**: read the stack trace bottom-up — where did execution start, and where did it fail?
- **Time-based**: did the bug appear after a specific commit? Use `git bisect`.

## 3. Diagnose

### Common Patterns

| Symptom | Likely Cause |
|---------|-------------|
| Nil/null pointer | Missing nil check, uninitialized variable |
| Index out of range | Off-by-one, empty collection |
| Deadlock | Mutex held across a channel send, lock order inversion |
| Race condition | Shared state without synchronization |
| Memory leak | Unclosed resources (files, connections), growing slice/map |
| Infinite loop | Exit condition never met, concurrent modification |
| Wrong output | Logic error, wrong operator, type coercion |
| Timeout | Blocking I/O without deadline, slow external dependency |

### Tools

- Use `grep` to trace where a variable is set and read.
- Use `ls` and `glob` to find related files.
- Run the code with extra logging, debug flags, or a debugger.

## 4. Fix

- Make the smallest possible change that fixes the root cause.
- Add a regression test that fails before the fix and passes after.
- Consider: does this fix expose a deeper problem?

## 5. Verify

- Run the reproduction case — does it pass now?
- Run the full test suite.
- Check: could this fix have broken anything else? Grep for callers.
