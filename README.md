<p align="center">
  <img src="docs/logo-animated.svg" alt="Reasonix-Hermes" width="540"/>
</p>

<p align="center">
  <a href="./README.zh-CN.md">中文</a>
  &nbsp;·&nbsp;
  <a href="./docs/GUIDE.md">Guide</a>
  &nbsp;·&nbsp;
  <a href="./docs/SPEC.md">Spec</a>
  &nbsp;·&nbsp;
  <a href="./docs/PROJECT.md">Project</a>
</p>

> **Reasonix Hermes** is an extended fork of
> [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix)
> (synced to v1.11.x), the DeepSeek-native AI coding agent. We build on
> upstream's config-driven, plugin-driven Go core and add a Discord bot,
> MCP bridge server, Hindsight memory server, curated skill registry,
> native hook runner, and portable mode — everything an agent ecosystem
> needs to connect, remember, and collaborate.

<br/>

<h3 align="center">A DeepSeek-native AI coding agent — forked and extended for the community.</h3>
<p align="center">Statically-linked Go binaries. Config-driven. Plugin-driven. Tuned for DeepSeek's prefix cache so token costs stay low across long sessions.</p>

<br/>

## What Hermes adds

Hermes keeps upstream's core — the agent loop, providers, tools, permissions,
plugin system, desktop app — and layers on cross-agent connectivity and
persistent memory:

| Addition | What it does |
|----------|-------------|
| **Discord bot** | `/goal <objective>` autonomous loop, `/model flash\|pro\|mimo` per-session switching, slash commands for approval/deny/ask, multi-platform gateway |
| **MCP bridge server** | 6 tools (`reasonix_run`, `reasonix_doctor`, `plan_task`, `orchestrate_task`, `get_skill`, `get_skills`) — connect Claude Code, Codex, or any MCP client to Reasonix over stdio or HTTP |
| **Hindsight memory** | 3 tools (`hindsight_retain`, `hindsight_recall`, `hindsight_reflect`) — cross-session persistent memory with SQLite + file backends, TTL/importance decay, and TF-IDF vector search |
| **Skills hub** | 17 curated community skills (debugging, security audit, code review, refactoring, frontend builder, migration assistant, adversarial review…) — frontmatter playbooks with `runAs` and `allowedTools` |
| **Native hook runner** | Zero-dependency Go binary for PreToolUse/Stop hooks — replaces shell scripts, POSTs retain/reflect to the memory server |
| **Portable mode** | `REASONIX_PORTABLE=1` redirects all data (config, sessions, cache, memory, skills) to `<binary_dir>/.reasonix/` — run from a USB drive or air-gapped machine |

### Desktop Hermes enrichment

The desktop app (Wails + React 19) has been enriched with 17 Hermes-specific features:

