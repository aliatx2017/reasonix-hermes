# Reasonix + DeepSeek Ecosystem: Complete Reference (2026)

> A comprehensive survey of integrations, plugins, tools, skills, MCP bridges, protocols, and use cases in the Reasonix ecosystem as of mid-2026.

---

## Table of Contents

1. [Overview: What is Reasonix?](#1-overview-what-is-reasonix)
2. [Core Architecture & Cost Model](#2-core-architecture--cost-model)
3. [Official Integration (DeepSeek Docs)](#3-official-integration-deepseek-docs)
4. [Desktop Application v1.5.0](#4-desktop-application-v150)
   - 4.1 [Key Desktop Features](#key-desktop-features)
   - 4.2 [Bot Gateway (Feishu / Weixin / QQ)](#bot-gateway-v150)
   - 4.3 [Desktop vs CLI](#desktop-vs-cli)
5. [MCP Bridges & Cross-Platform Integrations](#5-mcp-bridges--cross-platform-integrations)
   - 5.1 [reasonix-mcp-server (Claude Code ↔ Reasonix)](#51-reasonix-mcp-server-claude-code--reasonix)
   - 5.2 [reasonix-bridge (Codex ↔ DeepSeek, Multi-Agent)](#52-reasonix-bridge-codex--deepseek-multi-agent)
   - 5.3 [opencode-reasonix-mcp (OpenCode Bridge)](#53-opencode-reasonix-mcp-opencode-bridge)
   - 5.4 [collab-cli (Universal Multi-Agent Protocol)](#54-collab-cli-universal-multi-agent-protocol)
6. [Community Skills Hub (awesome-reasonix)](#6-community-skills-hub-awesome-reasonix)
   - 6.1 [Full Skill Registry](#61-full-skill-registry)
   - 6.2 [Skill Categories](#62-skill-categories)
7. [Long-Term Memory: Hindsight-Reasonix](#7-long-term-memory-hindsight-reasonix)
8. [Portable Toolkit: Reasonix PortaKit](#8-portable-toolkit-reasonix-portakit)
9. [Protocol Landscape: MCP / ACP / CDP](#9-protocol-landscape-mcp--acp--cdp)
10. [Use Cases & Workflows](#10-use-cases--workflows)
11. [Appendix: Quick-Reference Links](#11-appendix-quick-reference-links)

---

## 1. Overview: What is Reasonix?

**Reasonix** is an open-source (MIT), terminal-based AI coding agent built specifically for **DeepSeek models**. It is listed on DeepSeek's official API documentation as a recommended integration, giving it a degree of first-party endorsement that few third-party tools have.

| Attribute | Value |
|-----------|-------|
| **License** | MIT (fully open source) |
| **Language** | Go (1.0, `main-v2` branch — current); TypeScript (0.x, `v1` branch — legacy/maintenance) |
| **GitHub Stars** | 20,700+ |
| **Forks** | 1,200+ |
| **Releases** | 46 (v1.5.0 — Jun 10, 2026) |
| **Commits** | 777+ |
| **Default Model** | DeepSeek V4 Flash (via config TOML); also presets for DeepSeek V4 Pro, MiMo-v2.5-pro, MiMo-v2.5 |
| **Upgrade Model** | DeepSeek V4 Pro or MiMo-v2.5-pro (per-model via `/model` or `/preset max`) |
| **Platforms** | macOS (brew + prebuilt), Linux (.deb + prebuilt), Windows (prebuilt, code-signed via SignPath) |
| **Config Format** | TOML (`reasonix.toml` — flag > project > user > defaults resolution) |
| **Node Requirement** | None for v1.0 Go binary; Node ≥ 22 only for npm install shim or legacy 0.x |
| **Alias** | `dsnix` |
| **Install** | `npm i -g reasonix`, `brew install esengine/reasonix/reasonix`, prebuilt archives (6 targets), `make build`/`make cross` from source |
| **Official Docs** | [DeepSeek Agent Integrations](https://api-docs.deepseek.com/quick_start/agent_integrations/reasonix); [Guide (GUIDE.md)](https://github.com/esengine/deepseek-reasonix/blob/main-v2/docs/GUIDE.md); [Spec (SPEC.md)](https://github.com/esengine/deepseek-reasonix/blob/main-v2/docs/SPEC.md) |

### Three Operating Modes

| Mode | Command | Description |
|------|---------|-------------|
| **chat** (default) | `reasonix chat` | Interactive conversation with full tool access. Run `/init` to generate AGENTS.md project memory. Use for actual development. |
| **run** (one-shot) | `reasonix run "task"` | Executes a single task and exits. Accepts piped input and `--model` flag. Ideal for CI/CD, scripting, automation pipelines. |
| **code** | `reasonix chat` | Legacy alias preserved from 0.x; routes to `chat` in v1.0+. |

### Model Switching

- **`/model`** — Interactively switch the active model mid-session; selection persists to user config.
- **`/preset max`** — Switches the entire session to a higher-tier model (e.g. DeepSeek V4 Pro).
- **Two-model collaboration** — Set `[agent].planner_model` in `reasonix.toml` to run a separate low-frequency planner (e.g. MiMo-v2.5-pro) alongside the executor, each in its own cache-stable session.
- **Subagent models** — `subagent_model` and `subagent_models` config keys let specific skills (like `review`, `security_review`) run on different models.
- **Multi-provider presets** — DeepSeek (flash/pro) and MiMo (mimo-v2.5-pro / mimo-v2.5) ship as config presets; any OpenAI-compatible endpoint is a config entry.

### Built-in Features

- **Plan mode** — Auto-plan (`/auto-plan on`) or manual (`/plan`); two-model planner with read-only research tools before executor acts.
- **Checkpoints & Rewind** — File-snapshot-based undo (`Esc-Esc` or `/rewind`); restore code, conversation, or both; fork-from-here; works without touching `.git`.
- **Goal mode** — `/goal <objective>` starts an autonomous, session-scoped active goal loop with blocked-state audit; desktop-only.
- **Memory** — Hierarchical `REASONIX.md` (committed) + `REASONIX.local.md` (personal, gitignored) + user-global `~/.config/reasonix/REASONIX.md` + ancestor dirs; `AGENTS.md` accepted as fallback.
- **`remember` tool** — Saves durable facts to per-project auto-memory (frontmatter files + `MEMORY.md` index), loaded into prefix on next session.
- **`#<note>` quick-add** — Type `#<note>` in chat to quick-add a line to the project's `REASONIX.md`.
- **`@path` imports** — A line containing `@path/to/another/file.md` injects that file's content into the memory prefix.
- **Session branching** — `/tree` shows saved conversation branches; `/branch [name]` forks current tip; `/branch <turn> [name]` forks from a checkpointed turn; `/switch <id|name>` loads another branch.
- **Subagent transcripts** — `/task` spawns sub-agents; continue/resume past transcripts in v1.5.0+.
- **MCP/Plugins** — First-class MCP client: stdio, HTTP (Streamable), and SSE transports; `.mcp.json` drop-in from Claude Code works unchanged.
- **Skills** — Reusable prompt templates in `.reasonix/skills/*.md`; frontmatter supports `runAs: subagent` and `allowed-tools` for isolated execution; `/skill new my-skill` scaffolds one.
- **`read_skill` tool** — Load inline skills in plan mode without execution (v1.5.0+).
- **Slash commands** — 20+ built-ins (`/compact`, `/new`, `/clear`, `/rewind`, `/tree`, `/branch`, `/switch`, `/todo`, `/model`, `/mcp`, `/skills`, `/hooks`, `/memory`, `/output-style`, `/sandbox`, `/language`, `/auto-plan`, `/help`); custom commands in `.reasonix/commands/*.md`.
- **Chat references (`@`)** — `@path/to/file` injects file contents (or directory listing) as tagged context; `@<server>:<uri>` injects MCP resources. Autocomplete on `/` or `@`.
- **Web search** — Mojeek, SearXNG, or Metaso backends.
- **Transcript replay & events** — Every event hits disk; `reasonix replay`, `reasonix events`, `reasonix stats`.
- **Bot gateway** — Feishu/Weixin/QQ adapters (v1.5.0+).
- **PDF extraction** — Extract text from PDF attachments (v1.5.0+).
- **Sandbox** — macOS Seatbelt enforcement for Bash (jailed to workspace + temp + toolchain caches, network opt-in); file-writers confined to `workspace_root` with symlink/`..` traversal prevention; Linux bubblewrap/landlock on roadmap.
- **Permisssions** — `deny > ask > allow > fallback` per-tool-call gating; tool-approval postures: ask, auto, yolo.
- **Context compaction** — Low-frequency cache-reset point when approaching `context_window` limit; older middle of session summarized; dropped originals archived as JSONL.
- **Codegraph** — Semantic code index for faster file discovery (daemon process with MCP interface).

---

## 2. Core Architecture & Cost Model

### Cache-First Philosophy

Reasonix is engineered around **DeepSeek's byte-stable prefix cache**. Every LLM API provider offers prefix caching, but DeepSeek's is particularly aggressive — if the start of a prompt matches a cached prefix exactly, the input token cost drops to ~1/5th the normal rate (per official reasonix.homes: ~94% cache hit, ~2.5× cost reduction). Any change invalidates the cache.

Generic tools restructure prompts for correctness. Reasonix structures them for **cache stability**:

1. **Fixed prefix ordering** — System prompt, project context, and memory are always in the same position. New information is appended, never inserted.
2. **Incremental context growth** — Files added to context go at the end. Nothing before them shifts.
3. **Stable tool call formatting** — Tool results are formatted identically every time.

### Real-World Cost Numbers

| Session Type | Input Tokens | Cache Hit Rate | Effective Cost | Without Cache |
|:-------------|:-------------|:---------------|:---------------|:--------------|
| Quick fix (5 min) | 50K | 99.5% | ~$0.01 | ~$0.04 |
| Feature (30 min) | 500K | 99.8% | ~$0.05 | ~$0.24 |
| Refactor (2 hrs) | 3M | 99.9% | ~$0.15 | ~$1.31 |
| Full day coding | 20M | 99.8% | ~$0.85 | ~$4.12 |

**Headline stat**: 435M input tokens processed with 99.82% cache hits, costing approximately **$12** instead of the **$61** it would have cost without cache optimization. (Real production data, not a benchmark.)

### Configuration

**Resolution order**: `flag > ./reasonix.toml > ~/.config/reasonix/config.toml > built-in defaults`.

Secrets come from the environment via `api_key_env` and are **never stored in config files**. A `.env` in the working directory is loaded if present. Step-limit preferences belong in user config; project `reasonix.toml` should override them only when the repository needs shared runtime bounds.

**Minimal `reasonix.toml`** — one provider and a default model is enough to start:

```toml
default_model = "deepseek-flash"

[[providers]]
name        = "deepseek-flash"
kind        = "openai"
base_url    = "https://api.deepseek.com"
model       = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
```

**Full example**:

```toml
default_model = "deepseek-flash"   # executor; set [agent].planner_model to add a planner

[agent]
max_steps = 0                    # executor tool-call rounds; 0 = no limit
planner_max_steps = 12           # planner read-only tool-call rounds; 0 = no limit
# planner_model = "mimo-pro"          # optional low-frequency planner
# subagent_model = "deepseek-pro"     # optional default for runAs=subagent skills
# subagent_models = { review = "deepseek-pro", security_review = "deepseek-pro" }
auto_plan = "off"                  # off|on; off keeps plan mode manual

[[providers]]
name           = "deepseek"
kind           = "openai"
base_url       = "https://api.deepseek.com"
models         = ["deepseek-v4-flash", "deepseek-v4-pro"]
default        = "deepseek-v4-flash"
api_key_env    = "DEEPSEEK_API_KEY"
context_window = 1000000

[[providers]]
name        = "mimo-pro"
kind        = "openai"
base_url    = "https://token-plan-cn.xiaomimimo.com/v1"
model       = "mimo-v2.5-pro"
api_key_env = "MIMO_API_KEY"

[tools]
enabled = []   # omit/empty = all built-ins
bash_timeout_seconds = 120

[skills]
paths = []            # extra custom skill roots
excluded_paths = []   # hide convention roots without deleting
disabled_skills = []  # hidden from prompt, slash invocation, and skill tools

[permissions]
mode  = "ask"                                # writer fallback: ask|allow|deny
deny  = ["Bash(rm -rf*)", "Bash(git push*)"] # hard-blocked in every mode
allow = ["Bash(go test:*)"]                  # never prompted

[sandbox]
# workspace_root = ""   # file-writers confined here; empty = current dir
# allow_write = ["/tmp"]

[[plugins]]
name    = "github"
command = "npx"
args    = ["-y", "@modelcontextprotocol/server-github"]

# Remote MCP server over Streamable HTTP
[[plugins]]
name    = "stripe"
type    = "http"
url     = "https://mcp.stripe.com"
headers = { Authorization = "Bearer ${STRIPE_KEY}" }
```

**`.mcp.json` drop-in support**: Reasonix also reads a project-root `.mcp.json` using Claude Code's exact `mcpServers` schema (`command`/`args`/`env`, `type`/`url`/`headers`, `${VAR}` expansion). Both sources are merged; on a name collision `reasonix.toml` wins.

```json
{
  "mcpServers": {
    "filesystem": { "command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"] },
    "stripe": { "type": "http", "url": "https://mcp.stripe.com", "headers": { "Authorization": "Bearer ${STRIPE_KEY}" } }
  }
}
```

---

## 3. Official Integration (DeepSeek Docs)

DeepSeek's official API documentation includes Reasonix as a **recommended agent integration** alongside Claude Code, GitHub Copilot, WorkBuddy, OpenCode, Kilo Code, and others.

**Installation**:

```bash
# Any OS — pulls the prebuilt Go native binary
npm i -g reasonix

# macOS
brew install esengine/reasonix/reasonix

# Prebuilt archives for all platforms (darwin|linux|windows × amd64|arm64)
# Available on every GitHub release, with SHA256SUMS

# Build from source
git clone https://github.com/esengine/deepseek-reasonix.git
cd deepseek-reasonix
make build      # -> bin/reasonix(.exe)
make cross      # -> dist/ (6 targets)
```

Windows builds are **code-signed** with a free certificate provided by the [SignPath Foundation](https://signpath.io/). Linux `.deb` packages are built on release.

**Quick-start** (from [deepseek.com](https://api-docs.deepseek.com/quick_start/agent_integrations/reasonix)):

```bash
# 1. Get a DeepSeek API Key from platform.deepseek.com
# 2. Run the config wizard:
reasonix setup                      # writes ./reasonix.toml

# 3. Start chatting:
export DEEPSEEK_API_KEY=sk-...      # or put it in .env
reasonix chat                       # then run /init to generate AGENTS.md

# Or one-shot:
reasonix run "implement the TODOs in main.go"
reasonix run --model mimo-pro "add unit tests for this function"
echo "explain this code" | reasonix run
```

No Node.js required for v1.0+ (single Go binary). First launch with `reasonix setup` writes a minimal `reasonix.toml`. Secrets stay in environment variables (`api_key_env`), never in config files.

---

## 4. Desktop Application v1.5.0

**Built into the main repo**: [desktop/](https://github.com/esengine/deepseek-reasonix/tree/main-v2/desktop) directory in `esengine/DeepSeek-Reasonix`.  
**Stack**: Wails + React 19 + TypeScript 6 | **Latest**: v1.5.0 (Jun 10, 2026)

The desktop app is a **visual companion to the CLI**, not a Cursor replacement. It provides a GUI wrapper around the same Go controller — the terminal TUI, desktop webview, and HTTP/SSE server all drive the same `control.Controller`, so every feature surfaces identically across frontends.

### Key Desktop Features

- **Themed workspace UI** — Dark and light themes with a full sidebar: session list, file tree, MCP settings panel, skills browser, and memory view. Screenshots at `docs/desktop-theme-dark.jpg` and `docs/desktop-theme-light.jpg`.
- **Collaboration-mode picker** — Three choices: **normal** (standard chat), **plan** (`/plan`), and **goal** (`/goal`).
- **Goal mode** (`/goal <objective>`) — Starts an autonomous, session-scoped active goal loop. The controller prepends goal context to user turns (outside the cache-stable system prompt) and keeps issuing continuation turns until the model reports completion, repeats the same blocked state three times, or the safety limit is reached. Blocked-state matching is normalized for casing, whitespace, and punctuation. `/goal clear` removes the active goal.
- **Tool-approval postures** — **ask** (each writer/bash call prompts), **auto** (writer fallback auto-allowed, explicit ask/deny rules still apply), **yolo** (approval prompts auto-allowed unless denied). These are separate from the collaboration-mode picker.
- **`read_skill` tool** — Load inline skills in plan mode without execution (v1.5.0+).
- **Session workspace isolation** — Desktop sessions are isolated per workspace directory; switching workspaces loads the correct session history.
- **MCP settings panel** — View connected servers, connection status, retry failures, inspect each server's tools/prompts/resources. Collapses noisy startup errors into a summary.
- **Reveal-in-file-manager** — Right-click any file in the workspace tree to reveal it in Finder/Explorer.
- **Image paste** — Paste clipboard images directly into the chat composer.
- **Model selector dropdown** — Switch models interactively; scroll support and tooltips for truncated names; preserves curated provider models on refresh.
- **Rewind (hover)** — Each user message in the transcript has a hover rewind control → menu: rewind code / rewind conversation / both / fork-from-here.
- **Subagent transcript continue/resume** — Continue past subagent transcripts from where they left off (v1.5.0+).
- **PDF attachment text extraction** — Extract text from PDF attachments dropped into the chat (v1.5.0+).

### Bot Gateway (v1.5.0+)

Reasonix ships a **bot gateway** with adapters for:
- **Feishu (飞书)** — Chinese enterprise messaging
- **Weixin (微信)** — China's dominant messaging app
- **QQ** — Tencent's instant messaging platform

The gateway receives messages from these platforms, routes them through the Reasonix agent, and sends responses back. Each adapter handles authentication (Feishu webhook tokens, Weixin account/login). This turns Reasonix into a deployable AI coding assistant accessible from chat platforms — useful for teams that want a shared coding agent reachable from their existing messaging tools.

### Desktop vs CLI

| Feature | CLI (TUI) | Desktop |
|---------|-----------|---------|
| Chat | Full transcript, scrolling (Ctrl+Home/End in v1.5.0) | Visual transcript with hover controls |
| Plan mode | `/plan` | Collaboration-mode picker |
| Goal mode | Not available | `/goal` with blocked-state audit |
| Checkpoints/Rewind | `Esc-Esc` / `/rewind` picker | Hover rewind on each message |
| MCP management | `/mcp` slash command | Full settings panel with retry |
| Skills | `/skills` | Visual browser |
| File tree | `/tree` | Sidebar with reveal-in-file-manager |
| Image paste | Via file path | Clipboard paste |
| Bot gateway | Available via serve mode | Included |

---

## 5. MCP Bridges & Cross-Platform Integrations

### 5.1 reasonix-mcp-server (Claude Code ↔ Reasonix)

**Repo**: [github.com/Belveth02/reasonix-mcp-server](https://github.com/Belveth02/reasonix-mcp-server)  
**Stars**: ⭐2 | **Language**: JavaScript | **License**: MIT

This MCP server lets **Claude Code call Reasonix as a DeepSeek sub-agent** through the Model Context Protocol. The core idea: use Claude for high-level reasoning and orchestration, but delegate the actual file-editing work to Reasonix (which costs ~1/5 as much thanks to prefix caching).

#### Tools Exposed

| Tool | Parameters | Description |
|------|------------|-------------|
| `reasonix_run` | `task` (string, required), `model` (string, optional), `effort` (enum: low/medium/high/max), `budget` (number, optional), `timeout` (number, default 300000ms) | Execute a coding task via Reasonix |
| `reasonix_doctor` | `json` (boolean, optional) | Run health check: API key, network, config |

#### Installation

```bash
npm install -g reasonix
git clone https://github.com/Belveth/reasonix-mcp-server.git
cd reasonix-mcp-server
npm install
```

Register with Claude Code:

```bash
claude mcp add reasonix --scope user -- \
  --command node \
  --args "/absolute/path/to/server.js"
```

Or manually edit `~/.claude.json`:

```json
"mcpServers": {
  "reasonix": {
    "type": "stdio",
    "command": "node",
    "args": ["C:\\Users\\xxx\\.claude\\tools\\reasonix-mcp-server\\server.js"],
    "env": {}
  }
}
```

#### Cost Comparison

In real-world testing: 435M tokens processed at ~$12 via Reasonix/DeepSeek. Equivalent work on Claude: ~$61. That's an **~80% cost reduction** by routing the heavy file-editing work through Reasonix's cache-optimized DeepSeek pipeline.

---

### 5.2 reasonix-bridge (Codex ↔ DeepSeek, Multi-Agent)

**Repo**: [github.com/OwlCodeTech/reasonix-bridge](https://github.com/OwlCodeTech/reasonix-bridge)  
**Stars**: ⭐6 | **Language**: JavaScript (~1500 lines, single file) | **License**: MIT

A **multi-agent parallel execution engine** that lets **Codex delegate tasks to DeepSeek via MCP**. Supports two-phase execution, multi-agent parallelism, incremental mode, and dynamic budget allocation.

#### Architecture

```
Codex (MCP Client)
  │  stdio pipe
  ▼
index.js (MCP Server)
  │
  ├─ 6 MCP tools
  ├─ 8 local sandbox tools (read_file / write_file / run_command / ...)
  └─ DeepSeek API (streaming)
       ├─ Planning mode (no tools, produces plan)
       └─ Execution mode (8 tools, agent loop)
```

#### Tools Exposed

| Tool | Purpose | Recommended Use |
|------|---------|-----------------|
| `execute_task` | Auto-detect: single or multi-agent | Daily use |
| `plan_task` | Generate plan only, no execution | "Show me the plan first" |
| `execute_step` | Execute a specific step | Pair with `plan_task` |
| `orchestrate_task` | Force multi-agent parallelism | When explicit division of labor is needed |
| `improve_file` | DeepSeek-targeted fix; Codex retains override | When Codex finds something to fix |
| `delegate_to_reasonix` | Legacy one-shot execution | Backward compatibility |

#### The "Scientist Team" (Role-Based Parallel Agents)

When a task requires multi-agent execution, Bridge assigns roles automatically:

| Emoji | Scientist | Role | Domain |
|:-----|:----------|:-----|:--------|
| 🖥️ | Dijkstra | Backend Engineer | Servers, APIs, algorithms |
| 🎨 | Hopper | Frontend Developer | UI, interactions, components |
| 🗄️ | Turing | Data Engineer | Database, storage |
| 🔗 | Berners-Lee | Interface Engineer | API communication |
| 🧪 | Lovelace | Test Engineer | Validation, deployment |

**Sub-agent decomposition**: A single-domain complex task auto-splits:
```
🖥️ Dijkstra
  ├─ Dijkstra-A (user auth module)
  ├─ Dijkstra-B (CRUD operations)
  └─ Dijkstra-C (file upload)
```

#### Two-Phase Execution

1. **Phase 1 (Planning)**: DeepSeek decomposes task → JSON step list (5-10 seconds)
2. **Phase 2 (Execution)**: Step-by-step or multi-agent parallel (each step has independent context)
   - Previous outputs already on disk, unaffected by subsequent budget limits

**Parallel scheduling**: Independent sub-tasks → `Promise.all` parallel execution. Dependent sub-tasks → wait for dependencies. Interface contracts extracted after each step and passed to subsequent agents.

#### Security Model: 5-Level Command Control

| Mode | Permissions |
|------|-------------|
| `off` | No commands, read-only files |
| `readonly` | `pwd`, `echo`, `which` |
| `static` | + `git` (read-only), `node --check`, `tsc` |
| `verify` | + `npm test`, `eslint`, `jest` |
| `full` | All open + write files |

Additional security: file sandbox (validates all paths, rejects traversal/symlink escapes), hash-based overwrite prevention, read-only mode auto-disables `edit_file`/`write_file`, forced-write audit logging, `REASONIX_MAX_TOKENS` hard cap.

#### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DEEPSEEK_API_KEY` | — | Required |
| `REASONIX_COMMAND_MODE` | `static` | Command permission level |
| `WORKSPACE_ROOT` | `cwd` | Sandbox root directory |
| `DEEPSEEK_MODEL` | `deepseek-chat` | Model identifier |
| `DEEPSEEK_TEMPERATURE` | `0.2` | Temperature |
| `REASONIX_MAX_TOKENS` | `64000` | Cumulative token budget |
| `HTTP_PROXY` | — | Proxy (for non-China API access) |
| `GITHUB_TOKEN` | — | GitHub API rate-limit boost |

---

### 5.3 opencode-reasonix-mcp (OpenCode Bridge)

**Repo**: [github.com/StarWHAT-BUG/opencode-reasonix-mcp](https://github.com/StarWHAT-BUG/opencode-reasonix-mcp)  
**Stars**: ⭐0 | **Language**: JavaScript

Dispatches **stateless sub-tasks from OpenCode to Reasonix CLI** via MCP, saving ~99% of token costs compared to running the same tasks on Claude or GPT, by routing them through DeepSeek's prefix-cached pipeline.

---

### 5.4 collab-cli (Universal Multi-Agent Protocol)

**Repo**: [github.com/Staryinsang0910-star/collab-cli](https://github.com/Staryinsang0910-star/collab-cli)  
**Stars**: ⭐7 | **Language**: TypeScript | **Tags**: `mcp`, `collaboration`, `protocol`, `multi-agent`

A **universal collaboration protocol + CLI** for multi-agent LLM teams. Supports **Claude Code, Reasonix, Codex, WorkBuddy, and Cursor** as peers in a shared agent mesh. This is the closest project found to an **ACP (Agent Communication Protocol)** implementation — it defines how different agent systems discover each other, delegate work, share context, and synchronize results.

---

## 6. Community Skills Hub (awesome-reasonix)

**Repo**: [github.com/hikari-2424/awesome-reasonix](https://github.com/hikari-2424/awesome-reasonix)  
**Stars**: ⭐2 | **Language**: HTML / Markdown | **License**: MIT  
**Site**: [hikari-2424.github.io/awesome-reasonix-site/](https://hikari-2424.github.io/awesome-reasonix-site/)

An **independent community project** (not affiliated with Reasonix) that collects, rates, and shares community-contributed Skills. Skills are Markdown files dropped into `.reasonix/skills/` and invoked via `/skill <name>`.

**Registry version**: 3 | **Last updated**: 2026-05-30 | **80+ published skills**

### Installation

```bash
# Single skill
curl -o .reasonix/skills/file-organizer.md https://raw.githubusercontent.com/hikari-2424/awesome-reasonix/main/skills/file-organizer.md

# Then inside Reasonix:
> /skill file-organizer
```

### 6.1 Full Skill Registry

#### File & System

| Skill | Description |
|-------|-------------|
| `file-organizer` | Classify files by extension, find duplicates, tree view, clean empty dirs |
| `disk-cleaner` | Disk space analysis and cleanup automation |

#### Text Processing

| Skill | Description |
|-------|-------------|
| `text-polish` | Polish, translate, summarize, rewrite — 4 output styles (professional/casual/concise/tech-doc) |
| `data-convert` | Convert data between formats (JSON, CSV, YAML, etc.) |
| `crosspost` | Cross-post content across platforms |

#### Learning & Research

| Skill | Description |
|-------|-------------|
| `learn-topic` | Search → fetch → organize → summarize (subagent mode) |
| `deep-research` | Deep research agent with iterative retrieval |
| `data-scraper-agent` | Web scraping agent |

#### System Control

| Skill | Description |
|-------|-------------|
| `input-control` | Windows mouse/keyboard automation via PowerShell |
| `knowledge-ops` | Knowledge base management operations |

#### Development Tools

| Skill | Description |
|-------|-------------|
| `git-helper` | Conventional Commits, PR descriptions, history analysis, conflict resolution |
| `code-review` | 4-dimension review (logic, security, performance, style) with file:line output |
| `changelog-generator` | Auto-generate CHANGELOG.md from git history, semver inference |
| `database-migrations` | Database migration patterns and automation |
| `deployment-patterns` | Deployment strategies and CI/CD patterns |
| `docker-patterns` | Docker and containerization patterns |
| `github-ops` | GitHub operations automation |

#### Project Generation

| Skill | Description |
|-------|-------------|
| `project-scaffold` | Interactive scaffolding: Python, TypeScript, Go, Rust |
| `readme-gen` | Auto-generate README.md from project structure analysis |
| `api-connector-builder` | Build API connectors matching existing integration patterns |

#### API & Integration

| Skill | Description |
|-------|-------------|
| `api-design` | REST API design patterns: naming, status codes, pagination, filtering, error responses, versioning |
| `jira-integration` | Jira issue tracking integration |

#### Content & Design

| Skill | Description |
|-------|-------------|
| `article-writing` | Write articles, guides, blog posts, tutorials, newsletters |
| `brand-voice` | Extract writing style profiles from real content, build reusable brand voice config |
| `code-tour` | Create step-by-step CodeTour files with real file:line anchors |
| `dashboard-builder` | Build dashboards and data visualizations |
| `frontend-slides` | Create HTML presentations from scratch or convert PowerPoint |
| `content-engine` | Content generation and management engine |
| `liquid-glass-design` | Modern UI design patterns |

#### AI & Agent

| Skill | Description |
|-------|-------------|
| `autonomous-loops` | Autonomous agent loop patterns: from simple pipelines to RFC-driven multi-agent workflows |
| `blueprint` | Turn one-line objective into multi-phase, multi-agent engineering plan with dependency graph |
| `council` | Multi-agent discussion and decision-making council |
| `continuous-learning` | Auto-extract reusable patterns from Claude Code sessions |
| `continuous-learning-v2` | Instinct-based learning system with confidence scoring |
| `agentic-engineering` | Agent-driven engineering patterns |
| `agent-harness-construction` | Build agent harnesses for testing and evaluation |
| `agent-introspection-debugging` | Debug and introspect agent behavior |
| `agent-sort` | Agent categorization and sorting |
| `ai-first-engineering` | AI-first development methodology |
| `eval-harness` | Formal evaluation framework for AI sessions |
| `cost-aware-llm-pipeline` | Cost-optimized LLM pipeline patterns |
| `iterative-retrieval` | Progressive context retrieval pattern |

#### Architecture & Standards

| Skill | Description |
|-------|-------------|
| `coding-standards` | Cross-project coding conventions: naming, readability, immutability, review criteria |
| `backend-patterns` | Backend architecture: API design, database optimization, Node.js/Express/Next.js best practices |
| `frontend-patterns` | Frontend patterns: React, Next.js, state management, performance |
| `frontend-design` | UI/UX design patterns |
| `android-clean-architecture` | Clean Architecture for Android/Kotlin Multiplatform |
| `compose-multiplatform-patterns` | Compose Multiplatform / Jetpack Compose: state, navigation, theming, testing |
| `dart-flutter-patterns` | Dart and Flutter patterns |
| `kotlin-patterns` | Kotlin patterns and best practices |
| `kotlin-coroutines-flows` | Coroutines and Flow patterns |
| `kotlin-exposed-patterns` | JetBrains Exposed ORM patterns |
| `kotlin-ktor-patterns` | Ktor framework patterns |
| `java-coding-standards` | Java coding standards for Spring Boot |
| `jpa-patterns` | JPA/Hibernate patterns |
| `springboot-patterns` | Spring Boot architecture and patterns |
| `django-patterns` | Django architecture patterns |
| `django-security` | Django security best practices |
| `laravel-patterns` | Laravel framework patterns |
| `laravel-security` | Laravel security practices |
| `golang-patterns` | Idiomatic Go patterns |
| `cpp-coding-standards` | C++ Core Guidelines-based standards |
| `dotnet-patterns` | .NET architecture patterns |
| `perl-patterns` | Perl patterns and best practices |
| `rust-patterns` | Rust patterns and best practices |
| `clickhouse-io` | ClickHouse database patterns and optimization |

#### Testing

| Skill | Description |
|-------|-------------|
| `django-tdd` | Django testing with pytest-django, TDD methodology |
| `django-verification` | Django verification loop: migrations, linting, tests, security |
| `laravel-tdd` | Laravel testing patterns |
| `laravel-verification` | Laravel verification loop |
| `springboot-tdd` | Spring Boot testing patterns |
| `springboot-verification` | Spring Boot verification loop |
| `golang-testing` | Go testing: table-driven, subtests, benchmarks, fuzzing |
| `cpp-testing` | C++ testing with GoogleTest/CTest |
| `kotlin-testing` | Kotlin testing patterns |
| `csharp-testing` | C# testing patterns |
| `e2e-testing` | Playwright E2E testing, Page Object Model, CI/CD |
| `ai-regression-testing` | Regression testing for AI-assisted development |

#### Business & Operations

| Skill | Description |
|-------|-------------|
| `carrier-relationship-management` | Carrier portfolio management, freight rate negotiation, performance tracking |
| `customer-billing-ops` | Customer billing operations |
| `customs-trade-compliance` | Customs and trade compliance |
| `energy-procurement` | Energy procurement strategies |
| `enterprise-agent-ops` | Enterprise agent operations |
| `finance-billing-ops` | Finance and billing operations |
| `inventory-demand-planning` | Inventory and demand planning |
| `investor-materials` | Investor materials generation |
| `investor-outreach` | Investor outreach automation |
| `lead-intelligence` | Lead intelligence and scoring |
| `logistics-exception-management` | Logistics exception handling |
| `connections-optimizer` | Network/connections optimization |

#### Security & Compliance

| Skill | Description |
|-------|-------------|
| `defi-amm-security` | DeFi AMM security patterns |
| `llm-trading-agent-security` | LLM trading agent security |
| `evm-token-decimals` | EVM token decimal handling |
| `healthcare-phi-compliance` | Healthcare PHI compliance |
| `hipaa-compliance` | HIPAA compliance patterns |
| `ecc-tools-cost-audit` | Tool cost auditing for ECC |

### 6.2 Skill Categories

| # | Category | Count |
|---|----------|-------|
| 1 | AI & Agent | ~15 |
| 2 | Architecture & Standards | ~25 |
| 3 | Testing | ~12 |
| 4 | Backend & API | ~10 |
| 5 | Frontend & UI | ~6 |
| 6 | Business & Operations | ~12 |
| 7 | Security & Compliance | ~6 |
| 8 | Content & Design | ~7 |
| 9 | File & System | ~3 |
| 10 | Learning & Research | ~3 |
| 11 | Project Generation | ~3 |
| 12 | Text Processing | ~3 |

### Rating & Safety System

The awesome-reasonix project includes:
- **Validation pipeline**: Automated checks on PRs validate format and safety. Pass → published immediately. Fail → comment explains why.
- **Rating system**: `ratings.json` tracks community ratings. Low-rated skills hidden automatically.
- **Quarantine**: `quarantine.json` for flagged skills.
- **Security policy**: `SECURITY.md` + `CODE_OF_CONDUCT.md`. Skills run on your machine — read the file, check ratings, test first.

---

## 7. Long-Term Memory: Hindsight-Reasonix

**Repo**: [github.com/houycth/Hindsight-Reasonix](https://github.com/houycth/Hindsight-Reasonix)  
**Stars**: ⭐0 | **Language**: Python | **Dependencies**: `hindsight-client>=0.4.22`, `mcp>=1.0.0`

A **cross-session long-term memory system** adapted for Reasonix's Hook and MCP mechanisms. Uses the Hindsight API backend for vector storage and retrieval.

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  hindsight_hooks.py  ← auto-triggered by Reasonix hooks      │
│  ├─ session-start   → health check                          │
│  ├─ pre-turn        → recall memory → stdout injection → LLM │
│  └─ stop            → read JSONL → retain conversation turn  │
├──────────────────────────────────────────────────────────────┤
│  hindsight_mcp.py    ← optional MCP server for active search  │
│  ├─ hindsight_recall(query)    → search memory                │
│  ├─ hindsight_reflect(query)   → synthesize                   │
│  └─ hindsight_retain(...)      → manual save                  │
└──────────────────────────────────────────────────────────────┘
```

### Data Flow

```
Turn N:
  UserPromptSubmit → pre-turn:
    ① recall(user message) → stdout → Reasonix captures & injects as <memory-context>
    ② LLM sees memory automatically (no manual reading required)

Stop → stop:
    ① Get session_id from hook payload
    ② Locate session JSONL → read last user message
    ③ Strip <memory-context> injection block
    ④ retain(original user message + assistant reply) → store in Hindsight
```

### Key Features

- **Auto-prefetch**: Pre-turn queries memory and injects via stdout. LLM sees it 100% automatically.
- **Auto-write**: Stop hook ensures every turn is stored (without injected `<memory-context>` content).
- **Active search**: LLM can use MCP tools `hindsight_recall` for deep retrieval.

### Setup

Three Reasonix hooks must be configured in `~/.reasonix/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [{ "command": "python hindsight_hooks.py session-start", "timeout": 15000 }],
    "UserPromptSubmit": [{ "command": "python hindsight_hooks.py pre-turn", "timeout": 30000 }],
    "Stop": [{ "command": "python hindsight_hooks.py stop", "timeout": 30000 }]
  }
}
```

**Note**: Requires patched Reasonix source — specifically `internal/hook/hook.go`, `internal/hook/runner.go`, `internal/control/controller.go`, and `internal/agent/save.go` — to pass `session_id` through the hook payload and strip `<memory-context>` from saved user messages.

---

## 8. Portable Toolkit: Reasonix PortaKit

**Repo**: [github.com/CS-Faith/reasonix-portakit](https://github.com/CS-Faith/reasonix-portakit)  
**Stars**: ⭐8 | **Language**: PowerShell + JavaScript + Batch | **License**: MIT  
**Latest Release**: v1.0.0 — June 8, 2026

Makes Reasonix **truly portable** — USB drive or cloud-sync folder, plug into any Windows machine, and all memories, conversations, skills, and MCP configurations come with you.

### The Problem

| Symptom | Root Cause |
|---------|------------|
| Memories vanish | Project memory directory hashed by workspace path; path changes → hash mismatch |
| Conversation history blank | Session metadata stores old workspace path; mismatch → all filtered out |
| Skills / MCP broken | `config.json` hardcodes MCP paths with old drive letter (E:\Reasonix) |
| Nothing reads from here | Reasonix defaults to `C:\Users\...`, never looks at current directory |

### How It Works

```
双击 启动Reasonix.bat
  │
  ├─ 1. Auto-detect current path (any drive, any folder)
  ├─ 2. Tell Reasonix: "Your data is here, not in C:\Users\..."
  ├─ 3. SHA1(path) → project hash → create dirs; auto-rename/merge old hash dirs
  ├─ 4. Merge local machine's Reasonix data (newer-only, no overwrites)
  ├─ 5. Patch config.json — old paths → current path
  ├─ 6. Repair all session metadata (strip BOM + fix JSON escaping + align workspace)
  └─ 7. Launch Reasonix
```

### Files

| File | Size | Purpose |
|------|------|---------|
| `启动Reasonix.bat` | 0.6 KB | Entry point |
| `_patch_config.ps1` | 6 KB | Core engine |
| `_fix_sessions.js` | 2.7 KB | Session metadata repair |
| `PortaKit-核心规则.md` | 2.4 KB | Core rules for Reasonix to remember portable setup |

### Usage

1. Ensure Reasonix has been launched at least once (API key configured)
2. Copy PortaKit files into the Reasonix directory (next to `reasonix-desktop.exe`)
3. Always launch via `启动Reasonix.bat` — never double-click `reasonix-desktop.exe` directly
4. (Recommended) Import `PortaKit-核心规则.md` into Reasonix so it learns the portable mechanism
5. After one bat launch, the entire folder becomes fully portable:
   - USB drive → any Windows PC → double-click bat → everything intact
   - Sync folder (OneDrive, Dropbox, etc.) → new PC sync → double-click bat → local data merged in

---

## 9. Protocol Landscape: MCP / ACP / CDP

### MCP (Model Context Protocol) — ✅ Well Established

MCP is the **dominant extension protocol** in the Reasonix ecosystem. Reasonix is a first-class MCP client with three transport types:

- **`stdio`** (default) — launches a local subprocess; one JSON message per line over stdin/stdout
- **`http`** (Streamable HTTP) — remote server at a URL; POST with optional SSE stream
- **`sse`** — legacy 2024-11-05 transport; recognized but deferred (deprecated upstream)

Tools surface to the model as `mcp__<server>__<tool>`; a tool declaring MCP's `readOnlyHint: true` joins parallel dispatch. MCP prompts appear as `/mcp__<server>__<prompt>` slash commands; MCP resources are pulled in via `@<server>:<uri>`.

**`.mcp.json` drop-in**: Reasonix reads a project-root `.mcp.json` using Claude Code's exact `mcpServers` schema — an existing Claude Code MCP config works in Reasonix unchanged.

**`/mcp`** — Lists connected servers and what each exposes.

| Integration | Type | MCP Tools |
|-------------|------|-----------|
| reasonix-mcp-server | Claude Code ↔ Reasonix | `reasonix_run`, `reasonix_doctor` |
| reasonix-bridge | Codex ↔ DeepSeek | `execute_task`, `plan_task`, `orchestrate_task`, etc. |
| opencode-reasonix-mcp | OpenCode ↔ Reasonix | Task dispatch |
| Hindsight-Reasonix (MCP mode) | Memory retrieval | `hindsight_recall`, `hindsight_reflect`, `hindsight_retain` |
| codegraph (built-in) | Semantic code index | Daemon process with MCP interface |

Reasonix has **native MCP support** built into its TOML config:

```toml
[[plugins]]
name    = "hindsight"
command = "python"
args    = ["/path/to/hindsight_mcp.py"]

# Remote MCP over Streamable HTTP
[[plugins]]
name    = "stripe"
type    = "http"
url     = "https://mcp.stripe.com"
headers = { Authorization = "Bearer ${STRIPE_KEY}" }
```

The reference MCP stdio plugin (`cmd/reasonix-plugin-example`) ships as a runnable example with `echo`, `wordcount`, a `review` prompt, and a `style-guide` resource.

### ACP (Agent Communication Protocol) — ✅ Implemented (v1.5.0+)

**Confirmed in the v1.5.0 changelog** (Jun 10, 2026):

```
fix(cli): disable codegraph in ACP session config test (#3663)
```

This confirms Reasonix has a native **ACP (Agent Communication Protocol)** implementation. ACP sessions run with codegraph indexing disabled to avoid daemon interference, which means ACP is a distinct session mode with its own constraints.

Other ACP-adjacent projects and patterns in the ecosystem:

- **[collab-cli](https://github.com/Staryinsang0910-star/collab-cli)** (⭐7) — Universal collaboration protocol + CLI for multi-agent LLM teams. Supports Claude Code, Reasonix, Codex, WorkBuddy, and Cursor as peers in a shared agent mesh. Defines how heterogeneous agents discover each other, delegate work, share context, and synchronize results.
- **reasonix-bridge** — The "Scientist Team" pattern defines role-based inter-agent communication (Dijkstra ↔ Hopper ↔ Turing, etc.) with interface contract extraction between parallel agents.
- **council skill** (awesome-reasonix) — Multi-agent discussion and decision-making pattern.
- **Two-model collaboration** (Reasonix native) — The Coordinator runs an executor and planner in separate cache-stable sessions, handing off structured plans between them — a form of intra-tool agent communication.

### CDP (Context Data Protocol) — 🔶 Emerging

No project explicitly branded "CDP" was found. The closest implementations:

- **[Hindsight-Reasonix](https://github.com/houycth/Hindsight-Reasonix)** — Provides cross-session context persistence via MCP tools (`hindsight_recall`, `hindsight_retain`, `hindsight_reflect`). Functions as a context data layer that survives across sessions.
- **Reasonix's built-in memory** (`.reasonix/memory.md`) — Persistent across sessions, loaded at startup.
- **Reasonix's semantic index** — Codebase-wide context for faster file discovery.

---

## 10. Use Cases & Workflows

### 10.1 Low-Cost AI Coding (Daily Development)

**Stack**: Reasonix core (DeepSeek V4 Flash default)

The primary use case. Point Reasonix at a codebase, describe what you want, and it reads files, writes code, runs commands, and iterates. Costs $0.03–$0.15 per typical session thanks to 99.8% prefix cache hits.

**Best for**: Feature development, bug fixing, refactoring, code review, test generation.

```bash
cd your-project
reasonix chat
> Add input validation to the signup form. Email must be valid, password minimum 8 chars.
> Also add a confirm password field that must match.
```

### 10.2 Hybrid Claude Code + DeepSeek Workflow

**Stack**: Claude Code + reasonix-mcp-server + Reasonix (DeepSeek)

Use Claude Code for high-level reasoning, architecture design, and complex orchestration. Route the heavy file-editing work to Reasonix/DeepSeek via MCP to save ~80% on token costs.

**Best for**: Teams already using Claude Code who want to reduce API spend without sacrificing quality.

```
Claude Code (orchestrator)
  → reasonix_run("refactor auth module to use JWT")
    → Reasonix/DeepSeek (executor, $0.05)
  → Claude reviews the diff
  → reasonix_run("add tests for the new JWT module")
```

### 10.3 Multi-Agent Parallel Development

**Stack**: Codex + reasonix-bridge + Scientist Team

Codex delegates complex multi-module tasks to reasonix-bridge, which automatically decomposes the work and assigns parallel sub-agents (Dijkstra for backend, Hopper for frontend, Turing for database, etc.).

**Best for**: Full-stack feature development, multi-module refactors, greenfield project scaffolding.

```
Codex → execute_task("Build a blog system with backend API, frontend UI, and database")
  → Bridge auto-detects complexity → multi-agent mode
    ├─ Dijkstra (backend): API routes, controllers, services
    ├─ Hopper (frontend): React components, pages, routing
    └─ Turing (database): schema, migrations, queries
  → Parallel execution, interface contracts extracted
  → All results merged
```

### 10.4 CI/CD Automation

**Stack**: `reasonix run "task"` (non-interactive, piped input)

The `run` mode is designed for automation. It accepts piped input, runs non-interactively, and exits with appropriate status codes.

**Best for**: Automated code review, test generation, documentation updates, security scanning.

```bash
# In CI pipeline:
git diff HEAD~1 | reasonix run "review these changes for security issues"
reasonix run "generate documentation for the ./api directory"
reasonix run "update CHANGELOG.md based on git log since last tag"
```

### 10.5 Cross-Session Persistent Memory

**Stack**: Hindsight-Reasonix (hooks + MCP)

Every conversation turn is automatically recalled (pre-turn) and retained (stop). Memory persists across sessions without manual management. The optional MCP server enables active search.

**Best for**: Long-running projects where institutional knowledge must accumulate over weeks/months; onboarding new agents to existing codebases.

```
Session 1: "Set up the project structure"
  → Hindsight retains: architecture decisions, tech stack, conventions

Session 2 (next day): "Add the payment module"
  → Hindsight recalls previous session's context
  → LLM sees it automatically as <memory-context>
  → No need to re-explain the project structure
```

### 10.6 Portable Development Environment

**Stack**: Reasonix PortaKit + USB drive / cloud sync

Carry your entire Reasonix environment (memories, skills, MCP config, conversations) on a USB drive. Plug into any Windows machine, double-click the bat file, and everything is exactly where you left it.

**Best for**: Developers who work across multiple machines (work PC, home PC, laptop); freelancers; educators.

### 10.7 Community Skill Extensions

**Stack**: awesome-reasonix skills

Extend Reasonix with 80+ community-contributed skills covering everything from file organization to multi-agent councils. New skills can be installed in seconds.

**Best for**: Customizing Reasonix for specific workflows, languages, or domains.

```bash
# Install
curl -o .reasonix/skills/deep-research.md https://raw.githubusercontent.com/.../deep-research.md
# Use
> /skill deep-research "Compare the top 5 Rust web frameworks"
```

### 10.8 Multi-Agent Collaboration Protocol

**Stack**: collab-cli

Use multiple different agent systems (Claude Code + Reasonix + Codex + Cursor) in a coordinated mesh, each playing to its strengths.

**Best for**: Teams using heterogeneous agent tooling; complex pipelines requiring specialized agents.

---

## 11. Appendix: Quick-Reference Links

### Core

| Resource | URL |
|----------|-----|
| Reasonix Main Repo (v1.0 Go) | [github.com/esengine/DeepSeek-Reasonix](https://github.com/esengine/deepseek-reasonix) |
| Official Website | [reasonix.homes](https://reasonix.homes) |
| DeepSeek Official Docs | [api-docs.deepseek.com/.../reasonix](https://api-docs.deepseek.com/quick_start/agent_integrations/reasonix) |
| npm Package | [npmjs.com/package/reasonix](https://www.npmjs.com/package/reasonix) |
| brew Formula | `brew install esengine/reasonix/reasonix` |
| Architecture Docs | [esengine.github.io/DeepSeek-Reasonix](https://esengine.github.io/DeepSeek-Reasonix/) |

### Official Docs (in repo)

| Document | URL |
|----------|-----|
| GUIDE.md (Configuration & Usage) | [github.com/.../docs/GUIDE.md](https://github.com/esengine/deepseek-reasonix/blob/main-v2/docs/GUIDE.md) |
| GUIDE.zh-CN.md (简体中文指南) | [github.com/.../docs/GUIDE.zh-CN.md](https://github.com/esengine/deepseek-reasonix/blob/main-v2/docs/GUIDE.zh-CN.md) |
| SPEC.md (Engineering Contract) | [github.com/.../docs/SPEC.md](https://github.com/esengine/deepseek-reasonix/blob/main-v2/docs/SPEC.md) |
| CHECKPOINTS.md (Rewind Design) | [github.com/.../docs/CHECKPOINTS.md](https://github.com/esengine/deepseek-reasonix/blob/main-v2/docs/CHECKPOINTS.md) |
| MIGRATING.md (0.x → 1.0) | [github.com/.../docs/MIGRATING.md](https://github.com/esengine/deepseek-reasonix/blob/main-v2/docs/MIGRATING.md) |
| REASONIX.md (Project Memory) | [github.com/.../REASONIX.md](https://github.com/esengine/deepseek-reasonix/blob/main-v2/REASONIX.md) |
| SESSION_REFERENCE_ARCHITECTURE.md | [github.com/.../docs/SESSION_REFERENCE_ARCHITECTURE.md](https://github.com/esengine/deepseek-reasonix/blob/main-v2/docs/SESSION_REFERENCE_ARCHITECTURE.md) |

### Community

| Resource | URL |
|----------|-----|
| Discord (bilingual) | [discord.gg/XF78rEME2D](https://discord.gg/XF78rEME2D) |
| GitHub Issues | [github.com/.../issues](https://github.com/esengine/deepseek-reasonix/issues) (266 open) |
| GitHub Discussions | [github.com/.../discussions](https://github.com/esengine/deepseek-reasonix/discussions) |
| Pull Requests | [github.com/.../pulls](https://github.com/esengine/deepseek-reasonix/pulls) (48 open) |

### MCP Bridges

| Resource | URL |
|----------|-----|
| reasonix-mcp-server (Claude Code ←→ Reasonix) | [github.com/Belveth02/reasonix-mcp-server](https://github.com/Belveth02/reasonix-mcp-server) |
| reasonix-bridge (Codex ←→ DeepSeek) | [github.com/OwlCodeTech/reasonix-bridge](https://github.com/OwlCodeTech/reasonix-bridge) |
| opencode-reasonix-mcp | [github.com/StarWHAT-BUG/opencode-reasonix-mcp](https://github.com/StarWHAT-BUG/opencode-reasonix-mcp) |
| collab-cli (Universal Protocol) | [github.com/Staryinsang0910-star/collab-cli](https://github.com/Staryinsang0910-star/collab-cli) |

### Skills & Community

| Resource | URL |
|----------|-----|
| awesome-reasonix (Hub) | [github.com/hikari-2424/awesome-reasonix](https://github.com/hikari-2424/awesome-reasonix) |
| Skills Site (Browse) | [hikari-2424.github.io/awesome-reasonix-site](https://hikari-2424.github.io/awesome-reasonix-site/) |
| Full Registry JSON | [github.com/.../registry.json](https://github.com/hikari-2424/awesome-reasonix/blob/main/registry.json) |

### Memory & Portability

| Resource | URL |
|----------|-----|
| Hindsight-Reasonix (Memory) | [github.com/houycth/Hindsight-Reasonix](https://github.com/houycth/Hindsight-Reasonix) |
| Reasonix PortaKit | [github.com/CS-Faith/reasonix-portakit](https://github.com/CS-Faith/reasonix-portakit) |

### Guides & Comparisons

| Resource | URL |
|----------|-----|
| Complete Guide (aimadetools) | [aimadetools.com/blog/reasonix-complete-guide](https://www.aimadetools.com/blog/reasonix-complete-guide) |
| Setup Guide | [aimadetools.com/blog/how-to-use-reasonix](https://www.aimadetools.com/blog/how-to-use-reasonix) |
| Reasonix vs Claude Code | [aimadetools.com/...](https://www.aimadetools.com/blog/reasonix-vs-claude-code) |
| Reasonix vs Aider | [aimadetools.com/...](https://www.aimadetools.com/blog/reasonix-vs-aider-for-deepseek) |

### Distribution

| Channel | Details |
|---------|---------|
| npm | `npm i -g reasonix` (pulls prebuilt Go binary) |
| Homebrew | `brew install esengine/reasonix/reasonix` |
| Prebuilt Archives | 6 targets: darwin/linux/windows × amd64/arm64 + SHA256SUMS |
| Debian | `.deb` package built on release |
| Source | `make build` (→ `bin/reasonix`) / `make cross` (→ `dist/`) |
| Windows Signing | [SignPath Foundation](https://signpath.io/) (free certificate) |

### Contributors & Acknowledgments

Top contributors (alphabetical, by commit count + code volume): **ctharvey**, **dimasd-angga**, **Evan-Pycraft**, **ForeverYoungPp**, **GTC2080**, **kabaka9527**, **lisniuse**, **wade19990814-hue**, **wviana**.

Logo by **Bernardxu123**. Community promotion on XiaoHongShu (小红书) by **AIGC Link**.

### Donations

Reasonix is MIT-licensed and free. Donations stay "a coffee, not a contract" — they don't buy feature priority or change issue triage. [PayPal: paypal.me/yuhuahui](https://paypal.me/yuhuahui) · WeChat Pay (QR in README).

---

> **Document**: Updated Jun 2026 from primary source `esengine/DeepSeek-Reasonix` (main-v2, v1.5.0, Jun 10 2026), community GitHub repos, DeepSeek API docs, npm, aimadetools, reasonix.homes.  
> **Note**: The "Reasonix" name also refers to [reasonixos.com](https://reasonixos.com/about) — an "Integrated Intelligence Company" (AI-powered enterprise decision systems / AI Growth OS for marketing). This document covers only the DeepSeek-native coding agent ecosystem, not the enterprise platform.
