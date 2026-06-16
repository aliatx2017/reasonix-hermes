<!-- Generated: 2026-06-06 | Files: 101 TS/TSX + 48 components | Token estimate: ~850 -->

# Frontend — Reasonix Hermes Desktop

## Tech Stack
- **Framework**: Wails v2 (Go backend → Go `App` struct bound to JS via bridge)
- **UI**: React 19, TypeScript 6
- **Build**: Vite + pnpm
- **Entry**: `desktop/frontend/src/main.tsx` → `App.tsx`

## Architecture
```
Go backend (desktop/app.go, settings_app.go, tabs.go...)
    ↓ Wails bridge (bindings exposed as JS promises)
TypeScript AppBindings interface (lib/bridge.ts)
    ↓
React App (App.tsx) → 48 components + 35 lib modules
```

## Component Tree (App.tsx)
```
App
├── AppChrome
│   ├── TabBar
│   ├── StatusBar
│   └── TopBar (settings, update banner, onboarding)
├── Sidebar
│   ├── ProjectTree
│   ├── IM connections (BotConnection)
│   └── Navigation (history, trash, memory/skills, workspace)
├── WorkspacePanel
│   ├── File tree + preview
│   └── CodeViewer / DiffView
├── Transcript (main chat area)
│   ├── Message (user/assistant/system)
│   ├── ToolCard (tool calls + results)
│   ├── ProcessCard (goal mode)
│   ├── ApprovalModal (permission prompts)
│   └── AskCard (interactive questions)
├── Composer (input area)
│   ├── ModelSwitcher / EffortSwitcher
│   ├── SlashMenu
│   └── Attachment/workspace refs
├── CommandPalette (Ctrl+K)
├── SettingsPanel
│   ├── General / Appearance / Models / Permissions
│   ├── Sandbox / Network / Agent / Bot / Memory / Skills / Hotbar
│   └── MCP / About
├── HistoryPanel
├── MemoryPanel
├── TodoPanel / ContextPanel
└── UpdateBanner
```

## Key Lib Modules
| Module | Concern |
|--------|---------|
| `bridge.ts` | Wails Go ↔ JS binding + mock app for dev |
| `useController.ts` | Session state machine: turn dispatch, streaming, abort |
| `types.ts` | All TypeScript interfaces (SettingsView, SessionMeta, Messages…) |
| `i18n.tsx` | Translation engine (en, zh) |
| `theme.ts` | Theme resolution (graphite, aurora, slate, carbon, nocturne, amber) |
| `workspaceLayout.ts` | Responsive dock layout math |
| `tools.ts` | Tool approval mode normalization |
| `sessionExport.tsx` | Session → Markdown/ZIP export |

## State Management
- **Session state**: `useController` hook → `app.Controller` bridge calls
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
Unknown/unset keys silently fall back to defaults. Go side validates action names on load.