| Feature | What it does |
|---------|-------------|
| **Hermes accent theme** | 7th theme style — caduceus gold (#d4a853) with warm dark surfaces and teal highlights |
| **Live data push** | Wails event loop replaces 5s frontend polling — all dashboard data pushed from Go |
| **Token sparkline chart** | Per-turn stacked bar chart (prompt + completion) with cache hit-rate and peak tokens |
| **Compaction timeline** | Timeline panel showing auto/manual compactions with trigger, message count, and summary |
| **Checkpoint file preview** | Expandable per-turn file list with pre-edit content preview (up to 500 chars) |
| **Write Mode** | Split-pane Markdown editor with file browser, live preview, Cmd+S save, new file creation |
| **Memory fact graph** | Facts clustered by type with color-coded badges (user/project/feedback/reference) |
| **Cache economy gauge** | Session-wide cache hit-rate badge |
| **Hindsight memory dashboard** | Facts, docs, and scopes from the auto-memory store |
| **Discord bot monitor** | Live online/offline status and session count in StatusBar |
| **Goal progress widget** | Active goal bar with turn/block status badges |
| **Skills hub browser** | 17 skills with search and category filter |
| **Sub-agent task tree** | Indented tree with status badges (running/done/failed) |
| **Constitution health check** | Rules viewer with JSON template from .reasonix/constitution.json |
| **StatusBar compact widgets** | Cache% gauge and Discord dot in the bottom bar |
| **Hotbar** | 7 keyboard digit shortcuts (Ctrl+1-7) for palette/workspace/new/history/dock/sidebar/settings |
| **Profiles** | Named profile switching — fast/cheap vs. deep reasoning with model/effort/mode overrides |

Full reference: **[Hermes Master Guide](./docs/HERMES-GUIDE.md)** — 19 sections covering all upstream + Hermes features.

### CLI TUI Hermes enrichment

The terminal chat UI has been enhanced:

- ⚚ **Pinned header** — REASONIX-HERMES branding visible above the transcript at all times
- **Bottom status counters** — turns, messages, goal progress, and memory facts always visible
- **`/stats` sparkline** — Unicode block character (▁▂▃▄▅▆▇█) token bar chart per turn
- **`/stats` compaction log** — each compaction pass with trigger, message count, and summary
- **`/stats` memory facts** — fact list with name and title
- **`/stats` goal progress** — status, turns, and blocks when a goal is active
- **`/write <file>`** — opens .md files in $EDITOR for Write Mode
- **`/publish` / `/cost`** — session transcript export (HTML/JSON) and cost summary

### Expansion packs

| Feature | What it does |
|---------|-------------|
| **Telegram bot adapter** | Long-polling Telegram adapter implementing `bot.Adapter` — DMs, groups, supergroups, media extraction, message splitting. Config at `[bot.telegram]`. |
| **LINE bot adapter** | Webhook-based LINE Messaging API adapter implementing `bot.Adapter` — handles text messages via reply API, paragraph-aware splitting. Runs local HTTP server for webhook events. Config at `[bot.line]`. |
| **Self-improving skill loops** | `internal/learn/` — observes turn patterns (edit→test, write→build), detects repeated sequences, builds reflection prompts for automated SKILL.md generation. Config at `[learn]`. |
| **Agent-to-agent MCP mesh** | `internal/mesh/` — peer delegation, broadcast, council mode (N peers → consensus synthesis). HTTP JSON-RPC transport with bearer auth. Config at `[mesh]` with `[[mesh.peers]]`. |
| **Live collaboration** | `internal/collab/` — WebSocket hub for real-time session sharing between Reasonix instances. Subscribe/broadcast/steer protocol. Config at `[collab]`. |
| **Schedule dashboard** | Cron-driven automated agent tasks with next-run timers and result ring (desktop widget). Config at `[[schedule.tasks]]`. |
| **Session publishing** | Export session transcripts as self-contained HTML (inline CSS, syntax highlighting) or JSON. Desktop widget with one-click download+clipboard. |
| **PR review action** | GitHub Action: on PR open/sync → fetches diff → runs Reasonix review with 6-dimension prompt (correctness, CI, constraints, security, trustworthiness, completeness) → posts comment. |
| **Helm chart + docker-compose** | One-command deploy to Kubernetes or single-node $5 VPS. `deploy/helm/reasonix/` (7-file chart) + `deploy/docker-compose.yml`. |

<br/>

## Upstream foundation

- **Config-driven.** Providers, the agent, enabled tools, and plugins are all
  declared in `reasonix.toml`. No hardcoded models.
- **Multi-model & composable.** DeepSeek ships as a preset; any
  OpenAI-compatible endpoint is a config entry, not new code. Optionally run
  two models together (executor + planner) in separate, cache-stable sessions.
- **Plugin-driven.** External tools run as subprocesses over stdio JSON-RPC
  (MCP-compatible). Built-in tools self-register at compile time.
- **Cache-aware context maintenance.** Startup injects a small stable environment
  summary, stale tool output is snipped/pruned before summary compaction, and the
  built-in tool schema contract is documented for regression review.
- **Zero-friction distribution.** `CGO_ENABLED=0` single binary; cross-compile
  to six targets with one command. The only dependency is a TOML parser.

See the [Guide](./docs/GUIDE.md), [Spec](./docs/SPEC.md), and [Hermes Guide](./docs/HERMES-GUIDE.md) for the full picture.

## Install

### One-line (npm)

```sh
npm i -g reasonix-hermes
```

This pulls the prebuilt binary for your platform and adds `reasonix-hermes`
(and `reasonix`) to your PATH.

### Build from source

```sh
git clone https://github.com/aliatx2017/reasonix-hermes.git
cd reasonix-hermes

# Core CLI
go build -o bin/reasonix ./cmd/reasonix

# Hermes services
go build -o bin/reasonix-mcpbridge  ./cmd/reasonix-mcpbridge   # MCP bridge (6 tools)
go build -o bin/reasonix-memoryserver     ./cmd/reasonix-memoryserver # Hindsight memory
go build -o bin/reasonix-bot        ./bot                       # Discord, Telegram, LINE, Slack bot
go build -o bin/reasonix-hooks      ./cmd/reasonix-hooks        # Hook runner
go build -o bin/reasonix-pr-review     ./cmd/reasonix-pr-review    # PR review CLI
go build -o bin/reasonix-e2ebench     ./cmd/e2ebench             # E2E benchmark tool
go build -o bin/reasonix-learner-live-test ./cmd/learner-live-test # Learner live e2e validation

# Desktop app (Wails + React 19)
cd desktop && wails build -o ../bin/reasonix-desktop
```

### Install the 17-skill community registry

```sh
./bin/reasonix install-source install \
  --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills
```

<br/>

## Quick start

```sh
reasonix setup                      # config wizard → ./reasonix.toml
export DEEPSEEK_API_KEY=sk-...      # or let setup save it to Reasonix home .env
reasonix chat                       # start a session

# Run a one-shot task
reasonix run "add unit tests for the auth module"

# Start the MCP bridge (expose Reasonix to other agents)
reasonix-mcpbridge --http --port 9090

# Start the memory server
reasonix-memoryserver --backend sqlite --http --port 8080

# Run the Discord/Telegram/LINE/Slack bot
export DISCORD_BOT_TOKEN="..."
reasonix-bot
reasonix                            # then run /init to generate AGENTS.md (project memory)
reasonix run "implement the TODOs in main.go"
reasonix run --model deepseek-pro "add unit tests for this function"
echo "explain this code" | reasonix run
```

> If you built from source, replace `reasonix` with `./bin/reasonix` above.

<br/>

## Configuration

A minimal `reasonix.toml` — one provider and a default model — is enough to start:

```toml
default_model = "deepseek-flash"

[[providers]]
name        = "deepseek-flash"
kind        = "openai"
base_url    = "https://api.deepseek.com"
model       = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
```

Resolution order is **flag > `./reasonix.toml` > `~/.config/reasonix/config.toml` >
built-in defaults**. Secrets come from the environment via `api_key_env` — never
written to config files.

### Memory server as an MCP plugin

```toml
[[plugins]]
name    = "hindsight"
command = "./bin/reasonix-memoryserver"
args    = ["--backend", "sqlite"]
```

### Discord bot config

```toml
[bot]
enabled = true
model   = "deepseek-flash"

[bot.allowlist]
enabled       = true
discord_users = ["123456789"]
discord_groups = ["987654321"]

[bot.discord]
token_env    = "DISCORD_BOT_TOKEN"
server_id    = "123456789"
allow_dms    = true
```

Full reference: **[Guide](./docs/GUIDE.md)** covers permissions, sandbox, plugins
(MCP), slash commands, `@` references, two-model collaboration, hooks, and memory.

<br/>
Resolution order is **flag > `./reasonix.toml` > the user config file >
built-in defaults**; starting with **Reasonix v1.11.0**, the user file lives at
`~/.reasonix/config.toml` on macOS/Linux and
`%AppData%\reasonix\config.toml` on Windows. See
**[Configuration paths](./docs/CONFIG_PATHS.md)** for migration details and the
full `config.toml` / `.env` structure. Provider entries name secrets with
`api_key_env`; the secret values themselves live in Reasonix's global
`<Reasonix home>/.env`, shared by CLI and desktop. Project `.env` files are not
provider-key runtime fallbacks, but still feed workspace-scoped, non-provider
`${VAR}` expansion for MCP/plugin settings without importing Reasonix control
variables. Permissions, the sandbox, plugins (MCP), slash
commands, `@` references, and two-model setup are all in the
**[Guide](./docs/GUIDE.md)**.

## Documentation

<<<<<<< HEAD
| Doc | What |
|-----|------|
| **[Guide](./docs/GUIDE.md)** | Configuration, permissions & sandbox, plugins (MCP), slash commands, two-model collaboration |
| **[Spec](./docs/SPEC.md)** | Engineering contract — architecture, registries, data types, and design principles |
| **[Hermes Guide](./docs/HERMES-GUIDE.md)** | Comprehensive Hermes feature guide — 19 sections covering all extensions |
| **[Project](./docs/PROJECT.md)** | Hermes fork architecture, commands, customizations, and contributor notes |
| **[Domain Model](./CONTEXT.md)** | Project glossary — canonical terms, `_Avoid_` alternatives, and domain boundaries |
| **[Desktop App](./docs/DESKTOP.md)** | Wails desktop app — Hermes dashboard, write mode, bot connections, live data |
| **[Bot Guide](./docs/BOT_GUIDE.md)** | Connect Discord, Telegram, LINE, Slack, Feishu, WeChat, QQ bots |
| **[Marketplace](./docs/MARKETPLACE.md)** | Community skill registry + LobeHub sync (850+ agents) |
| **[Skills Catalog](./docs/SKILLS-CATALOG.md)** | Complete inventory of all 59 skills — project, community, and global |
| **[Constitution](./docs/CONSTITUTION.md)** | Project invariants — principles, constraints, and code-level rules |
| **[Session Eval](./docs/EVAL.md)** | Compare two agent sessions for eval-driven development |
| **[Headroom Proxy](./docs/HEADROOM.md)** | LLM context optimization proxy — compression, caching, cost savings |
| **[Force English](./docs/HOWTO-FORCE-ENGLISH.md)** | Hard language enforcement — stop the model from switching to Chinese |
| **[Token Saving](./docs/HOWTO-TOKEN-SAVING.md)** | Step-by-step guide for grafting the sqz compressor into any Reasonix fork |
| **[Migrating from 0.x](./docs/MIGRATING.md)** | Moving from legacy TypeScript releases to the 1.0 Go rewrite |
| **[Checkpoints & rewind](./docs/CHECKPOINTS.md)** | Snapshot-based edit safety net (Esc-Esc / `/rewind`) |
| **[Changelog](./docs/CHANGELOG-HERMES.md)** | Hermes fork milestones, expansion packs, bot platforms |
| **[Releasing](./docs/RELEASING.md)** | Build, tag, and release process for all binaries |
| **[Session Memory Retrieval](./docs/SESSION_MEMORY_RETRIEVAL.md)** | Lightweight local history + BM25 retrieval |
| **[Session Reference Architecture](./docs/SESSION_REFERENCE_ARCHITECTURE.md)** | upstream-issue reference for session state routing |
| **[Ecosystem Reference](./reasonix-deepseek-ecosystem-2026.md)** | Full landscape: MCP bridges, skills, desktop, IDE, forks, protocols |
| **[Repo Evaluations](./docs/repo-evaluations-2026-06-20.md)** | 17-repo deep-dive audit — Adopt/Watch/Skip verdicts with rationale |
| **[Reasoning Language](./docs/REASONING_LANGUAGE.md)** | Force the model to reason in a specific language |
| **[Force English — Reddit case study](./docs/FORCE-ENGLISH-REDDIT.md)** | Real-world Reddit post showing why force-english matters |
| **[Codebase Audit (Historical)](./AUDIT.md)** | June 2025 audit — all bugs resolved, kept for reference |
| **[协作模式 (中文)](./docs/COLLABORATION_MODES.zh-CN.md)** | 计划模式、目标模式与省 token 模式 |
| **[桌面端 Hooks (中文)](./docs/DESKTOP_HOOKS.zh-CN.md)** | 桌面端 Hooks 使用说明 |
| **[工具权限模式 (中文)](./docs/TOOL_APPROVAL_MODES.zh-CN.md)** | 询问、自动与 Yolo 模式 |
| **[推理语言 (中文)](./docs/REASONING_LANGUAGE.zh-CN.md)** | 强制模型以指定语言进行推理 |
| **[Goal 执行 (中文)](./docs/GOAL_ENFORCEMENT.zh-CN.md)** | OMO 风格目标执行功能 |
| **[Architecture Decisions](./docs/adr/)** | Recorded architectural decisions — cache-first prefix, controller seam |
=======
- **[Guide](./docs/GUIDE.md)** — configuration, permissions & sandbox, plugins
  (MCP), slash commands, `@` references, two-model collaboration.
- **[Bot guide](./docs/BOT_GUIDE.md)** — connect Feishu, Lark, and WeChat bots
  from the desktop app, then use approvals, YOLO, and commands from IM.
- **[Spec](./docs/SPEC.md)** — engineering contract: architecture, registries,
  data types, and roadmap.
- **[Tool contract](./docs/TOOL_CONTRACT.md)** — provider-visible built-in tool
  names, read-only flags, and schema snapshot guard.
- **[Migrating from 0.x](./docs/MIGRATING.md)** — moving from the legacy
  TypeScript releases to the 1.0 Go rewrite.
- **[Checkpoints & rewind](./docs/CHECKPOINTS.md)** — the snapshot-based edit
  safety net (Esc-Esc / `/rewind`).
>>>>>>> upstream/main-v2

<br/>

## Relationship to upstream

Hermes tracks [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix)
(`main-v2` branch) as its upstream. We merge upstream releases into our `main`
branch and layer our additions on top:

```sh
git fetch upstream
git merge upstream/main-v2
```

Our custom code lives in `cmd/reasonix-*`, `internal/bot/`, `internal/learn/`,
`internal/mesh/`, `internal/collab/`, `internal/compress/`,
`internal/scheduler/`, `internal/publish/`, `pkg/`, `bot/`, `deploy/`,
`skills-hub/`, and the `desktop/hermes_dashboard.go` + React hermes components.
We do not modify the upstream engine (`internal/agent`, `internal/provider`,
`internal/tool`, `internal/plugin`, `internal/skill`, `internal/lsp`, etc.)
except for the shared `internal/bot/` gateway and `internal/control/` getters
used by our adapters and desktop dashboard.

For the full upstream feature set — desktop app (Wails + React 19), bot gateway
(Feishu/WeChat/QQ), ACP sessions, PDF extraction, themeable workspace — see the
[upstream README](https://github.com/esengine/deepseek-reasonix).

<br/>

## Acknowledgments

Hermes is built on [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) —
a community effort by [dozens of
contributors](https://github.com/esengine/DeepSeek-Reasonix/graphs/contributors).
All credit for the core engine, provider abstraction, tool system, permission
layer, plugin framework, desktop app, and the original vision belongs to the
upstream team.

<p align="center">
  <a href="https://github.com/esengine/DeepSeek-Reasonix/graphs/contributors">
    <img src="https://contrib.rocks/image?repo=esengine/DeepSeek-Reasonix&max=100&columns=12" alt="Contributors to esengine/DeepSeek-Reasonix" width="860"/>
  </a>
</p>

<br/>

---
<p align="center">
  <sub>MIT — see <a href="./LICENSE">LICENSE</a></sub>
  <br/>
  <sub>Hermes extras built by <a href="https://github.com/aliatx2017">aliatx2017</a> · upstream by <a href="https://github.com/esengine/DeepSeek-Reasonix">esengine/DeepSeek-Reasonix</a></sub>
</p>
