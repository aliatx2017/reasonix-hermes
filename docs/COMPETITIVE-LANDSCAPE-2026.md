# Reasonix Hermes — Competitive Landscape Analysis

> **Date:** June 13, 2026
> **Scope:** Comprehensive analysis of the AI coding agent harness landscape, comparing Reasonix Hermes against upstream DeepSeek-Reasonix, Nous Hermes Agent, OpenClaw, Pi, oh-my-pi, Claude Code, Orca, OpenHands, Plandex, roach-code, and others.
> **Methodology:** Web research of GitHub repos, documentation sites, and community ecosystem; deep codebase exploration of Reasonix Hermes to catalog every custom component.

---

## 1. The Landscape — Major Projects by Platform

The AI coding agent space has exploded. Here are the major projects (June 2026), categorized by their core approach.

### 1.1 Single-Binary Harnesses (Go)

| Project | Stars | Binary Size | Model Focus | Unique Angle |
|---------|-------|-------------|-------------|--------------|
| **DeepSeek-Reasonix** (upstream) | 21.8k | ~26MB | DeepSeek (+ OpenAI compat) | Prefix-cache stability, CGO-free, TUI + Desktop + IM bots |
| **Plandex** | 15.5k | ~20MB | Any (via OpenRouter) | 2M token context, cumulative diff sandbox, autonomous debugging |
| **Reasonix Hermes (us)** | — | ~27MB | DeepSeek (+ OpenAI compat) | MCP bridge server + memory server + Discord bot + Constitution |
| **roach-code** | 34 | ~27MB | DeepSeek + Codex + MiniMax + GLM | Multi-model Reasonix fork (adds providers, TUI theme) |

### 1.2 TypeScript-Native Harnesses

| Project | Stars | Runtime | Unique Angle |
|---------|-------|---------|-------------|
| **OpenClaw** | 379k | Node 24 | 20+ IM channels, voice, live canvas, companion apps |
| **Pi** (earendil-works) | 62.3k | Node/npm | Clean monorepo (4 packages), multi-provider, no permission system |
| **oh-my-pi** | 12.2k | Node/npm | Hash-anchored edits, LSP, browser automation, subagents |
| **Codegraph** | 48.5k | Node/npm | Pre-indexed knowledge graph for all agents |
| **Cherry Studio** | 47.3k | Electron | 300+ AI assistants, multi-provider chat |

### 1.3 Python Harnesses

| Project | Stars | Runtime | Unique Angle |
|---------|-------|---------|-------------|
| **Nous Hermes Agent** | 193k | Python 3.11+ | Self-improving learning loop, 200+ models, six deployment backends |
| **OpenHands** | 76.8k | Python | Full stack: SDK + CLI + GUI + Cloud + Enterprise |
| **PraisonAI** | 8.1k | Python | Multi-agent framework, MCP, RAG, guardrails, 5 lines to deploy |

### 1.4 Commercial / Proprietary

| Project | Pricing | Model Lock-In | Unique Angle |
|---------|---------|---------------|-------------|
| **Claude Code** | $10–100/mo + API | Claude-only (3rd-party in CLI) | Most polished UX, all surfaces (Terminal/IDE/Desktop/Web/iOS/Slack) |
| **GitHub Copilot** | $0–100/mo | Multi-model (via agent marketplace) | Deep IDE integration, AI credits economy |
| **Orca** (stablyai) | Free (MIT) | Any CLI agent | ADE: fleets of parallel agents in worktrees + mobile companion |

### 1.5 Desktop Hubs (Aggregators)

| Project | Stars | Agents Supported | Stack |
|---------|-------|-----------------|-------|
| **CC Switch** | 100k | 7 (Claude Code, Codex, Gemini CLI, OpenClaw, Hermes Agent, etc.) | Tauri + Rust + React |
| **AionUi** | 28.2k | 20+ | Electron + Vite + React |

---

## 2. Detailed Comparison With Named Competitors

### 2.1 vs. Upstream DeepSeek-Reasonix (esengine, 21.8k ⭐)

**Shared foundation:** Go single binary, prefix-cache stability, config-driven TOML, MCP client, bubbletea TUI, Wails desktop, `control.Controller` architecture, subagents, checkpoints, permission deny/ask/allow, sandbox (bubblewrap/Seatbelt), skills system, i18n (en/zh/zh-TW), slash commands, session persistence, context compaction.

