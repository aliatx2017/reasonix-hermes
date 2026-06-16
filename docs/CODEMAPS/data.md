<!-- Generated: 2026-06-06 | Storage backends: 2 (SQLite, filesystem) | Token estimate: ~400 -->

# Data Layer — Reasonix Hermes

## Storage Backends

### 1. Config (TOML, filesystem)
- `~/.config/reasonix/config.toml` — user global config
- `./reasonix.toml` — project-level override
- Loaded via `internal/config/config.go` (TOML → `Config` struct)
- Edit surface: `internal/config/edit.go` (atomic write via temp file)

### 2. Hindsight Memory (SQLite + file)
- Package: `cmd/reasonix-memoryserver/`
- Storage: `sqlite_storage.go` — SQLite (`memory.db`)
- Tables: `memories` (id, content, ttl, importance, embedding_b64, created_at, access_at)
- Operations: `retain` (upsert), `recall` (vector + keyword hybrid), `reflect` (session summary)
- Vector search: cosine similarity via stored embeddings
- TTL: automatic expiry via `DELETE WHERE created_at + ttl < now()`
- Graceful shutdown: SIGINT/SIGTERM handler closes DB

### 3. Session Storage (filesystem)
- `~/.config/reasonix/sessions/` — chat transcripts (JSON/TOML)
- `desktop/tabs.go` — tab metadata (`tabs.json`, `projects.json`, `topics/`)
- Session keys: compound hash of platform + chat type + chat ID + user ID

### 4. Bot Model Prefs (JSON file)
- `~/.config/reasonix/bot-model-prefs.json` — per-session model preferences
- Written on every `/model` command; loaded on gateway init
- `internal/bot/gateway.go`: `saveModelPrefs()` / `loadModelPrefs()`

### 5. Codegraph Index (filesystem)
- `internal/codegraph/` — LSP-based semantic index
- `.codegraph/` directory per project (symbols, edges, positions)

## No External Databases
The project is self-contained: all data lives on the local filesystem or embedded SQLite. No Postgres, Redis, MongoDB.
