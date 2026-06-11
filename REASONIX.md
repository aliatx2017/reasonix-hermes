# Reasonix project memory

This file is loaded into every session's system prompt (the cache-stable prefix),
so keep it concise and durable — it is the project's standing instructions to the
agent. It is the Reasonix analog of Claude Code's CLAUDE.md.

## Conventions

- Go kernel under `internal/`; each package owns one concern and documents it in a
  package comment. Match the surrounding comment density and idiom when editing.
- One transport-agnostic `control.Controller` sits behind every frontend (chat
  TUI, HTTP/SSE serve, Wails desktop). Add behavior to the controller, not a
  frontend, so all three inherit it.
- Cache-first: the system-prompt prefix (base prompt + tools + memory) must stay
  byte-stable across turns so DeepSeek's automatic prefix cache stays warm. Never
  mutate it mid-session — ride the turn tail instead (see `control.Compose`).

## Memory

- Hierarchical docs: `REASONIX.md` (this file, committed/shared), `REASONIX.local.md`
  (personal, git-ignored), user-global `~/.config/reasonix/REASONIX.md`, and any
  `REASONIX.md` in an ancestor dir. `AGENTS.md` is accepted as a fallback name.
- `@path` on its own line imports another file's contents.
- `#<note>` in chat quick-adds a line here. The `remember` tool saves durable
  facts to the per-project auto-memory store (frontmatter files + `MEMORY.md`
  index), which loads into the prefix on the next session.

## Notes

- **Upstream synced**: `v1.5.0` merged (commit e5e8f02, 2026-06-25). Clean merge, zero conflicts.
- **Implementation plan**: `docs/HERMES-IMPLEMENTATION-PLAN.md` — P0 ✅, P1 ✅, P2 ✅, P3 ✅. All 17 plan items complete. Remaining: install_source integration, CI extension, VS Code fork.
- **Ecosystem reference**: `reasonix-deepseek-ecosystem-2026.md` + `docs/RESEARCH-FINDINGS-JUNE-2026.md`.
- **Key differentiators**: Discord bot (real agent loop + /goal + /model), MCP bridge (6 tools), Hindsight memory (3 tools, SQLite + TTL/importance + vector search), 17-skill registry, native Go hooks, portable mode.

## Architecture Notes (post-v1.5.0 sync)

- `control.Options` fields: `Runner`, `Executor`, `Sink`, `Policy`, `Hooks`, `OnRemember`, `Registry`, `PluginCtx`, `Jobs`, `BalanceURL/Key`, `AutoPlan`, `Classifier`, `Label`, `SystemPrompt`, `SessionDir/Path`, `Host`, `Commands`, `Skills/AllSkills/SkillStore/AllSkillStore`, `Memory`, `Cleanup`, `WorkspaceRoot`.
- `event.Sink` = `Emit(Event)` — single-method interface. `event.FuncSink` adapts closures. `event.Discard` for no-op.
- `permission.Approver` = `Approve(ctx, toolName, subject, args) (allow, remember, err)`. Front-end implements interactive approval.
- `permission.Gate` = `Policy` + optional `Approver`. Nil Approver = auto-allow (yolo).
- `install_source` tool exists in upstream (`internal/installsource/`) — skills hub should integrate with it, not build a separate script.
- `hook.Runner` exists upstream — supports `PreToolUse`/`PostToolUse`/`Stop` hooks. Memory server P3 should use this.

## Next Session TODOs

- **Skills hub website** — deploy `skills-hub/site/index.html` to GitHub Pages at `aliatx2017.github.io/reasonix-hermes/`
- **CI: e2e-bot Discord adapter** — extend `.github/workflows/e2e-bot.yml` with Discord smoke test (needs bot token)
- **VS Code extension** — fork `whishi47/deepseekcode-reasonix-vscode` (separate repo, Hermes branding)
- **roach-code multi-provider** — study `tmdgusya/roach-code` for MiniMax/GLM/Anthropic provider patterns
