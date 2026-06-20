<!-- Generated: 2026-06-20 | Storage backends: 4 (SQLite, filesystem, JSON sidecars, embeddings) | Token estimate: ~500 -->

# Data Layer — Reasonix Hermes

## Storage Backends

### 1. Config (TOML, filesystem)
- `~/.config/reasonix/config.toml` — user global config
- `./reasonix.toml` — project-level override
- Loaded via `internal/config/config.go` (TOML → `Config` struct)
- Edit surface: `internal/config/edit.go` (atomic write via temp file)
- Render surface: `internal/config/render.go` (all 30+ sections preserved on save)

### 2. Hindsight Memory (SQLite + file)
- Package: `cmd/reasonix-memoryserver/`
- Storage: `memory.db` (SQLite with WAL mode)
- Tables: `memories` (id, content, ttl, importance, dense_vector, embedding_b64, created_at, last_access_at, last_decay_at)
- Operations: `retain` (upsert), `recall` (sparse + dense hybrid), `reflect` (session summary)
- Vector search: cosine similarity via stored embeddings (sparse TF-IDF + dense external)
- Dense embeddings: configurable provider (OpenAI-compatible API), auto-embed on retain
- TTL: automatic expiry via periodic cleanup; `Tidy()` preserves `CreatedAt` via `LastDecayAt`
- Logging: structured `slog` (was `log.Printf`)
- Graceful shutdown: SIGINT/SIGTERM handler closes DB

### 3. Session Storage (filesystem)
- `~/.config/reasonix/sessions/` — chat transcripts (JSONL)
- `desktop/tabs.go` — tab metadata (`tabs.json`, `projects.json`, `topics/`)
- Sidecar files: `.sessionstats` (token counts, turns, cost), `.meta` (branch metadata)
- Session keys: compound hash of platform + chat type + chat ID + user ID
- Agent log: `agent.log` with rotation (size-based, `agent.log.1`, `agent.log.2`, …)

### 4. Bot Model Prefs + LobeHub Sync (JSON files)
- `~/.config/reasonix/bot-model-prefs.json` — per-session model preferences
- `~/.config/reasonix/lobehub-sync.json` — LobeHub marketplace sync metadata
- Written on `/model` command or marketplace sync; loaded on init

### 5. Codegraph Index + Checkpoints (filesystem)
- `internal/codegraph/` — LSP-based semantic index (`.codegraph/` per project)
- `internal/checkpoint/` — Snapshot-based edit safety net (file-level diffs)

### 6. Learner Sidecar (filesystem)
- `.learning` JSON snapshots per session — tool sequences, patterns, trajectories
- Written by `internal/learn/` after each turn; read by desktop `LearnedPatterns()`

### 7. Constitution (JSON)
- `.reasonix/constitution.json` — structured invariants (principles, constraints, rules)

## No External Databases
The project is self-contained: all data lives on the local filesystem or embedded SQLite. No Postgres, Redis, MongoDB.
