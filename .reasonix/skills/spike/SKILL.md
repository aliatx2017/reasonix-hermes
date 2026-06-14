---
name: spike
description: "Throwaway experiments to validate an idea before building. Write exploratory code in a temporary directory, gather data, then discard — never commit spike code."
version: 1.0.0
author: Hermes Agent (adapted for Reasonix-Hermes)
tags: [spike, prototype, experiment, feasibility, exploration, proof-of-concept]
---

# Spike

## When to Use

- You need to test whether an approach will work before committing to it
- You're unsure about API behavior, library compatibility, or algorithm performance
- You want to gather data about a problem space before designing a solution
- A plan step says "validate approach X"

## Core Rules

1. **Throwaway code** — Write in `/tmp/spike-<name>/`, never in the project tree
2. **Time-boxed** — 15-30 minutes max, then decide
3. **No commits** — Spike code is never committed. Extract learnings, discard code.
4. **One question per spike** — "Will library X handle Y?" not "Build feature Z"

## Process

```
1. State the question clearly
2. Create /tmp/spike-<name>/
3. Write minimal code to answer the question
4. Run it, observe results
5. Report: what worked, what didn't, what you learned
6. rm -rf /tmp/spike-<name>/
```

## Go-Specific

```bash
mkdir -p /tmp/spike-<name>
cd /tmp/spike-<name>
go mod init spike
# Write main.go
go run .
```

## Output

A concise report:
- **Question:** What were you testing?
- **Approach:** What did you try?
- **Result:** What happened?
- **Decision:** Go / No-Go / Need more info
- **Artifacts:** Any data or logs worth keeping (move to project docs, not code)

## When NOT to Spike

- You already know the answer (just build it)
- The question needs a full implementation to answer (spike won't help)
- You're using the spike as an excuse to avoid making a decision

## Related

- Project skill: `evidence-first-reasoning` — verify hypotheses discriminatively
- Project skill: `plan` (if adopted) — spike before committing to a plan
