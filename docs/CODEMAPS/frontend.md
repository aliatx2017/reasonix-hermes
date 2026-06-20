<!-- Generated: 2026-06-19 | Files: 180 TS/TSX + 77 components | Token estimate: ~900 -->

# Frontend — Reasonix Hermes Desktop

## Tech Stack
- **Framework**: Wails v2 (Go backend → Go `App` struct bound to JS via bridge)
- **UI**: React 19, TypeScript 6
- **Build**: Vite + pnpm
- **Entry**: `desktop/frontend/src/main.tsx` → `App.tsx`

## Architecture
```
Go backend (desktop/app.go, settings_app.go, tabs.go, hermes_dashboard.go...)
    ↓ Wails bridge (~230 bindings exposed as JS promises)
TypeScript AppBindings interface (lib/bridge.ts + mock app)
    ↓
React App (App.tsx) → 77 components + 35 lib modules
```

## Component Tree (App.tsx)
```
App
├── AppChrome
│   ├── TabBar (multi-tab with context menus)
│   ├── StatusBar (hermes group: cache%, Discord dot, sqz↓, aux↓)
│   └── TopBar (settings, update banner, onboarding)
├── Sidebar
│   ├── ProjectTree (file tree with icons, disclosure, scroll)
│   ├── IM connections (BotConnection per platform)
│   └── Navigation (history, trash, memory/skills, workspace)
├── WorkspacePanel
│   ├── File tree + preview
│   └── CodeViewer / DiffView
├── Transcript (main chat area)
│   ├── Message (user/assistant/system)
│   ├── ToolCard (tool calls + results + Young diagram/Katex)
│   ├── ProcessCard (goal mode)
│   ├── ApprovalModal (permission prompts, keyboard nav)
│   ├── AskCard (interactive questions)
│   └── ModelSwitcher (provider grouping + search)
├── Composer (input area)
│   ├── EffortSwitcher
│   ├── SlashMenu
│   └── Attachment/workspace refs
├── CommandPalette (Ctrl+K)
├── SettingsPanel
│   ├── General / Appearance / Models / Permissions
│   ├── Sandbox / Network / Agent / Bot / Memory / Skills / Hotbar
│   ├── Hermes (7 sections: cache, memory, Discord, goals, skills, subagents, constitution)
│   └── MCP / About
├── HistoryPanel
├── MemoryPanel (facts list + D3 force graph)
├── TodoPanel / ContextPanel
├── WritePanel (CodeMirror 6 markdown editor + preview)
└── UpdateBanner

## Hermes-Specific Components (desktop/frontend/src/components/hermes/)
| Component | Concern |
|-----------|---------|
| `BotLiveMonitor.tsx` | Multi-platform bot status (Discord/Telegram/LINE/Slack) |
| `CacheEconomyGauge.tsx` | Cache hit-rate badge + sparkline |
| `CompactionTimeline.tsx` | Per-turn compaction events |
| `CompressStatsWidget.tsx` | Token compressor savings (sqz↓ + aux↓) |
| `ConstitutionPanel.tsx` | Rule viewer + JSON template |
| `CostWidget.tsx` | Session cost summary |
| `CouncilPanel.tsx` | Multi-model council (peers + status) |
| `EvalPanel.tsx` | Session comparison (Jaccard, tool table) |
| `GoalProgressWidget.tsx` | Goal bar + status badges |
| `HindsightDashboard.tsx` | Memory facts/docs/scopes list |
| `LearnedPatternsPanel.tsx` | Learner patterns + trajectories |
| `MarketplacePanel.tsx` | Skill marketplace browser + LobeHub sync |
| `OrchestratePanel.tsx` | Chain/Pair/CI-Fix copyable commands |
| `PublishWidget.tsx` | Session export (HTML/JSON) |
| `ScheduleWidget.tsx` | Cron task CRUD (add/remove/edit) |
| `SkillStorePanel.tsx` | Unified 4-tab skill browser (LobeHub/Market/MCP/Custom) |
| `SubagentTreePanel.tsx` | Task tree with status badges |
| `TokenSparkline.tsx` | Per-turn token bar chart |
| `CollabPanel.tsx` | Live collab hub (watchers + sessions) |

## Key Lib Modules
| Module | Concern |
|--------|---------|
| `bridge.ts` | Wails Go ↔ JS binding interface (~250 methods) + mock app for dev |
| `useController.ts` | Session state machine: turn dispatch, streaming, abort |
| `useHermesLiveData.ts` | Push-event driven Hermes dashboard data |
| `types.ts` | All TypeScript interfaces (80+ view structs) |
| `i18n.tsx` | Translation engine (en, zh, zh-TW) |
| `theme.ts` | Theme resolution (graphite, aurora, slate, carbon, nocturne, amber, hermes) |
| `workspaceLayout.ts` | Responsive dock layout math |
| `tools.ts` | Tool approval mode normalization |
| `sessionExport.tsx` | Session → Markdown/ZIP export |

## State Management
- **Session state**: `useController` hook → `app.Controller` bridge calls
- **Live data**: `useHermesLiveData` → Wails EventsOn push (no polling)
- **UI state**: React `useState` (tab, panel visibility, theme, locale) + `useRef` (hotbar bindings)
- **Settings**: loaded from Go `App.Settings()` → `SettingsView` interface
- **Persistence**: Wails bridge writes back through Go `App.Set*()` methods → `config.Config` → TOML file

## Hotbar (keyboard shortcuts, keys 1-7)
Config-driven via `[desktop.hotbar]` in reasonix.toml:
```
1 → palette     (CommandPalette)
2 → workspace   (WorkspacePanel toggle)
3 → new         (new session)
4 → history     (HistoryPanel)
5 → dock        (ContextPanel toggle)
6 → sidebar     (sidebar toggle)
7 → settings    (SettingsPanel)
```
Unknown/unset keys fall back to defaults. Go side validates action names on load.