**What Reasonix Hermes adds (18 distinct capabilities):**

| Capability | Upstream | Reasonix Hermes | Files |
|---|---|---|---|
| MCP Bridge Server (6 tools) | ❌ | ✅ | `cmd/reasonix-mcpbridge/main.go` |
| Hindsight Memory Server (3 tools) | ❌ | ✅ | `cmd/reasonix-memoryserver/main.go` |
| Discord Bot (real agent loop) | ❌ (Feishu/WeChat/QQ only) | ✅ | `bot/main.go`, `internal/bot/discord/` |
| Constitution System | ❌ | ✅ | `internal/constitution/constitution.go` |
| Remote Sandbox API (OpenSandbox) | ❌ | ✅ | `internal/sandbox/remote.go` |
| Native Go Hook Runner | ❌ (bash scripts only) | ✅ | `cmd/reasonix-hooks/main.go` |
| Skills Hub (17 curated playbooks) | ❌ | ✅ | `skills-hub/skills/*.md`, `skills-hub/registry.json` |
| Desktop Hermes Dashboard (12 widgets) | ❌ | ✅ | `desktop/hermes_dashboard.go`, `desktop/hermes_tier3.go` |
| Write Mode (CodeMirror 6 + FIM) | ❌ | ✅ | `desktop/frontend/src/components/hermes/` |
| Checkpoint Diff Viewer (Myers unified) | ❌ | ✅ | `internal/diff/` |
| Memory D3 Force Graph | ❌ | ✅ | `desktop/frontend/src/components/hermes/MemoryFactGraph.tsx` |
| Token Sparkline (ring buffer) | ❌ | ✅ | `desktop/frontend/src/components/hermes/TokenBreakdownChart.tsx` |
| Gold Hermes Theme | ❌ | ✅ | `desktop/frontend/src/` |
| Tray Icon (gold overlay) | ❌ | ✅ | `desktop/tray_icon_gold.go` |
| Nix Flake + Docker | ❌ | ✅ | `flake.nix`, `Dockerfile` |
| Automated Upstream Sync (daily) | ❌ | ✅ | `.github/workflows/sync-upstream.yml` |
| CI for Custom Binaries | ❌ | ✅ | `.github/workflows/ci-hermes.yml` |
| Install Source Manifest | ❌ | ✅ | `reasonix-hermes.json` |

**Key differentiators from upstream:**
- Upstream targets Feishu/WeChat/QQ bot users (Chinese market). We target Discord + global English-speaking developers.
- Upstream has no MCP server capability. We are both MCP client *and* server.
- Upstream has no persistent memory server. Our Hindsight server provides SQLite-backed, TF-IDF vector search memory.
- Upstream has no constitution/system of project rules. Our Constitution system auto-injects project rules into the system prompt.
- Upstream has no Nix/Docker/automated CI for custom builds. We ship infrastructure-as-code.

### 2.2 vs. Hermes Agent (NousResearch, 193k ⭐)

**Important:** Despite the shared "Hermes" name, these are **completely independent projects with zero shared code.** Nous Hermes Agent is a Python research project from Nous Research; Reasonix Hermes is a Go fork of DeepSeek-Reasonix.

| Dimension | Nous Hermes Agent | Reasonix Hermes |
|---|---|---|
| **Language** | Python 82%, TypeScript 14% | Go 99% |
| **Distribution** | pip/curl.sh (~50MB+ with deps) | Single 27MB static binary |
| **Learning Loop** | ✅ Autonomous skill creation from experience | ❌ (memory is retrieval, not learning) |
| **Multi-Platform Gateway** | ✅ Telegram, Discord, Slack, WhatsApp, Signal, Email, CLI, TUI | ✅ Discord only |
| **Model Support** | 200+ via OpenRouter + custom endpoints | DeepSeek + OpenAI-compatible |
| **Voice/TTS** | ✅ TTS with wake word | ❌ |
| **Self-Improvement** | ✅ Batch trajectory gen + compression for model training | ❌ |
| **Deployment** | Python 3.11+, uv, pip deps | Single binary, zero deps |
| **Prefix Cache** | ❌ (not designed for DeepSeek) | ✅ Core architecture principle |
| **MCP Server** | ❌ (client only) | ✅ Both client + server |
| **Memory** | ✅ FTS5 session search + LLM summarization | ✅ SQLite + TF-IDF vector |
| **Skills Hub** | ✅ agentskills.io standard | ✅ 17 curated custom skills |
| **Open Source** | ✅ MIT | ✅ MIT |
| **Stars** | 193,000 | — |

