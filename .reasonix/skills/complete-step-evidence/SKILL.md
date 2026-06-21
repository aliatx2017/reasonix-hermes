---
name: complete-step-evidence
description: How to sign off steps with complete_step without getting rejected for unverifiable evidence — run the command first, then cite it.
---

# Complete-Step Evidence Protocol

## The problem

`complete_step` evidence citations are checked against your actual command
history for the turn. If you cite a command you didn't literally run this
turn, the step is REJECTED. This causes loops: you write a plausible grep/test
command in the evidence, it gets rejected, you run it again, cite it again.

## The rule

**Run the exact command BEFORE calling complete_step. Never fabricate a
citation from memory.** The command string in the evidence must match a
bash invocation from this exact turn — character for character.

## Workflow

1. Decide what verification you need (grep for code, run a test, etc.)
2. **Run the bash command now** — in this turn, don't skip it
3. Call `complete_step` and cite that exact command (copy-paste the
   command string you just typed into bash)
4. If multiple verifications are needed, run each one as a separate
   bash call first, then cite all of them

## Common failure patterns

- Citing `grep -n WorkshopThreshold internal/boot/boot.go` but you only
  ran a combined `grep -n "WorkshopThreshold|..." internal/boot/boot.go`
  — these are different command strings → rejected
- Citing a test command from a previous turn that's now cached
- Citing a command you *planned* to run but didn't

## Anti-pattern

```
complete_step(evidence=[{command: "grep -n X file.go", ...}])
# REJECTED — grep -n X file.go was never run this turn
```

## Correct pattern

```
bash: grep -n X file.go
# output...
complete_step(evidence=[{command: "grep -n X file.go", ...}])
# ACCEPTED — command was literally run 2 seconds ago
```
