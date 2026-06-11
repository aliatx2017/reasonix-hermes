# Reasonix Hermes — New Research Findings (June 10, 2026)

> Supplement to `reasonix-deepseek-ecosystem-2026.md`. Documents findings from a
> deep-web sweep across GitHub, DeepSeek API docs, Reddit, and community
> repos on June 10, 2026.
>
> **Updated June 25, 2026**: We have since synced our fork to upstream v1.5.0
> (commit e5e8f02). The "behind" status noted below is now resolved. Our custom
> additions (MCP bridge, Hindsight memory, Discord bot, skills hub) have been
> rebased onto the v1.5.0 codebase.

---

## 1. Upstream Release: v1.5.0 (June 10, 2026) — **NOW SYNCED**

Our fork was behind at time of writing. We have since merged upstream v1.5.0
(2026-06-25, commit e5e8f02, branch `main-v2` → our `main`). Key upstream additions now in our fork:
- **Bot Gateway**: Native Feishu (飞书) / Weixin (微信) / QQ adapters in desktop
- **Goal Mode**: `/goal <objective>` autonomous session-scoped active goal loop with blocked-state audit (3-repeat detection)
- **Subagent Transcript Continuation**: Resume/continue past subagent transcripts (#3586)
- **`read_skill` tool**: Load inline skills in plan mode without execution (#3713)
- **PDF Attachment Extraction**: Extract text from PDFs in chat (#3618)
- **Themeable Workspace UI**: 4 theme variants (Graphite, Sandstone, Porcelain, Midnight) + font/scale (#3752)
- **Tool-approval Modes**: ask/auto/yolo in desktop UI with collaboration modes (normal/plan/goal)
- **Desktop `.deb` package**: Linux .deb builds on release (#3634)
- **Ctrl+Home/Ctrl+End**: Scroll transcript viewport (#3723)
- **Ctrl+Z suspend**: Clean TUI suspend (#3697)
- **Security**: CodeQL scan alerts resolved, Myers diff overflow fix (#3718), SSE stream replay safety (#3745)
- **React 19 + TypeScript 6**: Desktop frontend fully on latest
- **Split `/new` and `/clear` session flows** (#3780)
- **Ctrl+J background jobs panel, Ctrl+K command palette** in desktop
- **`!` prefix shell execution** from composer (#3186)
- **ACP (Agent Communication Protocol)** native sessions (#3663)

### 1.1 Our Post-Sync Custom Additions (as of June 25, 2026)

These are our Hermes-specific additions layered on top of v1.5.0:

| Component | Status | Description |
|-----------|--------|-------------|
| `pkg/mcpbridge/` | ✅ Complete | MCP bridge server: 6 tools (run, doctor, plan, orchestrate, get_skill, get_skills), stdio+HTTP modes |
| `pkg/memoryserver/` | ✅ Complete | Hindsight memory: 3 tools (retain, recall, reflect), SQLite + file backends, TTL/importance scoring, Bearer auth |
| `pkg/httputil/` | ✅ Complete | Shared auth middleware — consolidated Bearer auth from mcpbridge+memoryserver |
| `pkg/mcputil/` | ✅ Complete | Shared MCP types and server helpers |
| `bot/` + `internal/bot/discord/` | ✅ Complete | Discord bot gateway with /goal autonomous loop (upstream has Feishu/WeChat/QQ only) |
| `skills-hub/` | ✅ Complete | 17 curated community skills with registry (incl. adversarial-review) |
| `cmd/reasonix-hooks/` | ✅ Complete | Native Go hook runner — zero-dependency binary replacing shell scripts |
| Hook scripts | ✅ Hardened | `retain-hook.sh` / `reflect-hook.sh` — error handling, timeout, python3/curl checks |
| Portable mode | ✅ Complete | `REASONIX_PORTABLE=1` — all data next to binary, USB/sync-drive friendly |
| Tests | ✅ 80%+ | mcpbridge 82%, memoryserver 89%, discord 91% |

---

## 2. New MCP Bridges Discovered

### 2.1 kquuen/reasonix-mcp-server — Full Orchestration Layer
**Repo**: `github.com/kquuen/reasonix-mcp-server` | ⭐1 | TypeScript/Node.js | MIT

The most sophisticated single Reasonix MCP bridge. Unlike Belveth02's simple delegation, this implements:
- **16 tools**: `reasonix_task`, `reasonix_review`, `reasonix_subtask`, 13 file manipulation tools
- **Adversarial Review Contract**: Structured `BLOCK:`/`ALLOW:` output format. 5 attack surfaces: security, correctness, performance, maintainability, test coverage
- **Atomic state persistence**: Crash-resilient mid-task recovery
- **Zero npm dependencies**: Pure Node.js stdlib
- **3 execution modes**: task (full execution), review (adversarial), subtask (isolated sub-agent)

### 2.2 enderzcx/codex-reasonix-bridge — Codex→Reasonix Review
**Repo**: `github.com/enderzcx/codex-reasonix-bridge` | ⭐3 | JavaScript (CLI) | MIT

Focused bridge for Codex → Reasonix engineering review, modeled on openai/codex-plugin-cc.
- **7 review modes**: consult, engineering-feedback, engineering-plan, daily-review, final-review, adversarial-review, general
- **Auto git-context collection**: Like codex-plugin-cc, collects status/diff/untracked
- **Background jobs**: `crb review --background --json` for non-blocking reviews
- **Input isolation**: Reasonix reviewer cannot read Codex workspace
- **Structured JSON output**: verdict, findings[] (severity/confidence/recommendation), next_steps[]
- **Companion repo**: `codex-mimo-skill` for MiMo UI/文案 tasks
- **Explicit upstream PR strategy**: Documents which fixes belong in Reasonix core vs. stay external

### 2.3 picodozbotdoz/reasonix-mcp-bridge — Reasonix↔Reasonix
**Repo**: `github.com/picodozbotdoz/reasonix-mcp-bridge` | ⭐0 | JavaScript | MIT

Minimal same-machine peer-to-peer messaging via `/tmp/reasonix-bridge/state.json`. Tools: `send_message`, `check_inbox`, `reply_to_message`, `get_response`, `list_peers`. Demonstrates autonomous agent-to-agent communication pattern.

### 2.4 MYHMZ20/feishu-reasonix-bridge — Feishu/Lark→Reasonix
**Repo**: `github.com/MYHMZ20/feishu-reasonix-bridge` | ⭐2 | TypeScript | MIT | Windows

Bridges Feishu/Lark messenger with Reasonix CLI. Based on zarazhangrui/feishu-claude-code-bridge. Streaming Lark cards, per-chat sessions, multi-agent routing (Claude/Reasonix), workspace switching, access control. Requires patched Reasonix.

---

## 3. Community Skills Ecosystem — New Discoveries

### 3.1 reasonix-skill-powers (⭐45 — MOST STARRED)
**Repo**: `github.com/liu5540/reasonix-skill-powers`

Superpowers 7-step workflow adapted for Reasonix sub-agents: brainstorming→explore→decompose→plan→implement(TDD+subagent)→verify+debug→commit. YAML frontmatter with `runAs: subagent` + `allowed-tools` whitelist.

### 3.2 superpowers-reasonix (⭐10)
**Repo**: `github.com/2eho/superpowers-reasonix`

9 skills from obra/superpowers: superpowers-workflow, brainstorming, writing-plans, executing-plans, tdd, code-review, verification, debugging, finish-branch.

### 3.3 Deepseek-Reasonix-Autopilot (⭐2, v2.0)
**Repo**: `github.com/791994545/Deepseek-Reasonix-Autopilot`

**113 skills** — the largest known collection. Full autonomous pipeline: complexity assessment→skill assembly→strategy selection→code→test→fix→record. Watchdog process, adaptive routing weights, self-learning error pattern accumulation.

### 3.4 Efficient-Reasonix (⭐3)
**Repo**: `github.com/meisijiya/Efficient-Reasonix`

One-stop configuration pack. MCP configs (Docker/PostgreSQL/MySQL/Redis), MiniMax multimodal (text/image/video/speech/music/vision), Agent rules with Karpathy coding standards + Superpowers workflow, 11 skills, Open Design integration.

---

## 4. Domain-Specialized Applications

### 4.1 fish-ecology-assistant (⭐1, v6.5.0)
**Repo**: `github.com/fangtaocai041/fish-ecology-assistant`

Turns Reasonix into a PhD-level fish ecology research team:
- 21 MCP services, 28 AI skills, 12 search engines, 13 knowledge bases
- R 4.6.0 with 20+ ecology packages, PaddleOCR, Zotero SQL
- 5-stage auto-orchestrated research pipeline
- Cross-project co-evolution: fish + cognitive-search-engine + eon-core triangle
- Dual-core philosophy: Panta Rhei (dynamic worldview) + Systems Thinking (7 engineering principles)
- Dockerized deployment

### 4.2 Bug-Report-Skill (BRS v3.0, ⭐3)
**Repo**: `github.com/Kepsilent/Bug-Report-Skill`

Cross-platform crash monitoring + AI diagnosis. MCP server for agent log queries. Works with Reasonix, Claude Code, Cursor, Codex, Windsurf, Cline, Aider. Android/iOS/微信/uni-app/React Native/Electron. Breadcrumb tracking, state snapshots, privacy sanitization.

---

## 5. Alternative Desktop Clients & IDE Extensions

### 5.1 Desktop Clients

| Client | ⭐ | Stack |
|--------|-----|-------|
| **reasonix-desktop** (suply) | 8 | Electron 38 + Vue 3 + Element Plus + Pinia |
| **DeepSeek_Reasonix_GUI** (0Sun-shine0) | 5 | Electron 35 + React 19 + TS 5 + Vite 5 |
| **BEing-Reasonix** (Tiannan666) | 13 | Electron launcher, Windows installer + wizard |
| **xusu-ai/reasonix** (xusu-ai) | 7 | Pure Node.js, single-page WebUI, zero GPU |

### 5.2 IDE Extensions

| Extension | Platform | ⭐ | Key Feature |
|-----------|----------|-----|-------------|
| **vscode-deepseek-reasonix** (pcbtool) | VS Code Marketplace | **36** | Auto `@file#L10-L20` injection, smart terminal reuse |
| **deepseek-reasonix-vscode** (whishi47) | VS Code/Windsurf/Trae | 1 | 3 keyboard shortcuts, 2.5s readiness delay |
| **rx-gui** (68110923) | JetBrains (IntelliJ/PyCharm/WebStorm) | 5 | ACP protocol, streaming chat+Markdown+Diff review, cost display |
| **deepseek-reasonix-intellij** (hxr66666) | IntelliJ IDEA | 0 | Early-stage Java/Kotlin plugin |

---

## 6. Notable Forks

| Fork | ⭐ | Distinction |
|------|-----|-------------|
| **roach-code** (tmdgusya) | **34** | Multi-model: Codex/OpenAI (Responses API+ChatGPT OAuth), MiniMax, GLM, Anthropic. v1.3.5. |
| **agent-usage-stats** (huyiling1111) | 1 | Local token consumption tracker for Claude Code, CodeX, Hermes, OpenClaw, Reasonix, DeepSeek TUI. |

---

## 7. Undocumented Features & Hacks (Newly Cataloged)

| Feature | How | Source |
|---------|-----|--------|
| `!` prefix shell execution | Type `!ls -la` in composer → runs shell directly | v1.2.0+ / #3186 |
| `dsnix` alias | `dsnix chat` = `reasonix chat` | Built-in |
| `reasonix events` | Full event stream replay with cost/cache analytics | Built-in CLI |
| `reasonix stats` | Aggregate session statistics | Built-in CLI |
| `reasonix replay` | Replay session transcript | Built-in CLI |
| Effort levels | `low/medium/high/max` control reasoning depth per turn | `/effort` |
| Model keyword routing | `preset=auto` routes "architecture"/"refactor"/"security" → pro | Config |
| `.mcp.json` drop-in | Claude Code's exact `mcpServers` schema works unchanged | Built-in |
| `#<note>` quick-add | Type `#<note>` in chat → adds line to REASONIX.md | Built-in |
| Esc-Esc rewind | Empty composer + double-Esc → checkpoint undo picker | v1.2.0+ |
| Checkpoint forking | `/branch <turn> [name]` from any checkpointed turn | v1.2.0+ |

---

## 8. Cost Optimization — Production Patterns

| Pattern | Savings | Mechanism |
|---------|---------|-----------|
| Two-model split | ~5× | flash executor + pro/mimo planner (separate sessions) |
| Hybrid orchestrator | 5-10× | Claude Code/Codex plans → Reasonix executes via MCP |
| Stateless sub-tasks | ~99% | OpenCode bridge: $0.00004/task via prefix cache reuse |
| Effort levels | 2-10× | `low` for simple fixes, `max` only for architecture |
| `reasonix run` for CI/CD | sessionless | No context accumulation |
| PortaKit multi-machine | zero re-warm | Portable config preserves cache-warm prefixes |

**Headline**: 435M input tokens → 99.82% cache hits → **$12** (vs $61 without cache).

---

## 9. Key Patterns Emerging in the Ecosystem

1. **Agent Mesh Topology**: collab-cli and reasonix-bridge enable Reasonix instances to communicate directly, not just through a user
2. **Adversarial Review Contracts**: Structured `BLOCK:`/`ALLOW:` outputs from review subagents with formalized quality gates
3. **Domain-Specialized Agent Teams**: fish-ecology-assistant proves Reasonix can become a PhD-level domain team through skills+MCP+handbooks alone — zero code changes to core
4. **WebUI/Zero-Install Movement**: xusu-ai/reasonix and PortaKit push toward zero-friction deployment
5. **IDE Embedding**: VS Code extensions (⭐36) and JetBrains plugin (⭐5) make Reasonix feel IDE-native
6. **Hybrid Model Routing (Dominant Pattern)**: Frontier model plans + DeepSeek executes, bridged via MCP — 1/5 to 1/10 the cost
7. **Portability**: PortaKit solves the persistent pain of multi-machine setups with auto path-fixing
