# Reasonix Hermes — Master User Guide

<a href="../README.md">README</a>
&nbsp;·&nbsp;
<a href="./GUIDE.zh-CN.md">简体中文</a>
&nbsp;·&nbsp;
<a href="./SPEC.md">Spec</a>
&nbsp;·&nbsp;
<a href="../AGENTS.md">Project</a>

> Comprehensive day-to-day configuration and usage for Reasonix Hermes. Covers
> all upstream features plus Hermes extensions: Discord bot, MCP bridge server,
> Hindsight memory, skills hub, native hook runner, and portable mode.
> For the engineering contract and internals, see the **[Spec](./SPEC.md)**.

---

## Contents

1. [What is Reasonix Hermes](#1-what-is-reasonix-hermes)
2. [Installation](#2-installation)
   - 2.1 [Prebuilt (upstream binary)](#21-prebuilt-upstream-binary)
   - 2.2 [Build Hermes from source](#22-build-hermes-from-source)
   - 2.3 [Install the skills hub](#23-install-the-skills-hub)
3. [Getting Started](#3-getting-started)
   - 3.1 [Setup wizard](#31-setup-wizard)
   - 3.2 [Your first session](#32-your-first-session)
   - 3.3 [Project memory](#33-project-memory)
4. [Configuration](#4-configuration)
   - 4.1 [Resolution order](#41-resolution-order)
   - 4.2 [Minimal config](#42-minimal-config)
   - 4.3 [Full configuration reference](#43-full-configuration-reference)
5. [The Agent](#5-the-agent)
   - 5.1 [Chat mode](#51-chat-mode)
   - 5.2 [Run mode (one-shot)](#52-run-mode-one-shot)
   - 5.3 [Plan mode](#53-plan-mode)
   - 5.4 [Goal mode](#54-goal-mode)
   - 5.5 [Two-model collaboration](#55-two-model-collaboration)
   - 5.6 [Auto-plan](#56-auto-plan)
   - 5.7 [Subagent skills](#57-subagent-skills)
6. [Providers & Models](#6-providers--models)
   - 6.1 [Built-in presets](#61-built-in-presets)
   - 6.2 [Adding providers](#62-adding-providers)
   - 6.3 [Model references](#63-model-references)
   - 6.4 [Model switching](#64-model-switching)
7. [Built-in Tools](#7-built-in-tools)
   - 7.1 [File tools](#71-file-tools)
   - 7.2 [Search & navigation](#72-search--navigation)
   - 7.3 [Shell & execution](#73-shell--execution)
   - 7.4 [Meta tools](#74-meta-tools)
   - 7.5 [Tool registry](#75-tool-registry)
8. [Permissions & Sandbox](#8-permissions--sandbox)
   - 8.1 [Permission policy](#81-permission-policy)
   - 8.2 [Approval postures](#82-approval-postures)
   - 8.3 [Rule syntax](#83-rule-syntax)
   - 8.4 [Sandbox](#84-sandbox)
9. [Plugins / MCP](#9-plugins--mcp)
   - 9.1 [MCP client](#91-mcp-client)
   - 9.2 [Transport types](#92-transport-types)
   - 9.3 [.mcp.json support](#93-mcpjson-support)
   - 9.4 [MCP prompts & resources](#94-mcp-prompts--resources)
   - 9.5 [Managing MCP servers](#95-managing-mcp-servers)
10. [Slash Commands](#10-slash-commands)
    - 10.1 [Built-in commands](#101-built-in-commands)
    - 10.2 [Custom commands](#102-custom-commands)
11. [@ References](#11--references)
12. [Checkpoints & Rewind](#12-checkpoints--rewind)
13. [Memory System](#13-memory-system)
    - 13.1 [Hierarchical doc memory](#131-hierarchical-doc-memory)
    - 13.2 [Auto-memory (remember/forget)](#132-auto-memory-rememberforget)
    - 13.3 [Managing memory](#133-managing-memory)
14. [Skills](#14-skills)
    - 14.1 [Built-in skills](#141-built-in-skills)
    - 14.2 [Hermes skills hub](#142-hermes-skills-hub)
    - 14.3 [Skill lifecycle](#143-skill-lifecycle)
15. [Hooks](#15-hooks)
    - 15.1 [Upstream hook runner](#151-upstream-hook-runner)
    - 15.2 [Hermes native hook runner](#152-hermes-native-hook-runner)
16. [Hermes Extensions](#16-hermes-extensions)
    - 16.1 [Discord Bot](#161-discord-bot)
    - 16.2 [MCP Bridge Server](#162-mcp-bridge-server)
    - 16.3 [Hindsight Memory Server](#163-hindsight-memory-server)
    - 16.4 [Portable Mode](#164-portable-mode)
    - 16.5 [Skills Hub](#165-skills-hub)
17. [Desktop App](#17-desktop-app)
18. [Bot Gateway (Multi-Platform)](#18-bot-gateway-multi-platform)
19. [Troubleshooting & FAQ](#19-troubleshooting--faq)

---

## 1. What is Reasonix Hermes

**Reasonix Hermes** is an extended fork of [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix)
(synced to v1.7.0), the DeepSeek-native AI coding agent. Reasonix itself is a
**config- and plugin-driven** harness — a single static Go binary tuned around
DeepSeek's automatic prefix cache so token costs stay low across long sessions.

Hermes keeps the full upstream feature set (agent loop, providers, tools,
permissions, plugins, desktop app, bot gateway) and adds:

- **Discord bot** — `/goal` autonomous loop, `/model` switching, multi-platform gateway
- **MCP bridge server** — expose Reasonix to Claude Code, Codex, or any MCP client
- **Hindsight memory** — cross-session persistent memory with SQLite + vector search
- **17-skill community registry** — curated frontmatter playbooks
- **Native Go hook runner** — zero-dependency PreToolUse/Stop hooks
- **Portable mode** — run entirely from a USB drive or air-gapped machine

---

## 2. Installation

### 2.1 Prebuilt (upstream binary)

The quickest way to get a working Reasonix binary:

```sh
# Any OS — pulls the prebuilt native binary
npm i -g reasonix

# macOS
brew install esengine/reasonix/reasonix
```

Prebuilt archives (`darwin|linux|windows × amd64|arm64`) and `SHA256SUMS` are on
every [upstream GitHub release](https://github.com/esengine/DeepSeek-Reasonix/releases).

> **Note:** The upstream binary gives you the full Reasonix engine plus desktop
> app. Hermes extras (MCP bridge, memory server, Discord bot) require building
> from source (§2.2).

### 2.2 Build Hermes from source

```sh
git clone https://github.com/aliatx2017/reasonix-hermes.git
cd reasonix-hermes

# The main CLI
go build -o bin/reasonix ./cmd/reasonix

# Hermes extras
go build -o bin/reasonix-bridge ./cmd/reasonix-mcpbridge      # MCP bridge server
go build -o bin/reasonix-memory  ./cmd/reasonix-memoryserver   # Hindsight memory
go build -o bin/reasonix-bot     ./bot                         # Discord bot
go build -o bin/reasonix-hooks   ./cmd/reasonix-hooks          # Native hook runner
```

Requirements: **Go 1.25+**. No CGO needed (`CGO_ENABLED=0`). Cross-compile to
six targets with `make cross`.

### 2.3 Install the skills hub

```sh
# Via upstream install_source (preferred)
reasonix install-source install \
  --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills

# Or via script
bash scripts/install-skills.sh
```

This installs 17 curated skills into your project or user config. See
[§14.2](#142-hermes-skills-hub) for the full list.

### 2.4 Updating

```sh
# CLI self-update (fetches latest upstream release)
reasonix upgrade
reasonix update    # alias

# Check current version
reasonix --version
```

The `upgrade` command fetches the latest release from upstream (esengine/deepseek-reasonix)
and replaces the current binary. To update Hermes extensions (MCP bridge, memory server,
Discord bot, hooks), rebuild from source with `go build`.

---

## 3. Getting Started

### 3.1 Setup wizard

```sh
reasonix setup
```

Walks you through: pick a provider → enter API key → choose default model.
Writes a minimal `reasonix.toml` to the current directory.

### 3.2 Your first session

```sh
export DEEPSEEK_API_KEY=sk-...     # or put it in .env
reasonix chat                       # interactive TUI session
```

Once in chat you can:

| Action | How |
|--------|-----|
| Ask the agent to do something | Type your message and press Enter |
| See what tools are available | Type `/help` |
| Generate project memory | Type `/init` (creates AGENTS.md) |
| Run a one-shot task | `reasonix run "fix the failing test in auth_test.go"` |
| Pipe input | `echo "explain this function" \| reasonix run` |
| Use a different model | `reasonix run --model deepseek-pro "add unit tests"` |

### 3.3 Project memory

Reasonix loads hierarchical memory files at session start to understand your
project. The search order:

1. `REASONIX.md` in the current directory (committed, shared with team)
2. `REASONIX.local.md` (git-ignored, personal)
3. `~/.config/reasonix/REASONIX.md` (user-global)
4. `REASONIX.md` in ancestor directories
5. `AGENTS.md` (fallback name, same locations)

In chat, type `#<note>` to quick-add a line to the project REASONIX.md.
Use `/memory` to view and manage memory files. The `remember` tool saves
durable facts to an auto-memory store that loads on the next session.

Create an initial `AGENTS.md` tailored to your codebase by running `/init` in
chat — the agent analyzes your project and writes conventions, build commands,
and architecture notes.

---

## 4. Configuration

### 4.1 Resolution order

```
CLI flag > ./reasonix.toml > ~/.config/reasonix/config.toml > built-in defaults
```

Secrets come from the environment via `api_key_env` — **never written to config
files**. A `.env` file in the working directory is loaded if present.

### 4.2 Minimal config

```toml
default_model = "deepseek-flash"

[[providers]]
name        = "deepseek-flash"
kind        = "openai"
base_url    = "https://api.deepseek.com"
model       = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
```

### 4.3 Full configuration reference

```toml
# ── Top-level ──────────────────────────────────────────────────
default_model = "deepseek-flash"    # executor default
# language    = "zh"                # UI language; empty = auto-detect from $LANG

# ── Agent ──────────────────────────────────────────────────────
[agent]
max_steps           = 0             # executor tool-call rounds; 0 = no limit
planner_max_steps   = 12            # planner read-only rounds; 0 = no limit
planner_model       = "mimo-pro"    # optional second model for planning
subagent_model      = "deepseek-pro" # default for runAs=subagent skills
subagent_models     = { review = "deepseek-pro" }  # per-skill overrides
auto_plan           = "off"         # off|on — auto-enter plan mode for complex tasks
auto_plan_classifier = "deepseek-flash"  # cheap model for borderline classification
compact_ratio       = 0.8           # compact when prompt_tokens reach 80% of window
compact_keep        = 8             # verbatim messages kept after compaction

# ── Providers ──────────────────────────────────────────────────
[[providers]]
name        = "deepseek-flash"
kind        = "openai"
base_url    = "https://api.deepseek.com"
model       = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
# context_window = 131072  # override the default window size
# effort        = "high"   # reasoning effort sent in every request

[[providers]]
name        = "deepseek-pro"
kind        = "openai"
base_url    = "https://api.deepseek.com"
model       = "deepseek-v4-pro"
api_key_env = "DEEPSEEK_API_KEY"

[[providers]]
name        = "mimo-pro"
kind        = "openai"
base_url    = "https://token-plan-cn.xiaomimimo.com/v1"
model       = "mimo-v2.5-pro"
api_key_env = "MIMO_API_KEY"

# ── Tools ──────────────────────────────────────────────────────
[tools]
enabled               = []          # omit/empty = all built-ins
bash_timeout_seconds  = 120         # foreground safety cap; 0 = no tool-local cap

# ── Skills ─────────────────────────────────────────────────────
[skills]
paths            = ["~/my-skills"]   # extra custom skill roots
excluded_paths   = ["~/.agents/skills"]  # hide convention roots
disabled_skills  = ["review"]        # names of skills to suppress

# ── Permissions ────────────────────────────────────────────────
[permissions]
mode  = "ask"                                  # writer fallback: ask|allow|deny|auto
deny  = ["Bash(rm -rf*)", "Bash(git push*)"]   # always blocked
allow = ["Bash(go test:*)", "Edit(docs/**)"]   # never prompted

# ── Sandbox ────────────────────────────────────────────────────
[sandbox]
workspace_root = ""          # confine file-writers; empty = current dir
allow_write    = ["/tmp"]    # extra directories writers may touch
# [sandbox.bash]             # macOS Seatbelt for bash (see §8.4)
# network = true

# ── Plugins / MCP ──────────────────────────────────────────────
[[plugins]]
name    = "filesystem"
command = "npx"
args    = ["-y", "@modelcontextprotocol/server-filesystem", "/path"]

[[plugins]]
name    = "stripe"
type    = "http"
url     = "https://mcp.stripe.com"
headers = { Authorization = "Bearer ${STRIPE_KEY}" }

# ── Hermes: Hindsight memory as MCP plugin ─────────────────────
[[plugins]]
name    = "hindsight"
command = "./bin/reasonix-memory"
args    = ["--backend", "sqlite"]

# ── Hermes: Hooks ──────────────────────────────────────────────
[hooks]
pre_tool_use  = ["./bin/reasonix-hooks", "retain"]
stop          = ["./bin/reasonix-hooks", "reflect"]

# ── Hermes: Bot Gateway ────────────────────────────────────────
[bot]
enabled      = true
model        = "deepseek-flash"
max_steps    = 30
debounce_ms  = 1500
session_idle_timeout = "30m"

[bot.allowlist]
enabled        = true
discord_users  = ["123456789"]
discord_groups = ["987654321"]

[bot.discord]
token_env    = "DISCORD_BOT_TOKEN"
server_id    = "123456789"
channel_id   = ""
allow_dms    = true
```

---

## 5. The Agent

### 5.1 Chat mode

```sh
reasonix chat
```

Interactive Bubble Tea TUI. The agent has the full tool set and responds
turn-by-turn. Key controls:

| Key | Action |
|-----|--------|
| Enter | Send message |
| Ctrl+C | Cancel the current turn |
| Ctrl+D / `/quit` | Exit chat |
| Ctrl+Z | Suspend to background |
| Esc-Esc | Open rewind/checkpoint picker |
| `!<cmd>` | Run a shell command directly (bypasses the agent) |

### 5.2 Run mode (one-shot)

```sh
reasonix run "implement the TODOs in main.go"
reasonix run --model mimo-pro "add unit tests for this function"
echo "explain this code" | reasonix run
```

Non-interactive. The agent runs until it finishes, hits `max_steps`, or is
cancelled. Permissions still apply: `deny` rules are enforced, but `ask` rules
resolve to **allow** (no TTY to prompt). Plan mode is skipped.

### 5.3 Plan mode

Plan mode is a **coarse gate** that prevents all writer tools (file edits,
bash) from executing. The agent can only read, search, and think — producing a
plan for you to review.

- **Manual**: Send a message starting with "plan:" or use `--plan` flag
- **Auto**: Set `auto_plan = "on"` in config for complex tasks
- **Exit**: The agent calls `exit_plan_mode` when ready; you approve the plan,
  then it executes with an auto-approve window for the planned writes

In chat, send `plan: refactor the auth module` to start. Use `/auto-plan on|off`
to toggle automatic plan mode. Outside chat: `reasonix config auto-plan on`.

### 5.4 Goal mode

`/goal <objective>` starts an **autonomous, multi-turn goal** — the agent loops
until it reports completion, gets blocked, or is stopped:

```
/goal implement user registration with email verification
/goal status     # check progress
/goal clear      # cancel the goal
```

The controller prepends goal context outside the cache-stable prompt, issues
continuation turns automatically, and stops when:
- The model reports the goal complete
- The same blocked state repeats 3 times (normalized audit)
- You type `/stop` or `/goal clear`
- The safety continuation limit is reached

Goal mode is available in chat, desktop, and Discord (via `/goal` slash command).

### 5.5 Two-model collaboration

Add a **planner** — a separate model that researches before the executor acts:

```toml
[agent]
planner_model      = "deepseek-pro"
planner_max_steps  = 12
```

The planner sees the same memory context but only **read-only** research tools
(read_file, grep, glob, ls, codegraph tools). It produces a concise plan,
then the executor (full tool set) carries it out. Sessions are separate so
neither model disturbs the other's prefix cache.

### 5.6 Auto-plan

```toml
[agent]
auto_plan            = "on"
auto_plan_classifier = "deepseek-flash"   # optional; only called for borderline inputs
```

When enabled, complex-looking tasks automatically enter plan mode — the agent
drafts a read-only plan, waits for approval, then executes. The classifier
uses a cheap provider so borderline tasks don't waste expensive model calls.

Control from chat: `/auto-plan on|off`. From shell: `reasonix config auto-plan on`.

### 5.7 Subagent skills

Skills tagged `[🧬 subagent]` spawn an isolated sub-agent loop — its tool calls
and reasoning never enter your context, only its final answer does. Use for
context-heavy work (deep exploration, multi-step research).

```toml
[agent]
subagent_model  = "deepseek-pro"                    # default for all subagents
subagent_models = { review = "deepseek-pro" }       # per-skill overrides
```

Adjust effort with `/effort low|medium|high|max` — controls the reasoning
(token spend on thinking) for the current model.

**Concurrent sub-agents:** Run multiple independent sub-agents in parallel by
spawning them from MCP tools or by using `run_skill` in batch mode. Each
sub-agent gets its own isolated session and runs independently — the parent
collects results when all complete. This is useful for parallel codebase
exploration, multi-file reviews, or running independent tasks simultaneously.

---

## 6. Providers & Models

### 6.1 Built-in presets

Reasonix ships with no hardcoded models. These are pre-tested config presets
you can enable:

| Provider | Model | Base URL |
|----------|-------|----------|
| `deepseek-flash` | deepseek-v4-flash | `https://api.deepseek.com` |
| `deepseek-pro` | deepseek-v4-pro | `https://api.deepseek.com` |
| `mimo-pro` | mimo-v2.5-pro | `https://token-plan-cn.xiaomimimo.com/v1` |
| `mimo-flash` | mimo-v2.5 | `https://token-plan-cn.xiaomimimo.com/v1` |
| `anthropic-*` | claude-sonnet-4-* etc. | `https://api.anthropic.com` |

### 6.2 Adding providers

Any OpenAI-compatible endpoint is a config entry — no code changes needed:

```toml
[[providers]]
name        = "my-custom-model"
kind        = "openai"
base_url    = "https://my-api.example.com/v1"
model       = "my-model-name"
api_key_env = "MY_API_KEY"
```

The `kind` field selects the wire protocol: `openai` for
OpenAI-compatible `/chat/completions`, or `anthropic` for the Anthropic Messages
API.

### 6.2a Community provider presets

These OpenAI-compatible endpoints are tested and known to work as config entries.
Copy the relevant block into your `reasonix.toml`:

| Provider | Base URL | Model examples |
|----------|----------|---------------|
| **OpenRouter** | `https://openrouter.ai/api/v1` | `deepseek/deepseek-v4-pro`, `anthropic/claude-sonnet-4` |
| **Ollama (local)** | `http://localhost:11434/v1` | `llama3`, `codellama`, `deepseek-r1` |
| **NVIDIA NIM** | `https://integrate.api.nvidia.com/v1` | `deepseek-ai/deepseek-v4-pro` |
| **Fireworks** | `https://api.fireworks.ai/inference/v1` | `accounts/fireworks/models/deepseek-v4-pro` |
| **SiliconFlow** | `https://api.siliconflow.cn/v1` | `deepseek-ai/DeepSeek-V4-Pro` |
| **Together** | `https://api.together.xyz/v1` | `deepseek-ai/DeepSeek-V4` |
| **Groq** | `https://api.groq.com/openai/v1` | `llama-3.3-70b-versatile` |
| **vLLM (self-hosted)** | `http://localhost:8000/v1` | `deepseek-ai/DeepSeek-V4-Pro` |
| **SGLang (self-hosted)** | `http://localhost:30000/v1` | `deepseek-ai/DeepSeek-V4-Pro` |
| **Xiaomi MiMo** | `https://token-plan-cn.xiaomimimo.com/v1` | `mimo-v2.5-pro`, `mimo-v2.5` |
| **MiniMax** | `https://api.minimaxi.com/v1` | `minimax-m3` (binary thinking knob: adaptive/disabled) |
| **GLM (ZhipuAI)** | `https://open.bigmodel.cn/api/paas/v4` | `glm-4.5`, `glm-4-plus` (standard reasoning_effort) |

```toml
# Example: OpenRouter
[[providers]]
name        = "openrouter-deepseek"
kind        = "openai"
base_url    = "https://openrouter.ai/api/v1"
model       = "deepseek/deepseek-v4-pro"
api_key_env = "OPENROUTER_API_KEY"

# Example: local Ollama
[[providers]]
name        = "ollama-llama3"
kind        = "openai"
base_url    = "http://localhost:11434/v1"
model       = "llama3"
api_key_env = "OLLAMA_API_KEY"  # can be "ollama" if auth disabled
```

### 6.3 Model references

A **model reference** can be:
- A provider name → uses that provider's default model
- A bare model name → resolves across all providers
- `provider/model` → explicit

Examples: `deepseek-flash`, `mimo-pro`, `deepseek-pro/deepseek-v4-pro`.

### 6.4 Model switching

In chat: `/model` lists available models. `/model deepseek-pro` switches.
The change applies to the next turn.

In the Discord bot: `/model flash`, `/model pro`, `/model mimo`. Use `/new`
after switching to apply the change. As of v1.5.1, `/model` is a **real Discord
Application Command** (with autocomplete choices) — no longer just text-prefix
parsing. Model preferences persist across bot restarts in
`~/.config/reasonix/bot-model-prefs.json`. Set `webhook_url_env` in
`[bot.discord]` to receive lifecycle notifications (startup, errors) via
Discord webhook.

---

## 7. Built-in Tools

All 15 built-in tools self-register at compile time. Enable/disable subsets
via `[tools].enabled` in config.

### 7.1 File tools

| Tool | Description |
|------|-------------|
| `read_file` | Read a file with optional line offset/limit. Binary-safe with encoding detection. |
| `write_file` | Write content to a file (overwrites). Creates parent directories. |
| `edit_file` | Replace an exact string in a file. Old string must be unique. |
| `multi_edit` | Apply multiple edits to one file atomically. All-or-nothing. |
| `delete_range` | Delete a contiguous range using start/end text anchors. |
| `delete_symbol` | Delete a named Go symbol using AST parsing (function, type, method, etc.). |
| `notebook_edit` | Edit a Jupyter notebook cell (replace, insert, delete). |

### 7.2 Search & navigation

| Tool | Description |
|------|-------------|
| `grep` | Search with ripgrep. Honors `.gitignore`. Returns path:line:text. |
| `glob` | Find files by pattern. Supports `**` recursive matching. |
| `ls` | List directory entries with sizes. Recursive mode available. |
| `web_fetch` | Fetch a URL and return readable text (HTML → text, JSON verbatim). |

### 7.3 Shell & execution

| Tool | Description |
|------|-------------|
| `bash` | Execute a shell command. Timeout via `[tools].bash_timeout_seconds`. Sandboxed on macOS (Seatbelt), Linux (bubblewrap), and Windows (AppContainer). |

### 7.4 Meta tools

| Tool | Description |
|------|-------------|
| `completestep` | Record evidence-backed completion of a plan step. |
| `todo_write` | Manage a structured task list with phases and sub-steps. |
| `bgjobs` | Manage background jobs (bash/task started with `run_in_background`). |

### 7.5 Tool registry

Each run assembles a `*tool.Registry` from enabled built-ins plus plugin tools.
Plugin tools are namespaced `mcp__<server>__<tool>`. The Model Context Protocol's
`readOnlyHint` maps to `ReadOnly()` — plugins opt into parallel-batch dispatch
and permission reader-default by declaring it.

---

## 8. Permissions & Sandbox

### 8.1 Permission policy

Permission gating decides, **per tool call**, whether to allow, deny, or ask.
Precedence: `deny` > `ask` > `allow` > fallback.

```toml
[permissions]
mode  = "ask"
deny  = ["Bash(rm -rf*)", "Bash(git push --force*)"]
allow = ["Bash(go test:*)", "Edit(docs/**)"]
```

- `deny` always wins — even in yolo mode, even with an approved plan
- `ask` overrides a broad `allow` to force a prompt on a risky subset
- Fallback: `Allow` for read-only tools, `mode` for writers (default `Ask`)

### 8.2 Approval postures

Three interactive postures control how `Ask` decisions are resolved:

| Posture | Writer fallback | `deny` rules | Plan approval |
|---------|----------------|-------------|---------------|
| **Need approval** (`ask`) | Prompts user | Enforced | Waits for user |
| **Auto approve** (`auto`) | Writer fallback auto-allowed; explicit ask/deny rules still apply | Enforced | Waits for user |
| **YOLO** (`yolo`) | Auto-allowed | Enforced | Waits for user |

In headless mode (`reasonix run`, sub-agents, no TTY), `Ask` resolves to
**allow** — preserving autonomous behavior.

### 8.3 Rule syntax

Rules use Claude Code-style families:

```
Tool                    — matches any call to that tool
Tool(subject)           — matches when the call's subject matches
Tool(subject:*)         — bash command-prefix (rejects later shell operators)
Tool(glob*)             — glob-pattern match
```

Examples:
- `Bash(npm run build)` — exact command
- `Bash(npm run test:*)` — command prefix (covers `npm run test:unit`, etc.)
- `Edit(docs/**)` — any file under docs/
- `Bash` — any bash command (use with caution)

Approvals are stored as rules, not button labels. A session grant for
`Bash(go test:*)` means similar invocations won't prompt again.

### 8.4 Sandbox

The sandbox is **enforcement**, separate from permissions:

- **File writers** (`write_file`, `edit_file`, `multi_edit`) refuse paths
  outside `[sandbox].workspace_root` (default: current directory). Symlinks and
  `..` are resolved so a link can't tunnel out.
- **Reads** are unrestricted.
- **Bash on macOS** is jailed by default (Seatbelt): commands may write only
  the workspace root plus temp and toolchain caches. Network access requires
  `[sandbox.bash] network = true`.
- **Bash on Linux** uses bubblewrap (`bwrap`) with the same profile: read-only
  root filesystem, writable workspace + temp + caches, network isolated unless
  `[sandbox.bash] network = true`. Install with `apt install bubblewrap`.
- **Bash on Windows** runs inside an AppContainer (LowBox isolation, available since
  Windows 8): commands are confined via `CreateProcess` with `SECURITY_CAPABILITIES`
  — reads freely, writes only to workspace + temp + toolchain caches, network
  controlled via `internetClient` capability SIDs. No external dependencies needed.

```toml
[sandbox]
workspace_root = "/home/user/projects/myapp"
allow_write    = ["/tmp", "/var/log"]

[sandbox.bash]
network = true    # allow bash network access on macOS
```

---

## 9. Plugins / MCP

### 9.1 MCP client

Reasonix is an **MCP client**. Declare external tools as `[[plugins]]` entries
— they connect at session start and their tools appear to the model as
`mcp__<server>__<tool>`.

### 9.2 Transport types

| Type | Description |
|------|-------------|
| `stdio` (default) | Local subprocess over stdin/stdout JSON-RPC. Declare with `command`/`args`/`env`. |
| `http` | Remote server over Streamable HTTP. Declare with `url` and optional `headers`. |

`${VAR}` and `${VAR:-default}` are expanded in `command`, `args`, `env`, `url`,
and `headers` — secrets stay in the environment.

```toml
[[plugins]]
name    = "example"
command = "reasonix-plugin-example"

[[plugins]]
name    = "stripe"
type    = "http"
url     = "https://mcp.stripe.com"
headers = { Authorization = "Bearer ${STRIPE_KEY}" }
```

### 9.3 .mcp.json support

Drop a `.mcp.json` in the project root and Reasonix reads it as-is. The
`mcpServers` spec maps field-for-field onto `[[plugins]]`. On a name collision,
`reasonix.toml` wins.

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"]
    }
  }
}
```

Legacy `~/.reasonix/config.json` `mcpServers` are still read as lowest priority.

### 9.4 MCP prompts & resources

- **Prompts** surface as `/mcp__<server>__<prompt>` slash commands
- **Resources** are pulled in with `@<server>:<uri>` in a message
- Both resolve to text and reuse the same "start a turn" path as a typed message

### 9.5 Managing MCP servers

| Action | How |
|--------|-----|
| List connected servers | `/mcp` |
| Inspect a server | `/mcp <name>` |
| Reconnect a server | `/mcp <name> reconnect` |
| Disable for session | `/mcp <name> disable` |
| See MCP diagnostics | `/mcp diag` |

Enabled servers connect in the background after session start — chat stays
usable while tools come online.

### 9.6 Tool overrides via MCP

You can override or replace built-in tools with custom MCP plugins without
modifying the codebase. This is useful for:

- **Audit logging** — wrap `bash` with a script that logs every command
- **Alternative implementations** — replace `read_file` with a cached or
  remote-backed version
- **Policy enforcement** — add pre-execution checks to dangerous tools

```toml
# Replace bash with an audit-logging wrapper
[[plugins]]
name       = "bash-audit"
command    = "./bin/reasonix-mcpbridge"
# The wrapper receives the same JSON-RPC tool call and can
# delegate to the real reasonix after logging.
```

For finer-grained control, use MCP server tools that shadow built-in names.
The namespaced `mcp__<server>__<tool>` convention prevents clashes, but if
you name your tool the same as a built-in, the last registered tool wins.

---

## 10. Slash Commands

### 10.1 Built-in commands

All run locally and never reach the model:

| Command | Action |
|---------|--------|
| `/help` | List all commands |
| `/compact` | Force session compaction (summarize history) |
| `/new` | Start a new session (saves previous for history) |
| `/clear` | Discard current context (confirmation required) |
| `/rewind` | Open rewind/checkpoint picker |
| `/tree` | Show saved conversation branches |
| `/branch [name]` | Fork current conversation tip |
| `/branch <turn> [name]` | Fork from an earlier checkpointed turn |
| `/switch <id\|name>` | Load another branch |
| `/todo` | Show current task list |
| `/model [name]` | List or switch models |
| `/effort low\|medium\|high\|max` | Set reasoning effort |
| `/diff-fold` | Toggle diff line folding in tool output |
| `/mcp [name]` | List or manage MCP servers |
| `/skills` | List available skills |
| `/hooks` | Show hook status |
| `/memory` | View/manage memory files |
| `/output-style concise\|verbose` | Set output verbosity |
| `/sandbox` | Show sandbox status |
| `/language zh\|en` | Switch UI language |
| `/auto-plan on\|off` | Toggle automatic plan mode |
| `/goal <objective>` | Start autonomous goal |
| `/goal status` | Show goal progress |
| `/goal clear` | Cancel active goal |
| `/init` | Generate AGENTS.md from codebase analysis |
| `/learn` | Show detected patterns and suggest skill drafts |
| `/install-source` | CLI: install skills/MCP from URL, path, or package (`reasonix install-source install --source <url>`) |

### 10.2 Custom commands

Create Markdown files under `.reasonix/commands/` (project) or
`~/.config/reasonix/commands/` (user). The project directory overrides the user
directory on name clashes.

```
.reasonix/commands/
├── review.md          → /review
└── git/
    └── commit.md      → /git:commit
```

```markdown
---
description: Review the staged diff
argument-hint: [focus-area]
---
Review the staged diff. Focus on $ARGUMENTS, list bugs with file:line.
```

Template variables: `$ARGUMENTS` (all args), `$1`…`$N` (positional), `$$`
(literal `$`). MCP prompts appear alongside custom commands with the
`/mcp__<server>__<prompt>` prefix.

---

## 11. @ References

Embed `@` references in a message — Reasonix resolves them before sending as
tagged context blocks:

| Reference | Resolves to |
|-----------|------------|
| `@path/to/file` | File contents |
| `@path/to/dir` | Directory listing |
| `@file#L10-L20` | Lines 10–20 of a file |
| `@<server>:<uri>` | MCP resource |

A local path is only treated as a reference when it actually exists — ordinary
`@mentions` stay literal. Typing `/` or `@` opens an autocomplete menu with
slash commands and hierarchical file navigation (one directory level at a time).

---

## 12. Checkpoints & Rewind

Checkpoints are **file snapshots** independent of git — no commits, no staging,
no `.git/` pollution. Works in non-git directories.

**How it works:**

- One checkpoint opens per user turn, labelled with your prompt
- Before each edit-tool call (`write_file`, `edit_file`, `multi_edit`), the
  file's pre-edit content is snapshotted
- `bash` side effects are NOT tracked (can't know what a shell command touched)

**Using rewind:**

- **Esc-Esc** (chat) or `/rewind` — opens the checkpoint picker
- Restore **code**, **conversation**, or **both**
- **Fork from here** — branch the conversation from a past checkpoint
- **Summarize from/up to here** — generate a summary between any two checkpoints

The desktop app also offers hover-rewind on checkpoint cards.

---

## 13. Memory System

### 13.1 Hierarchical doc memory

At session start, Reasonix loads memory files in this order, merging them
into the system prompt:

1. `REASONIX.md` (project root) — committed, shared with team
2. `REASONIX.local.md` — personal, git-ignored
3. `~/.config/reasonix/REASONIX.md` — user-global preferences
4. Any `REASONIX.md` in an ancestor directory
5. `AGENTS.md` — fallback name, same locations

Within a memory file, `@path/to/file` on its own line imports that file's
contents. This is how `/init` bootstraps your project — it reads your codebase
and writes a tailored `AGENTS.md`.

### 13.2 Auto-memory (remember/forget)

The agent can save durable facts across sessions using the `remember` tool:

- **User facts** — who you are, preferences
- **Project facts** — ongoing goals, constraints, conventions not derivable from code
- **Feedback** — guidance with "Why:" and "How to apply:" structure
- **References** — pointers to external resources

Use `forget` to delete stale or wrong memories. Stored as frontmatter files
with a `MEMORY.md` index in `~/.config/reasonix/projects/<hash>/memory/`.

### 13.3 Managing memory

| Action | How |
|--------|-----|
| View loaded memory | `/memory` |
| Quick-add a note | `#<note>` in chat → appends to REASONIX.md |
| Save a durable fact | Tell the agent "remember that …" |
| Delete a memory | Tell the agent "forget about …" |
| Bootstrap project memory | `/init` |

### 13.4 Dense embeddings (optional)

When the Hindsight memory server is configured with embedding support, facts
stored via `hindsight_retain` are automatically embedded using the configured
provider's embeddings API (OpenAI-compatible `/v1/embeddings`). This enables
semantic search via `hindsight_recall` with `dense=true`:

```sh
# Start memory server with embedding:
EMBEDDING_PROVIDER=https://api.deepseek.com \
EMBEDDING_MODEL=text-embedding-3-small \
EMBEDDING_API_KEY=$DEEPSEEK_API_KEY \
./bin/reasonix-memory --backend sqlite --http
```

In `reasonix.toml`:
```toml
[embedding]
provider   = "deepseek"            # provider name from [[providers]]
model      = "text-embedding-3-small"
api_key_env = "DEEPSEEK_API_KEY"
batch_size = 20                    # facts per API call
```

Dense vectors are stored alongside sparse TF-IDF vectors. The memory server's
`hindsight_reflect` output includes embedding coverage stats. The desktop D3
memory graph shows which facts have dense embeddings via the `hasDenseEmbedding`
toggle.

---

| Action | How |
|--------|-----|
| View loaded memory | `/memory` |
| Quick-add a note | `#<note>` in chat → appends to REASONIX.md |
| Save a durable fact | Tell the agent "remember that …" |
| Delete a memory | Tell the agent "forget about …" |
| Bootstrap project memory | `/init` |

---

## 14. Skills

### 14.1 Built-in skills

Reasonix ships with a set of built-in subagent skills. Tagged `[🧬 subagent]`
in the skills index, they spawn isolated loops — you get only the final answer:

- `explore` — wide-net read-only codebase investigation
- `review` — correctness / security / missing-tests review of diffs
- `security_review` — injection / authz / secrets / crypto audit
- `research` — combine web_fetch + code reading for external reference questions

Invoke with `/<name>` in chat or use the dedicated tools (`review`, `explore`,
`security_review`, `research`).

### 14.2 Hermes skills hub

The Hermes fork adds a **17-skill community registry**. Install with:

```sh
reasonix install-source install \
  --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills
```

| Skill | Category | Description |
|-------|----------|-------------|
| `adversarial-review` | Security | BLOCK:/ALLOW: contract, 5 attack surfaces |
| `api-design` | Architecture | REST API design patterns, pagination, versioning |
| `ci-cd-helper` | DevOps | CI/CD pipeline debugging and optimization |
| `code-review` | Quality | Systematic code review with file:line citations |
| `council` | Collaboration | Multi-perspective review from different roles |
| `database-helper` | Database | Schema design, query optimization, migrations |
| `debugger` | Debugging | Reproduce → isolate → fix → verify workflow |
| `deep-research` | Research | Multi-source investigation with citations |
| `documentation` | Documentation | README, API docs, inline comments |
| `explore` | Exploration | Codebase survey and architecture analysis |
| `frontend-builder` | Frontend | React, component design, state management |
| `git-commit` | Git | Conventional commits, PR descriptions |
| `migration-assistant` | Migration | Framework/library migration planning |
| `performance-profiler` | Performance | Bottleneck identification, profiling |
| `refactoring` | Refactoring | Safe refactoring with verification |
| `security-audit` | Security | Full security audit with severity ratings |
| `test-generator` | Testing | Test generation from code analysis |

### 14.3 Skill lifecycle

```sh
# List all installed skills
reasonix skills list

# Disable a skill
reasonix skills disable review

# Re-enable
reasonix skills enable review

# Install from a URL or local path
reasonix install-source install --source ./my-custom-skill.md

# Uninstall by name
reasonix install-source uninstall --name my-skill
```

---

## 15. Hooks

### 15.1 Upstream hook runner

Reasonix supports hook scripts at three lifecycle points through
`internal/hook`. Configured in `reasonix.toml`:

```toml
[hooks]
pre_tool_use  = ["./scripts/pre-tool.sh"]   # before each tool call
post_tool_use = ["./scripts/post-tool.sh"]  # after each tool call
stop          = ["./scripts/on-stop.sh"]     # when a session stops
```

The hook runner receives a JSON payload on stdin with `event`, `tool_name`,
`tool_input`, `tool_result`, `session_id`, `last_assistant`, and `turn`.

### 15.2 Hermes native hook runner

Hermes provides a **zero-dependency Go binary** (`cmd/reasonix-hooks`) that
replaces shell scripts with a compiled executable:

```sh
go build -o bin/reasonix-hooks ./cmd/reasonix-hooks
```

Two actions:

| Action | When | What it does |
|--------|------|-------------|
| `retain` | PreToolUse | POSTs tool call context to the Hindsight memory server |
| `reflect` | Stop | Triggers a session reflection on the memory server |

```toml
[hooks]
pre_tool_use  = ["./bin/reasonix-hooks", "retain"]
stop          = ["./bin/reasonix-hooks", "reflect"]
```

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HINDSIGHT_URL` | `http://localhost:8080` | Memory server URL |
| `HINDSIGHT_KEY` | (none) | Bearer token for auth |
| `HINDSIGHT_TIMEOUT` | `5` | HTTP timeout in seconds |

Noise tools (`read_file`, `write_file`, `edit_file`, `bash`, `search`, `glob`)
are filtered — their invocations are not sent to the memory server. Exit code
is always 0 (never blocks the agent).

---

## 16. Hermes Extensions

### 16.1 Discord Bot

A full Discord bot gateway powered by the upstream `BotGateway` and
`control.Controller` — every Discord session gets a real agent loop.

**Quick start:**

```sh
export DISCORD_BOT_TOKEN="your-token-here"
export DISCORD_SERVER_ID="optional-server-id"
./bin/reasonix-bot
```

Or with explicit flags:

```sh
./bin/reasonix-bot \
  --token "your-token" \
  --server "guild-id" \
  --channel "channel-id" \
  --model deepseek-pro \
  --allow-all
```

**Slash commands:**

| Command | Action |
|---------|--------|
| `/stop` | Stop the current turn |
| `/new` | Start a fresh session |
| `/reset` | Reset the current session |
| `/model [flash\|pro\|mimo]` | Switch model (takes effect after `/new`) |
| `/model` | Show current model |
| `/goal <objective>` | Start an autonomous multi-turn goal |
| `/goal status` | Check goal progress |
| `/goal clear` | Cancel the active goal |
| `/approve <id>` | Approve a pending tool call |
| `/deny <id>` | Deny a pending tool call |
| `/answer <id> <option>` | Answer an ask question |
| `/status` | Show active tasks and session count |
| `/help` | List all commands |

**Architecture:** `Discord (discordgo)` → `discord.Adapter` → `BotGateway` →
`control.Controller` per session. The gateway handles concurrency, debounce
(1500ms default), approval flows, and session lifecycle. Sessions are evicted
after 30 minutes of idle time (configurable via `session_idle_timeout`).

**Configuration:**

```toml
[bot]
enabled              = true
model                = "deepseek-flash"
max_steps            = 30
debounce_ms          = 1500
session_idle_timeout = "30m"

[bot.allowlist]
enabled        = true
discord_users  = ["123456789"]
discord_groups = ["987654321"]

[bot.discord]
token_env    = "DISCORD_BOT_TOKEN"
server_id    = "123456789"
channel_id   = ""         # empty = all channels
allow_dms    = true
```

### 16.2 MCP Bridge Server

Exposes Reasonix as an MCP tool server — connect Claude Code, Codex, or any
MCP client to delegate work to Reasonix/DeepSeek.

**Start:**

```sh
# Stdio mode (default)
./bin/reasonix-bridge

# HTTP mode
./bin/reasonix-bridge --http --port 9090
```

**6 MCP tools exposed:**

| Tool | Description |
|------|-------------|
| `reasonix_run` | Execute a one-shot coding task via Reasonix + DeepSeek |
| `reasonix_doctor` | Run a system diagnostic check |
| `reasonix_plan` | Generate a read-only plan for a task |
| `reasonix_orchestrate` | Orchestrate multi-step workflows |
| `reasonix_get_skill` | Read a skill body by name |
| `reasonix_get_skills` | List all available skills |

**Configuration:**

The bridge reads `DEEPSEEK_API_KEY` and `DEEPSEEK_BASE_URL` from the
environment. It shells out to the `reasonix` CLI for task execution.

**Connect from Claude Code:**

```json
{
  "mcpServers": {
    "reasonix": {
      "command": "/path/to/bin/reasonix-bridge"
    }
  }
}
```

### 16.3 Hindsight Memory Server

Cross-session persistent memory with TTL, importance scoring, and vector search.

**Start:**

```sh
# File backend (default)
./bin/reasonix-memory

# SQLite backend (recommended for production)
./bin/reasonix-memory --backend sqlite

# HTTP mode
./bin/reasonix-memory --backend sqlite --http --port 8080
```

**3 MCP tools:**

| Tool | Description |
|------|-------------|
| `hindsight_retain` | Store a memory with optional tags. Content, session_id, and tags. Returns memory ID. |
| `hindsight_recall` | Search memories by query, session, tags, or semantic similarity. Returns scored results. |
| `hindsight_reflect` | Trigger a session reflection — summarize recent tool calls into memories. |

**Memory lifecycle:**

- **TTL:** 90 days default. Expired memories are excluded from recall and purged by `Tidy()`.
- **Importance:** Starts at 0.5. +0.05 on each recall. -1% per day decay. Expired + low importance = purged.
- **Vector search:** TF-IDF sparse vectors computed on retain. `semantic=true` flag on recall for cosine similarity ranking.

**Configuration options:**

| Flag | Description |
|------|-------------|
| `--backend file\|sqlite` | Storage backend (default: file) |
| `--http` | Run as HTTP server |
| `--port N` | HTTP port (default: 8080) |

**Connect as an MCP plugin:**

```toml
[[plugins]]
name    = "hindsight"
command = "./bin/reasonix-memory"
args    = ["--backend", "sqlite"]
```

**Environment variables:**

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMORY_API_KEY` | (none) | If set, requires Bearer token on HTTP endpoints |
| `REASONIX_PORTABLE` | (none) | Store data in `<binary_dir>/.reasonix/hindsight-memory/` |

### 16.4 Portable Mode

Run Reasonix entirely from a portable directory — all data stays with the binary:

```sh
REASONIX_PORTABLE=1 reasonix chat
```

When `REASONIX_PORTABLE` is set, all data paths redirect to
`<binary_dir>/.reasonix/`:

| Data | Normal path | Portable path |
|------|------------|---------------|
| Config | `~/.config/reasonix/` | `<exe_dir>/.reasonix/` |
| Sessions | `~/.config/reasonix/sessions/` | `<exe_dir>/.reasonix/sessions/` |
| Memory | `~/.config/reasonix/projects/` | `<exe_dir>/.reasonix/projects/` |
| Hindsight | `~/.reasonix/hindsight-memory/` | `<exe_dir>/.reasonix/hindsight-memory/` |
| Skills | `~/.reasonix/skills/` | `<exe_dir>/.reasonix/skills/` |
| Commands | `~/.config/reasonix/commands/` | `<exe_dir>/.reasonix/commands/` |

The MCP bridge and memory server also respect `REASONIX_PORTABLE`.

### 16.5 Skills Hub

A curated registry of 17 community skills with frontmatter playbooks. Each
skill declares:

```markdown
---
name: debugger
description: Systematic debugging workflow — reproduce, isolate, fix, verify.
runAs: inline
allowedTools:
  - read_file
  - grep
  - glob
  - bash
  - edit_file
---
```

**`runAs` modes:**
- `inline` (default) — the skill body folds into your turn; you read and act on it
- `subagent` — spawns an isolated child loop; only the final answer returns

**Categories:** architecture, collaboration, database, debugging, devops,
documentation, exploration, frontend, git, migration, performance, quality,
refactoring, research, security, testing.

The manifest (`reasonix-hermes.json`) enables installation via upstream's
`install_source` tool. The skills hub website deploys to GitHub Pages via
`.github/workflows/pages.yml`.

### 16.5 Token-Saving Compression (v1.7.0+)

Reasonix Hermes includes a built-in **tool output compressor** that reduces token
consumption by 25-92% on tool results before they enter the model's context. It
combines three techniques from the 2026 token-saving landscape:

**SHA-256 content cache (sqz-style):** Repeated tool output (e.g. reading the same
file twice) is replaced with a compact reference like
`[content unchanged since turn 3 (sha256:abc123…)]` instead of re-injecting the
full content.

**Repeated-line collapsing:** Bash output with dozens of identical lines (build
logs, test runs) is collapsed to one instance + a `[×49 above]` marker.

**JSON minification:** JSON tool results have null fields and empty lines stripped,
reducing token overhead on API responses.

**Safe mode:** Output containing ≥2 error markers (panic, stack trace, FAIL, diff,
goroutine) is preserved verbatim — compression of debugging output would destroy
the model's ability to diagnose and fix problems.

```toml
# Enabled by default. Disable if you prefer verbatim tool output.
[agent]
compress_tool_output = false
```

**External token-saving MCP plugins** (optional):

```toml
# sqz — Rust CLI, 25-92% reduction, SHA-256 cache + dedup
[[plugins]]
name    = "sqz"
command = "sqz"
args    = ["mcp"]

# context-mode — TypeScript sandbox, up to 98% reduction (Microsoft/Google/Meta)
[[plugins]]
name    = "context-mode"
command = "context-mode"
args    = ["mcp"]
```

### 16.6 Scheduled Tasks (v1.7.0+)

Automated cron-driven agent tasks. The scheduler runs in the background and
fires prompts at configured times.

```toml
[schedule]
[[schedule.tasks]]
name    = "daily-review"
cron    = "0 2 * * *"           # every day at 2am
prompt  = "Review all changes in the last 24 hours. Report any issues."
model   = "deepseek-flash"
enabled = true
```

Results are logged and visible on the desktop Hermes dashboard.

### 16.7 Session Publishing (v1.7.0+)

Export sessions as self-contained HTML with syntax-highlighted code blocks,
light/dark mode, and inline CSS. No external dependencies.

- **CLI:** `/publish` slash command in TUI — writes to `sessions/published/`
- **API:** `internal/publish/` package — `ToHTML()` and `ToJSON()` for programmatic use

### 16.8 Hash-Anchored Edits (v1.7.0+)

The `edit_file` tool accepts an optional `content_hash` (SHA-256 of file content
from a prior `read_file`). If the file changed between the read and the edit
(e.g. another process modified it), the edit is rejected:

```
file main.go changed since content_hash was computed — re-read the file and try again
```

This prevents stale-context edits from corrupting files.

### 16.9 Provider Cost Tracking (v1.7.0+)

Per-session cost accumulation via `provider.Pricing` (per 1M token rates). Cost
is displayed in:
- Desktop Hermes dashboard (live via `HermesDashboardEvent`)
- CLI `/stats` panel (`/cost` alias)
- CLI status line (session cost badge)

### 16.10 CLI: `reasonix models`

Lists all configured providers with model, kind, pricing, and connectivity status:

```bash
reasonix models           # list providers
reasonix models refresh   # test connectivity
```

---

## 17. Desktop App

Reasonix ships a full **Wails v2 desktop application** with a React 19 +
TypeScript 6 frontend:

```sh
cd desktop/frontend && npm install && cd ../..
cd desktop && wails dev
```

**Key desktop features (upstream v1.6.0):**
- Themeable workspace (4 variants: dark, light, high-contrast, sepia)
- File tree with drag-and-drop workspace organization
- Checkpoint cards with hover-rewind
- Model switcher and effort control
- Bot gateway panel (Feishu/WeChat/QQ/Discord)
- Plugin/MCP management panel
- Skills panel
- Memory panel (view/edit REASONIX.md, AGENTS.md)
- Settings panel (full config management)
- Multi-tab sessions with topic-based labels
- PDF attachment extraction
- Ctrl+Home/Ctrl+End scroll, Ctrl+Z suspend

**Hermes additions (v1.5.1):**
- **Hotbar**: Keys `1`–`7` mapped to common actions (palette, workspace, new,
  history, dock, sidebar, settings). Configurable via `[desktop.hotbar]` in
  `reasonix.toml` — set any key to `""` to unbind it.
- **Desktop toolchain**: `biome` (TS formatting/linting), `taplo` (TOML
  validation) recommended for development. `wails dev` for hot-reload.

**Hermes enrichment (v1.6.1, 2026-07-12):**
- **Hermes accent theme** — 7th theme style "hermes" with caduceus gold accent
  (#d4a853 dark / #b8912e light), warm dark surfaces, and teal highlights.
  Selectable in Settings → Appearance alongside Graphite/Aurora/Slate/Carbon/
  Nocturne/Amber.
- **Write Mode** — split-pane Markdown workspace with file browser sidebar,
  textarea editor, and live rendered HTML preview. Cmd+S to save, refresh to
  reload, plus button to create new `.md` files. Accessible from the workspace
  panel.
- **Live data push** — 3-second Wails event loop (`hermes:dashboard`) replaces
  frontend `setInterval` polling. All dashboard components (cache, memory, bot,
  goal, subagents, constitution, token chart, compaction log, memory facts)
  receive push updates from Go.
- **Token sparkline chart** — per-turn stacked bar chart showing prompt +
  completion tokens with cache hit-rate and peak token display. Accumulated
  in a 64-turn ring buffer on the Agent.
- **Compaction timeline** — shows each compaction pass (auto/manual) with
  trigger, message count, and truncated summary. 32-event ring buffer.
- **Checkpoint file preview** — expandable per-turn file list showing pre-edit
  content (up to 500 chars) from the checkpoint file snapshots.
- **Memory fact graph** — facts clustered by type (user/project/feedback/
  reference) with color-coded badges.
- **StatusBar updates** — Discord monitor and cache gauge now use push events
  instead of independent `setInterval` polling.
- All enrichment components live under `desktop/frontend/src/components/hermes/`
  and are accessible in the Hermes tab of Settings.

**CLI TUI enhancement (v1.6.1):**
- **Pinned banner** — compact ╔╗-bordered ⚚ REASONIX-HERMES header stays
  visible above the transcript; never scrolls away.
- **Bottom status counters** — turns, messages, goal progress, and memory
  facts moved to the always-visible bottom data row.
- **`/stats` sparkline** — Unicode block character (▁▂▃▄▅▆▇█) token bar chart,
  one bar per turn, scaled to terminal width.
- **`/stats` compaction log** — each compaction pass with trigger, message
  count, and summary.
- **`/stats` memory facts** — fact name and title list.
- **`/stats` goal progress** — status, turns, and blocks when `/goal` is active.
- **Fixed banner alignment** — padding math clamped to non-negative, no more
  crashes on narrow terminals.
- **Fixed workspace slug** — memory store paths now use `$HOME`-relativized
  slugs with spaces replaced by dashes (e.g.
  `$HOME-Library-Application-Support-reasonix-global-workspace`).

The desktop app drives the same `control.Controller` as the CLI — they share
all agent behavior, permissions, and plugin support.

---

## 18. Bot Gateway (Multi-Platform)

The upstream v1.6.0 introduced a multi-platform bot gateway supporting:

| Platform | Adapter | Status |
|----------|---------|--------|
| Discord | `internal/bot/discord/` | Hermes addition ✅ |
| Feishu (飞书) | `internal/bot/feishu/` | Upstream |
| LINE | `internal/bot/line/` | Hermes addition ✅ |
| QQ | `internal/bot/qq/` | Upstream |
| Slack | `internal/bot/slack/` | Hermes addition ✅ |
| Telegram | `internal/bot/telegram/` | Hermes addition ✅ |
| WeChat (微信) | `internal/bot/weixin/` | Upstream |

All adapters share the same `BotGateway` → `control.Controller` architecture.
The gateway provides:

- Session management with concurrency control (one turn per session)
- Debounce merging (consecutive messages combined within a configurable window)
- Slash command dispatch (`/stop`, `/new`, `/approve`, `/deny`, `/answer`, `/status`, `/goal`, `/model`)
- Discord `/model` is a **native Application Command** with autocomplete (flash/pro/mimo)
- Per-session model preferences persist to `~/.config/reasonix/bot-model-prefs.json`
- Optional Discord webhook notifications via `webhook_url_env` in `[bot.discord]`
- Interactive approval flow (buttons → approve/deny via callback)
- Platform-agnostic allowlist (per-user and per-group)
- Markdown → platform-specific message rendering

Start the multi-platform bot from the CLI:

```sh
reasonix bot start --channels discord,feishu
```

Or run the standalone Discord bot:

```sh
./bin/reasonix-bot
```

---

## 19. Troubleshooting & FAQ

### Common issues

**"api_key_env not set"**
→ Export the environment variable named in your provider config:
```sh
export DEEPSEEK_API_KEY=sk-...
```

**"context window full" / compaction not working**
→ Set `context_window` on your provider explicitly, or lower `compact_ratio`.
→ Force compaction with `/compact` in chat.

**MCP server won't connect**
→ Check with `/mcp diag` for detailed error output.
→ Verify the command/URL works outside Reasonix.
→ For stdio servers, ensure the binary is on `$PATH` or use an absolute path.

**Discord bot not responding**
→ Verify `DISCORD_BOT_TOKEN` is set. Check the bot has "Message Content Intent" enabled in Discord Developer Portal.
→ Check the allowlist — the bot only responds to configured user/group IDs unless `--allow-all` is passed.
→ Run with `reasonix bot start --channels discord` for more diagnostic output.

**Permission prompts too frequent**
→ Add `allow` rules for common commands:
  ```toml
  allow = ["Bash(go test:*)", "Bash(npm run *)"]
  ```
→ Or switch posture: `/auto-plan off` + set `mode = "auto"` in `[permissions]`.

**Building from source fails**
→ Requires Go 1.25+. Run `go version` to check.
→ `CGO_ENABLED=0 go build ./...` if you get CGO-related errors.

### Where things live

| Item | Path |
|------|------|
| User config | `~/.config/reasonix/config.toml` |
| Project config | `./reasonix.toml` |
| Sessions | `~/.config/reasonix/sessions/` |
| Archives (compaction) | `~/.config/reasonix/archive/` |
| Memory index | `~/.config/reasonix/projects/<hash>/memory/` |
| Auto-memory store | `~/.config/reasonix/projects/<hash>/memory/` |
| Custom commands | `.reasonix/commands/` (project), `~/.config/reasonix/commands/` (user) |
| Skills | `.reasonix/skills/` (project), `~/.reasonix/skills/` (user) |
| Hindsight memory DB | `~/.reasonix/hindsight-memory/memories.db` (SQLite) |
| Portable data | `<exe_dir>/.reasonix/` (everything) |
| Desktop config | `~/Library/Application Support/reasonix/` (macOS) |

### Getting help

- **Upstream docs:** [Guide](./GUIDE.md), [Spec](./SPEC.md), [Upstream README](https://github.com/esengine/deepseek-reasonix)
- **Project memory:** [REASONIX.md](../REASONIX.md), [AGENTS.md](../AGENTS.md)
- **Ecosystem:** [Ecosystem reference](../reasonix-deepseek-ecosystem-2026.md)
- **Implementation:** [Implementation plan](./HERMES-IMPLEMENTATION-PLAN.md)
- **Upstream Discord:** [discord.gg/XF78rEME2D](https://discord.gg/XF78rEME2D)
- **Upstream GitHub:** [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix)

---

<p align="center">
  <sub>Reasonix Hermes — built on <a href="https://github.com/esengine/deepseek-reasonix">esengine/deepseek-reasonix</a> v1.7.0</sub>
  <br/>
  <sub>MIT — see <a href="../LICENSE">LICENSE</a></sub>
</p>