**Strategic takeaway:** Nous Hermes Agent dominates the Python agent space with massive investment from Nous Research. We cannot compete on scope or community. Our niche is **Go-native performance, DeepSeek prefix-cache optimization, and cross-agent interoperability via MCP bridge.** Nous Hermes is a self-contained ecosystem; we are a bridge between ecosystems.

### 2.3 vs. OpenClaw (379k ⭐)

| Dimension | OpenClaw | Reasonix Hermes |
|---|---|---|
| **Language** | TypeScript 91% | Go |
| **Multi-Channel** | 20+ (WhatsApp, iMessage, Signal, IRC, Telegram, Discord, etc.) | Discord only |
| **Voice/Talk Mode** | ✅ macOS wake word, continuous Android voice | ❌ |
| **Live Canvas (A2UI)** | ✅ Agent-driven visual workspace | ❌ |
| **Companion Apps** | macOS menu bar, Windows Hub, iOS/Android | Wails Desktop (mac/Win/Linux) |
| **Agent Routing** | Multi-agent per channel/peer | Single agent loop |
| **Scheduled Tasks** | ✅ Cron-integrated | ❌ |
| **Memory** | ✅ SOUL.md + Honcho dialectic | ✅ Hindsight (SQLite + TF-IDF) |
| **Sandbox** | ✅ Docker/SSH/OpenShell for non-main sessions | ✅ bubblewrap/Seatbelt + remote OpenSandbox |
| **Security** | DM pairing default | deny/ask/allow + constitution |
| **DeepSeek Prefix Cache** | ❌ | ✅ Core strength |
| **MCP Bridge** | ❌ | ✅ |
| **Deployment** | Node 24 runtime needed | Single static binary |
| **Stars** | 379,000 | — |

**Strategic takeaway:** OpenClaw is the undisputed king of multi-channel AI assistants — 20+ communication channels, voice wake, live canvas, four companion apps. It is a lifestyle AI assistant first, coding agent second. OpenClaw also cannot leverage DeepSeek's prefix cache at all. Our Discord bot is a tiny fraction of OpenClaw's IM reach, but our DeepSeek optimization and MCP bridge capabilities are unique.

### 2.4 vs. Pi (earendil-works, 62.3k ⭐) + oh-my-pi (12.2k ⭐)

| Dimension | Pi | oh-my-pi | Reasonix Hermes |
|---|---|---|---|
| **Language** | TypeScript 94% | TypeScript + Rust | Go |
| **Architecture** | 4-package monorepo | Single package | Single binary |
| **Multi-Provider** | ✅ OpenAI/Anthropic/Google unified API | Same as Pi | DeepSeek + OpenAI-compatible |
| **LSP Integration** | ❌ | ✅ | ❌ |
| **Hash-Anchored Edits** | ❌ | ✅ (core feature) | ❌ |
| **Browser Automation** | ❌ | ✅ | ❌ |
| **Permission System** | ❌ (explicitly documented: none) | Same as Pi | ✅ deny/ask/allow + sandbox + constitution |
| **Prefix Cache** | ❌ | ❌ | ✅ Core architecture |
| **Distribution** | npm + pnpm | npm | Single Go binary |
| **MCP Client** | ❌ | ❌ | ✅ MCP client + server |
| **Desktop** | ❌ | ❌ | ✅ Wails + React 19 |
| **Session Publishing** | ✅ pi-share-hf | ❌ | ❌ |
| **Stars** | 62,300 | 12,200 | — |

**Strategic takeaway:** Pi is a clean TypeScript monorepo with excellent multi-provider support but **explicitly no permission system** ("Pi does not include a built-in permission system for restricting filesystem, process, network, or credential access"). This is a deliberate design choice — and a security risk. Our permission system, sandbox, and constitution are significant advantages for production/enterprise use. oh-my-pi adds hash-anchored edits and LSP — features we could adopt.

### 2.5 vs. Claude Code (Anthropic, proprietary)

