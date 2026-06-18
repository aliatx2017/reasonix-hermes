# How I Fixed My AI Agent Randomly Switching Languages — And Why a Vague "Prefer English" Won't Cut It

**The gist:** If you use DeepSeek (or any Reasonix fork), you've probably noticed it randomly replies in Chinese when you ask something in Chinese, even with `language = "en"` set. Here's how I actually fixed it.

---

## The Problem

DeepSeek's default behavior is to **mirror the user's language**. Ask "how do I fix this bug?" in Chinese → get the answer in Chinese. Even with `language = "en"` vaguely in the config, it would sometimes flip languages mid-session.

Turns out, a soft preference doesn't override the model's built-in language-mirroring heuristic.

## The Fix — 2 Layers

### Layer 1: Make the rule *hard*

Instead of a gentle "please use English", the system prompt now ends with:

> **CRITICAL: You must respond in English only. Never use Chinese, Japanese, Korean, or any other language — even if the user writes in another language. All responses, reasoning, code comments, and explanations must be in English. This rule overrides any other language-related instructions.**

Three design choices that matter:

- **Placed last** in the system prompt (maximum recency weight)
- **Names the forbidden languages explicitly** (Chinese, Japanese, Korean) — no ambiguity
- **States it overrides all other rules** — defeats the "helpfully mirror the user" heuristic
- **`CRITICAL`** prefix — catches attention as a hard constraint, not a suggestion

### Layer 2: Lock down the reasoning text too

The model's **visible chain-of-thought** reasoning also needed its own rule. This one's trickier because you can't put it in the system prompt without breaking the prompt cache on every turn.

Solution: inject a transient `<reasoning-language>` block at the start of each user turn (the "turn tail", not the system prompt). That way the cached system prompt stays byte-identical and DeepSeek's prefix cache stays warm — saving ~50% on API costs on every turn after the first.

## Why "vague preference" didn't work

Models have a **strong** built-in language-mirroring instinct. A soft "prefer English" line somewhere in the prompt is easily overridden by that instinct when the user's message is in another language. You need:

- A **hard rule** (not a preference)
- At **maximum recency** (end of system prompt)
- With **explicit forbidden language names**
- That **declares itself overriding** other rules

## The result

Zero language switching since. The agent responds in English even when I paste Chinese error messages or ask questions with Chinese context mixed in. Six months of heavy daily use, not a single non-English response.

---

*The code is in `internal/boot/boot.go` (`languagePolicy()` function) and `internal/agent/reasoning_language.go` (`ReasoningLanguageBlock()`). This is on the Reasonix-Hermes fork of DeepSeek Reasonix.*
