# Controller seam — one port behind every frontend

All behavior — message composition, tool approval, compaction, session persistence, memory injection — is added behind the `SessionAPI` driving port, never to individual frontends. The four frontends (CLI TUI, HTTP/SSE serve, Wails desktop, bot gateway) consume the same interface, making every feature available everywhere with a single implementation.

**Why:** Without this seam, each frontend would independently reimplement the agent loop, leading to divergent behavior, duplicated bugs, and features that work in the CLI but not the desktop (or vice versa). The SessionAPI port is the single source of truth for the agent's runtime behavior.

**Considered alternatives:**
- **Per-frontend features**: Add behavior directly in the TUI (`internal/cli/`), serve handler (`internal/serve/`), desktop app (`desktop/`), or bot (`internal/bot/`). Rejected — leads to 4× implementation burden and inevitable drift.
- **Shared library called by each frontend**: Better than per-frontend, but still allows each frontend to call the library differently. The port pattern ensures the call _order_ and _state_ are invariant.
- **Plugin hooks**: Each frontend registers callbacks. Rejected — adds complexity without benefit; the port already covers every customization point (tool approval, compaction, session routing).