| Dimension | Claude Code | Reasonix Hermes |
|---|---|---|
| **Model Support** | Claude only (3rd-party in CLI only for non-core) | DeepSeek + any OpenAI-compatible |
| **Pricing** | $10–100/mo subscription + API costs | Free + your API key |
| **Platforms** | Terminal, VS Code, JetBrains, Desktop, Web, iOS, Slack | Terminal, Wails Desktop, Discord |
| **Agent SDK** | ✅ Full TypeScript/Python SDK | ❌ |
| **Agent Teams** | ✅ Lead + sub-agents (parallel) | ✅ Sub-agents (`task`) |
| **Skills** | ✅ `/command` playbooks | ✅ Skills hub (17 playbooks) |
| **MCP** | ✅ Universal MCP client | ✅ MCP client + MCP bridge server |
| **Hooks** | ✅ Shell hooks (pre/post) | ✅ Go native + bash hooks |
| **Prompt Caching** | ✅ Anthropic-specific | ✅ DeepSeek prefix cache |
| **Permissions** | ✅ Workspace confinement + ask/allow | ✅ deny/ask/allow + sandbox + constitution |
| **Sandbox** | ✅ Docker/bubblewrap/Seatbelt | ✅ bubblewrap/Seatbelt + remote OpenSandbox |
| **Open Source** | ❌ (partially open SDK) | ✅ MIT, full source |
| **Self-Hosted Models** | ❌ (Cloud only, except Bedrock) | ✅ any OpenAI-compatible + Ollama |
| **Mobile App** | ✅ iOS | ❌ |

**Strategic takeaway:** Claude Code is the most polished, commercially-backed agent, but it is **Claude-only** (for the core agent loop), **not open source**, and **costs money monthly PLUS API costs.** Our unique value: free, open-source, DeepSeek-optimized, can route to any OpenAI-compatible endpoint, and can act as an MCP bridge so that other agents (including Claude Code) can delegate work to us. **We are not competing with Claude Code — we are the open-source, multi-model alternative that interoperates with it.**

### 2.6 vs. Orca (stablyai, 4.7k ⭐)

| Dimension | Orca | Reasonix Hermes |
|---|---|---|
| **Core Value** | Run N CLI agents in parallel worktrees | Run one optimized agent + bridge to others |
| **Parallel Agents** | ✅ 5+ side-by-side, compare results | ❌ (single agent, sub-agents within session) |
| **Worktrees** | ✅ Git worktree per agent | ❌ |
| **Mobile Companion** | ✅ iOS + Android | ❌ |
| **Design Mode** | ✅ Click UI → inspect element → agent prompt | ❌ |
| **SSH Worktrees** | ✅ Remote execution with auto-reconnect | ❌ (we have SSH sandbox via remote API) |
| **Diff Annotation** | ✅ Comment on diff lines → back to agent | ❌ |
| **MCP Bridge** | ❌ | ✅ |
| **Prefix Cache** | ❌ | ✅ |
| **Open Source** | ✅ MIT | ✅ MIT |
| **Stars** | 4,700 | — |

**Strategic takeaway:** Orca is a completely different category — an "ADE" (Agent Development Environment) that orchestrates other agents. It is not an agent harness itself; it's a meta-orchestrator. Our MCP bridge could complement Orca: Orca runs Reasonix as one of its agents, and Reasonix runs the actual coding task with DeepSeek prefix-cache efficiency.

---

## 3. Landscape Summary Matrix

| Feature | Us | Upstream | Hermes Agent | OpenClaw | Pi | Claude Code | Orca |
|---|---|---|---|---|---|---|---|
| **Go single binary** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **DeepSeek prefix cache** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **MCP bridge server** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **MCP client** | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| **Persistent memory server** | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ | ❌ |
| **Discord bot** | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Multi-IM (20+ channels)** | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Constitution system** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Local sandbox** | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| **Remote sandbox** | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Desktop app** | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ |
| **Cron / scheduled tasks** | ❌ | ❌ | ✅ | ✅ | ❌ | ✅ | ❌ |
| **LSP integration** | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| **Browser automation** | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| **Voice / TTS** | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Mobile companion** | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ | ✅ |
| **Parallel worktrees** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Session publishing** | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| **Nix / Docker infra** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Automated upstream sync** | ✅ | N/A | N/A | N/A | N/A | N/A | N/A |
| **Zero permission system** | ❌ (we have one) | ❌ | ❌ | ❌ | ✅ (bad) | ❌ | ❌ |

---

## 4. Our Unique Position

