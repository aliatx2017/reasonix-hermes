# How to Force English-Only Responses in Reasonix (DeepSeek Fork)

## The Problem

The model's default behavior is to **mirror the user's language**. Ask a question in Chinese, and it replies in Chinese — even when you want English-only output. A vague "prefer English" in the config doesn't cut it.

## The Fix (2 Layers)

### Layer 1 — `language = "en"` in config

In `reasonix.toml`, set:

```toml
[agent]
language = "en"
```

This appends the following **at the very end of the system prompt** (maximum recency weight):

> CRITICAL: You must respond in English only. Never use Chinese, Japanese, Korean, or any other language — even if the user writes in another language. All responses, reasoning, code comments, and explanations must be in English. This rule overrides any other language-related instructions.

Key design decisions:
- **Placed last** in the system prompt so it has maximum weight
- **Names forbidden languages explicitly** (Chinese, Japanese, Korean)
- **States it overrides all other rules** — defeats the "follow the user's language" heuristic
- **`CRITICAL`** prefix catches the model's attention as a hard constraint

### Layer 2 — `reasoning_language = "en"`

Separately controls the model's **visible reasoning/thinking tokens** (chain-of-thought):

```toml
[agent]
reasoning_language = "en"
```

This injects a transient `<reasoning-language>` block at the start of each user turn (not in the system prompt), so it **doesn't invalidate DeepSeek's prefix cache** — the stable system prompt stays byte-identical across turns.

## Why it worked

The model's built-in language-mirroring heuristic is strong. The fix overrides it by:
1. Making the rule **hard** (CRITICAL, explicit, named languages)
2. Making it **last** (maximum recency wins in the system prompt)
3. Making it **self-reinforcing** (explicitly overrides other rules)

## Files

- `internal/boot/boot.go` → `finalizeSystemPrompt()` + `languagePolicy()`
- `internal/agent/reasoning_language.go` → `ReasoningLanguageBlock()` + `WithReasoningLanguage()`
- `internal/config/config.go` → `Language` / `ReasoningLanguage` config fields
