# Reasonix-Hermes Desktop App

> Wails v2 desktop application with React 19 frontend and Go kernel.
> The Hermes Dashboard provides live monitoring of agent sessions, bot
> connections, cache economy, memory, and more.

## Quick Start

```bash
cd desktop/frontend && npm install && cd ../..
cd desktop && wails dev              # development with hot reload
cd desktop && wails build -o ../bin/reasonix-desktop  # production build
```

## Hermes Dashboard

The desktop app includes a rich monitoring dashboard with live data:

### Tier 1 — Core Monitoring
| Widget | Shows |
|--------|-------|
| **Cache Economy Gauge** | Prefix-cache hit rate as percentage badge |
| **Memory Dashboard** | Total facts, docs, scopes from Hindsight memory |
| **Discord Bot Monitor** | Online/offline status, active sessions, webhook URL |
| **Goal Progress** | Current goal, status, turns/blocks |

### Tier 2 — Live Data
| Widget | Shows |
|--------|-------|
| **StatusBar Hermes Chips** | Cache%, Discord dot, sqz/aux savings |
| **Skills Hub Browser** | 17 built-in skills with search + category filter |
| **Compress Gauge** | Bytes saved by token compressor, aux token savings |

### Tier 3 — Advanced
| Widget | Shows |
|--------|-------|
| **Sub-agent Task Tree** | Hierarchical tree of dispatched sub-agents with status badges |
| **Constitution Health** | Rules viewer with status, JSON template |
| **Checkpoint File List** | Per-turn file tracking with diff vs. current |
| **Token Sparkline** | Bar chart of token usage per turn (ring buffer) |
| **Compaction Timeline** | When and why compaction events occurred |
| **Collab Panel** | WebSocket watchers, shared sessions |
| **Council Panel** | Mesh peers and active council status |
| **Schedule Widget** | Cron tasks with ±/✎/✕ controls |
| **Cost Widget** | Session cost summary with currency |
| **Publish Widget** | Export session as HTML or JSON |

## Key Features

### Write Mode
Split-pane markdown editor with CodeMirror 6, FIM completions (Ctrl+Space →
DeepSeek API), Hindsight memory sidebar, file tabs, auto-save.

### Bot Connection
Connect Discord, Telegram, LINE, Slack, Feishu, or WeChat bots from the
Settings → Bots panel. Live status monitoring with diagnostic reporting.

### Session Management
Multi-tab session workspace, per-tab model/effort selection, checkpoint
rewind with hover preview, session history sidebar.

### Skill Store
4-tab panel for discovering and installing skills: LobeHub sync, built-in
marketplace registry, live MCP server status, custom skill management.

### Hotbar
Customizable keyboard shortcuts (keys 1-7) for quick model switching,
effort toggles, and profile activation.

## Architecture

```
desktop/
├── app.go              # Wails App struct: 200+ bound methods
├── hermes_dashboard.go # Hermes dashboard live data bindings
├── hermes_tier3.go     # Advanced monitoring (subagents, constitution)
├── hermes_eval.go      # Session comparison binding
├── settings_app.go     # Config persistence (theme, hotbar, profiles, bots)
├── tabs.go             # Multi-tab session management
├── updater_app.go      # Auto-update with R2 canary/stable channels
├── crash_app.go        # Crash capture + breadcrumb reporting
├── frontend/           # React 19 + TypeScript 6 + Vite 8
│   ├── src/
│   │   ├── components/       # UI components
│   │   │   ├── hermes/       # 20+ Hermes dashboard widgets
│   │   │   ├── editors/      # CodeMirror 6 integration
│   │   │   └── ...           # 40+ upstream components
│   │   └── lib/              # bridge.ts (Go↔React seam), types, i18n
│   └── wailsjs/              # Auto-generated Wails bindings (gitignored)
└── build/              # Platform build assets (icons, plists)
```

Live data flows: `Agent → Controller → Wails event → React useHermesLiveData hook`.
The bridge (`bridge.ts`) is the single seam between the Go kernel and the React UI.

## Configuration

Desktop-specific settings in `reasonix.toml`:

```toml
[desktop]
theme = "auto"          # auto | dark | light
theme_style = "hermes"  # hermes | graphite | ocean | forest | sunset
layout_style = "classic" # classic | workbench

[desktop.hotbar]
key1 = "/model flash"
key2 = "/effort high"
# ...

[[profiles]]
name = "review"
model = "deepseek-v4-pro"
effort = "high"
```

## Related

- `docs/HERMES-GUIDE.md` — full Hermes feature guide
- `docs/EVAL.md` — session comparison tool
- `docs/MARKETPLACE.md` — skill marketplace
- `docs/CONSTITUTION.md` — project invariants