```
                    ┌──────────────────────────────────┐
                    │     Commercial / Cloud Agents     │
                    │  Claude Code ($) | Copilot ($)    │
                    └────────────┬─────────────────────┘
                                 │
              Self-Contained     │     Interoperability
              Ecosystems         │     Bridge Layer
    ┌──────────────────┐         │    ┌─────────────────────────┐
    │  Hermes Agent    │         │    │   ★ REASONIX HERMES ★   │
    │  (193k ⭐ Python) │    ────┤    │                         │
    │  Self-improving  │         │    │  Go-native 27MB binary  │
    │  Multi-platform  │         │    │  DeepSeek prefix cache  │
    └──────────────────┘         │    │  MCP BRIDGE SERVER      │
                                 │    │    → Reasonix tools     │
    ┌──────────────────┐         │    │    → to any MCP client  │
    │  OpenClaw        │         │    │  Memory server (TF-IDF) │
    │  (379k ⭐ TS)    │    ────┤    │  Discord bot            │
    │  20+ channels    │         │    │  Constitution system    │
    │  Voice + Canvas  │         │    │  Permissions + sandbox  │
    └──────────────────┘         │    │  Nix + Docker infra     │
                                 │    │  Skills hub (17)       │
    ┌──────────────────┐         │    └─────────────────────────┘
    │  Pi / oh-my-pi   │         │
    │  (62k/12k ⭐ TS) │    ────┤
    │  No permissions  │         │    ┌─────────────────────────┐
    │  LSP + Browser   │         │    │  Desktop Hubs           │
    └──────────────────┘         │    │  CC Switch (100k)       │
                                 │    │  AionUi (28k)           │
    ┌──────────────────┐         │    │  Orca (4.7k)            │
    │  Plandex         │         │    └─────────────────────────┘
    │  (15.5k ⭐ Go)   │    ────┤
    │  2M token ctx    │         │
    └──────────────────┘         │
                                 │
    ┌──────────────────┐         │
    │  OpenHands       │         │
    │  (76.8k ⭐ Py)   │    ────┘
    │  SDK+CLI+GUI     │
    └──────────────────┘
```

**Our position is unique because:**

1. **Only Reasonix fork with MCP bridge + memory server + Discord bot.** roach-code (34 ⭐) is the only other Reasonix fork; it adds multi-provider support but none of our infrastructure.

2. **Only Go-native coding agent that is also an MCP server.** We export Reasonix as a tool provider that any MCP-compatible client can call. Claude Code can be our client; we can be theirs.

3. **Only DeepSeek-optimized agent with a Discord bot.** Upstream only targets Feishu/WeChat/QQ (Chinese IM platforms). We target English-speaking Discord communities.

4. **Only agent with a structured project governance system.** The Constitution system auto-injects `.reasonix/constitution.json` rules into the system prompt — no other agent harness has this.

5. **Only agent with automated upstream sync.** Our daily CI pipeline merges upstream changes, keeping us current while preserving our customizations. This is a sustainable fork maintenance strategy that no other fork provides.

6. **Smallest deployment footprint.** At 27MB single static binary with zero dependencies, we are dramatically smaller than any Python agent (50MB+ with runtime) or TypeScript agent (requires Node 24+).

7. **Bridge, not island.** Every other agent is a self-contained ecosystem. We explicitly designed Reasonix Hermes to interoperate — MCP bridge, Hindsight memory server, Hermes dashboard all export functionality to outside tools.

---

## 5. Feature Gap Analysis — What Could We Add?

Ranked by feasibility (effort estimate) × differentiation (uniqueness).

### Tier 1 — High Feasibility, High Differentiation (build next)

| # | Feature | Inspired By | Estimated Effort | Why It Matters |
|---|---------|-------------|------------------|----------------|
| 1 | **Multi-Provider Expansion** (Codex/MiniMax/GLM) | roach-code | ~500 lines per provider | Makes us the *complete* Reasonix fork — all models + all our infra |
| 2 | **Scheduled / Cron Tasks** | Hermes Agent, OpenClaw | ~300 lines in `internal/schedule/` | Turns agent from interactive to autonomous "nightly review" mode |
| 3 | **Hash-Anchored Edits** | oh-my-pi | ~150 lines in `internal/tool/edit.go` | Safety net against concurrent file modifications |
| 4 | **Cross-Session Learning** | Hermes Agent | ~200 lines in `internal/memory/learning.go` | Auto-distill patterns into skills after complex tasks |

