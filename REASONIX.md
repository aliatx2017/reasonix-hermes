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

- **Upstream tracking**: `v1.5.0` (June 10, 2026) — we need to sync. Major features: bot gateway (Feishu/Weixin/QQ), goal mode, read_skill tool, PDF extraction, themeable workspace, React 19/TypeScript 6, ACP sessions, 100+ fixes.
- **Implementation plan**: `docs/HERMES-IMPLEMENTATION-PLAN.md` — phased: P0 (sync + bot wiring + tests), P1 (skills hub + memory hooks), P2 (collab-cli + VS Code ext), P3 (portability + vector memory).
- **Ecosystem reference**: `reasonix-deepseek-ecosystem-2026.md` + `docs/RESEARCH-FINDINGS-JUNE-2026.md` — full survey of MCP bridges, skills, desktop clients, IDE extensions, forks, undocumented features.
- **Key differentiators for Hermes**: Discord bot with real agent loop (unique in ecosystem), MCP bridge server (5 tools), Hindsight memory server (3 tools), 16-skill curated registry. The bot must use `control.Controller` like every other frontend — not inline chat.
