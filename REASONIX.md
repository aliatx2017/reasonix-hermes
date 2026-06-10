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

- **Upstream synced**: `v1.5.0` merged (commit e5e8f02, 2026-06-25). Clean merge, zero conflicts. 7 upstream files (workspace layout, dock fixes, site config).
- **Implementation plan**: `docs/HERMES-IMPLEMENTATION-PLAN.md` — phased: P0 (sync ✅ + bot wiring + tests), P1 (skills hub + memory hooks), P2 (collab-cli + VS Code ext), P3 (portability + vector memory).
- **Ecosystem reference**: `reasonix-deepseek-ecosystem-2026.md` + `docs/RESEARCH-FINDINGS-JUNE-2026.md` — full survey of MCP bridges, skills, desktop clients, IDE extensions, forks, undocumented features.
- **Key differentiators for Hermes**: Discord bot with real agent loop (unique in ecosystem), MCP bridge server (5 tools), Hindsight memory server (3 tools), 16-skill curated registry. The bot must use `control.Controller` like every other frontend — not inline chat.

## Next Session TODOs

- **P1: Wire Discord bot → `control.Controller`** — `bot/main.go` still calls `simulateReasonix()`. Need `DiscordSink` implementing `event.Sink`, `DiscordApprover` implementing `permission.Approver`, per-channel `BotSession` wrapping `control.Controller`. Implementation plan §1.1-1.2 has full design.
- **P2: Test coverage for custom packages** — `pkg/mcpbridge/`, `pkg/memoryserver/`, `bot/` have zero `_test.go` files. Target 60%+. Implementation plan §2.1-2.3.
- **P2: HTTP auth for MCP servers** — bridge (port 9090) and memory (port 8080) accept unauthenticated requests. Add API key header check or bearer token.
- **P3: Skills hub auto-loading** — `skills-hub/` has 16 Markdown skills + `registry.json` but no install mechanism. Need `scripts/install-skills.sh` or `reasonix install-source` integration. Implementation plan §3.1-3.3.
- **P3: Memory server hook mode** — Reasonix `PreToolUse`/`PostToolUse`/`Stop` hooks for automatic session memory. Implementation plan §4.1.
