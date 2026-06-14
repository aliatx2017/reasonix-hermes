---
name: evidence-first-reasoning
description: Evidence-first diagnostic reasoning — estimate ambiguity, generate hypotheses, verify discriminatively, refuse premature conclusions. Encodes the LLM-as-Investigator protocol (Marozzo & Liò, 2026).
---

# Evidence-First Reasoning

When asked to diagnose a problem, debug a failure, or investigate an issue, follow this protocol before reaching a conclusion:

## Protocol (from Marozzo & Liò, 2026)

1. **Estimate ambiguity** — state how many plausible explanations exist for the observed symptoms. If only one, note it and proceed. If multiple, list them explicitly.

2. **Generate candidate hypotheses** — enumerate the possible causes, ranked by likelihood given what you already know. For each: what evidence would confirm it, and what would rule it out.

3. **Targeted verification** — for each hypothesis with non-trivial probability, run a focused check (read a file, run a command, inspect a log) that discriminates between it and the alternatives. Do NOT run broad exploratory scans — each check should eliminate at least one hypothesis.

4. **Update probabilities** — after each check, explicitly state which hypothesis is now more or less likely. If no hypothesis has become dominant after three checks, ask the user a targeted clarification question rather than guessing.

5. **Refuse premature conclusions** — do not state a diagnosis or recommend a fix until one hypothesis is clearly dominant (>80% confidence). If the evidence is ambiguous, say so and explain what additional evidence would resolve it.

## Anti-patterns (from the paper)

- **Premature alignment**: adopting the user's suggested explanation without verification
- **Plausible-but-wrong**: proposing a fix that matches the symptoms superficially but misses the root cause
- **Evidence shopping**: cherry-picking confirmatory evidence while ignoring contradictory signals

## Output format

When investigating, structure your response as:
```
## Ambiguity assessment
[One sentence on how many explanations are plausible.]

## Hypotheses (ranked)
1. **[H1]** — evidence for: ... | evidence against: ...
2. **[H2]** — evidence for: ... | evidence against: ...

## Verifying [H1]
[Focused check + result + probability update.]

## Verifying [H2]
[Focused check + result + probability update.]

## Conclusion
[Diagnosis, or statement that more evidence is needed + what to collect.]
```