### Tier 2 — Medium Feasibility, High Impact

| # | Feature | Inspired By | Estimated Effort | Why It Matters |
|---|---------|-------------|------------------|----------------|
| 5 | **LSP Integration** | oh-my-pi | ~600 lines in `internal/lsp/` | Makes agent IDE-aware — "fix all diagnostics" as one command |
| 6 | **Browser Automation** | oh-my-pi | ~800 lines + chromedp | Login flows, web app testing, form filling beyond static fetch |
| 7 | **Session Publishing** | Pi's pi-share-hf | ~300 lines in new binary | Transparency + debugging for the community |
| 8 | **Voice TTS Notifications** | OpenClaw, Hermes Agent | ~200 lines in `internal/notify/voice.go` | Hands-free monitoring of long tasks |

### Tier 3 — Lower Feasibility, Very High Differentiation

| # | Feature | Inspired By | Estimated Effort | Why It Matters |
|---|---------|-------------|------------------|----------------|
| 9 | **Mobile Companion App** | Orca, OpenClaw | Large (~3000 lines, new repo) | Monitor/approve/send prompts from phone |
| 10 | **Live Canvas / A2UI** | OpenClaw | Very large (~2000 lines + HTML/JS sandbox) | Agent creates visual UI on the fly |
| 11 | **Multi-Instance Worktrees** | Orca | ~1000 lines in `internal/worktree/` | Fan-out to 5 agents, compare, merge winner |
| 12 | **Remote Agent Fleet** | Orca's SSH worktrees | ~800 lines + SSH client | Deploy agents to remote boxes with auto-reconnect |

---

## 6. Strategic Recommendations

### 6.1 Immediate (June 2026)

**Build Multi-Provider Expansion first.** roach-code (34 ⭐) proved it's straightforward: the `init()` registration pattern in `internal/provider/` was designed for this. Adding MiniMax, GLM, and OpenAI Responses API providers would make Reasonix Hermes the **most complete Reasonix fork** — all models, all infrastructure. Implementation is one file per provider.

### 6.2 Short-Term (July 2026)

**Add Scheduled Tasks.** This is the highest ROI feature we don't have. Both Hermes Agent and OpenClaw have it. It turns the agent from "interactive tool you must be present for" to "background worker you check in on." Our session persistence already supports this — we need a `[schedule]` config block and a cron engine.

### 6.3 Medium-Term (Q3 2026)

- **Session Publishing** (~300 lines) — low effort, high trust building. Export agent sessions as HTML/JSON for review.
- **Hash-Anchored Edits** (~150 lines) — simple safety net, especially for the Write Mode desktop feature.
- **LSP Integration** (~600 lines) — makes the agent IDE-aware, enabling "fix all diagnostics" workflows.

### 6.4 Long-Term / Aspirational

- **Mobile Companion App** — would make us the only Reasonix fork with mobile monitoring. Major differentiator.
- **Multi-Instance Worktrees** — unique feature that only Orca has. Would set us apart from every other Reasonix-based project.

### 6.5 What NOT to Build

- **Multi-IM channels (20+)** — OpenClaw dominates this space at 379k stars. We cannot catch up.
- **Voice/talk mode** — requires native device access; OpenClaw and Hermes Agent already invest heavily here.
- **Own model training** — Nous Hermes Agent does trajectory generation for model training. This is a research-scale effort.

---

## 7. Competitive Threats to Monitor

