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

## Architecture Notes (post-v1.5.0 sync)

- `control.Options` fields: `Runner`, `Executor`, `Sink`, `Policy`, `Hooks`, `OnRemember`, `Registry`, `PluginCtx`, `Jobs`, `BalanceURL/Key`, `AutoPlan`, `Classifier`, `Label`, `SystemPrompt`, `SessionDir/Path`, `Host`, `Commands`, `Skills/AllSkills/SkillStore/AllSkillStore`, `Memory`, `Cleanup`, `WorkspaceRoot`.
- `event.Sink` = `Emit(Event)` — single-method interface. `event.FuncSink` adapts closures. `event.Discard` for no-op.
- `permission.Approver` = `Approve(ctx, toolName, subject, args) (allow, remember, err)`. Front-end implements interactive approval.
- `permission.Gate` = `Policy` + optional `Approver`. Nil Approver = auto-allow (yolo).
- `install_source` tool exists in upstream (`internal/installsource/`) — skills hub should integrate with it, not build a separate script.
- `hook.Runner` exists upstream — supports `PreToolUse`/`PostToolUse`/`Stop` hooks. Memory server P3 should use this.

## Next Session TODOs

- **P2: CI pipeline** — extend `.github/workflows/ci-hermes.yml` with `go test ./pkg/... ./bot/... ./internal/bot/...` steps. Current CI only covers desktop frontend.
- **P2: collab-cli integration** — add as pre-configured MCP plugin in `reasonix.example.toml`. 17 free tools (handshake, tasks, SHARD.md, agent commands, self-review).
- **P2: VS Code extension** — fork whishi47/deepseekcode-reasonix-vscode, add Hermes branding, publish to Marketplace.
- **P2: Adversarial review skill** — port kquuen BLOCK:/ALLOW: contract as `skills-hub/skills/adversarial-review.md`.
- **P3: Hook scripts → upstream `hook.Runner`** — replace `retain-hook.sh`/`reflect-hook.sh` shell scripts with native Go hooks via upstream `internal/hook` package. Shell scripts are hardened but fragile.
- **P3: Memory backend: SQLite** — current file-based JSON doesn't scale. Add SQLite backend with indexed search for `pkg/memoryserver/`.
- **P3: Memory TTL + importance scoring** — per-fact TTL (default 90d), frequently recalled facts get longer TTL, project-scoped isolation.
- **P3: `read_skill` MCP tool** — expose upstream's `read_skill` via mcpbridge so external agents can load skills.
- **P3: Discord `/goal` command** — leverage upstream goal mode from Discord bot.
- **P3: PortaKit portability** — `--portable` flag, auto-detect data dir relative to binary.
