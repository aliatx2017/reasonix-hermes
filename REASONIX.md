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
- **Language policy**: All Chinese comments translated to English in `internal/bot/` and `internal/config/` — SPEC §1 compliance restored.
- **Layout**: Executables moved from `pkg/` to `cmd/`: `pkg/mcpbridge/` → `cmd/reasonix-mcpbridge/`, `pkg/memoryserver/` → `cmd/reasonix-memoryserver/`.
- **Bug fixes applied** (June 12, 2026):
  - P0-1: BotGateway session eviction (idle timeout + background goroutine)
  - P0-2: `http.DefaultClient` → `netclient.DefaultClient()` across 7 files
  - P1-1: `sqliteStorage.Close()` deferred in memoryserver main
  - P2-1: `DELETE`+`INSERT` → `INSERT OR REPLACE` upsert
  - P2-2: LIKE wildcard escaping with `ESCAPE '\'` clause
  - P2-3: `io.LimitReader` on 5 unbounded HTTP response reads
- **Docs written/updated**:
  - New `docs/HERMES-GUIDE.md` — 1,300+ line comprehensive master guide (19 sections)
  - Rewrote `README.md` for Hermes identity
  - Updated `docs/SPEC.md` §2 (39 packages) and §4 (7 ChunkTypes)
  - Updated `docs/GUIDE.md` with `/goal` and `/effort`

- **Key differentiators**: Discord bot (real agent loop + /goal + /model), MCP bridge (6 tools), Hindsight memory (3 tools, SQLite + TTL/importance + vector search), 17-skill registry, native Go hooks, portable mode.