| Threat | Probability | Impact | Mitigation |
|--------|-------------|--------|------------|
| Upstream adds MCP server | Medium (v2.0?) | High — eliminates our marquee feature | Stay ahead with our unique tools (memory, constitution, skills hub) |
| roach-code adds MCP server | Low (34 ⭐, single dev) | Low | Would validate the pattern; our ecosystem is richer |
| Claude Code goes open source | Low (Anthropic won't) | Medium | Wouldn't affect DeepSeek users; different model ecosystem |
| OpenClaw adds Go binary | Very low (TypeScript-first team) | Low | Different architecture philosophy |
| New Reasonix fork appears | Medium (MIT license) | Low | We have brand + infrastructure + upstream sync |

---

## 8. Appendix: Ground Truth — What We Actually Have

This section documents every custom file in Reasonix Hermes, verified by codebase exploration. Use this as the source of truth for all competitive analysis.

### 8.1 Custom Binaries

| Binary | Location | Purpose | Size |
|---|---|---|---|
| `reasonix` | `cmd/reasonix/main.go` | Main CLI (upstream + our config) | ~26MB |
| `reasonix-mcpbridge` | `cmd/reasonix-mcpbridge/main.go` | MCP bridge server (6 tools) | ~14MB |
| `reasonix-memoryserver` | `cmd/reasonix-memoryserver/main.go` | Hindsight memory server (3 tools) | ~15MB |
| `reasonix-bot` | `bot/main.go` | Discord bot (real agent loop) | ~15MB |
| `reasonix-hooks` | `cmd/reasonix-hooks/main.go` | Native Go hook runner | ~9MB |

### 8.2 Desktop Hermes Widgets (12)

| Component | Location |
|---|---|
| CacheEconomyGauge | `desktop/frontend/src/components/hermes/CacheEconomyGauge.tsx` |
| CheckpointFileList | `desktop/frontend/src/components/hermes/CheckpointFileList.tsx` |
| CompactionTimeline | `desktop/frontend/src/components/hermes/CompactionTimeline.tsx` |
| ConstitutionHealthPanel | `desktop/frontend/src/components/hermes/ConstitutionHealthPanel.tsx` |
| DiscordMonitor | `desktop/frontend/src/components/hermes/DiscordMonitor.tsx` |
| GoalProgressWidget | `desktop/frontend/src/components/hermes/GoalProgressWidget.tsx` |
| HermesSettings | `desktop/frontend/src/components/hermes/HermesSettings.tsx` |
| MemoryFactGraph | `desktop/frontend/src/components/hermes/MemoryFactGraph.tsx` |
| ProfilePicker | `desktop/frontend/src/components/hermes/ProfilePicker.tsx` |
| SkillsHubBrowser | `desktop/frontend/src/components/hermes/SkillsHubBrowser.tsx` |
| SubagentTreePanel | `desktop/frontend/src/components/hermes/SubagentTreePanel.tsx` |
| TokenBreakdownChart | `desktop/frontend/src/components/hermes/TokenBreakdownChart.tsx` |

### 8.3 Skills Hub (17 Skills)

| Skill | Description |
|---|---|
| `code-review` | 5-dimension code review with severity levels |
| `debugger` | 5-step systematic debugging workflow |
| `deep-research` | Multi-source research with 3 depth tiers |
| `test-generator` | TDD unit test generation (Go/TS/Python/Rust/Java) |
| `council` | Multi-agent deliberation (Architect/Security/Performance/SRE/DX/Pragmatist) |
| `adversarial-review` | BLOCK/ALLOW structure with 5 attack surfaces |
| `api-design` | REST API design patterns |
| `ci-cd-helper` | CI/CD pipeline optimization |
| `database-helper` | Database schema and query optimization |
| `documentation` | Documentation generation and structure |
| `explore` | Codebase exploration |
| `frontend-builder` | Frontend component architecture |
| `git-commit` | Commit message generation |
| `migration-assistant` | Code migration patterns |
| `performance-profiler` | Performance analysis and optimization |
| `refactoring` | Refactoring strategies |
| `security-audit` | Security vulnerability assessment |

### 8.4 Infrastructure Files

| File | Purpose |
|---|---|
| `flake.nix` | Nix flake for reproducible builds (5 derivations) |
| `Dockerfile` | Multi-stage Docker build (distroless/static) |
| `reasonix-hermes.json` | Install source manifest for skills |
| `.github/workflows/sync-upstream.yml` | Daily upstream merge automation |
| `.github/workflows/ci-hermes.yml` | Hermes-specific CI (binaries, tests, frontend, lint) |
| `.github/workflows/e2e-bot.yml` | PR-triggered e2e tests against real provider |
| `.github/workflows/e2e-discord.yml` | Manual Discord smoke test |
| `.github/workflows/release-desktop.yml` | Desktop release pipeline |
| `scripts/install-skills.sh` | Batch skill installer |
| `scripts/desktop-build.sh` | Desktop build automation |
| `scripts/resolve-desktop-release.sh` | Desktop release resolution |

---

## 9. Changelog

| Date | Author | Changes |
|---|---|---|
| 2026-06-13 | Reasonix | Initial comprehensive competitive landscape analysis |
