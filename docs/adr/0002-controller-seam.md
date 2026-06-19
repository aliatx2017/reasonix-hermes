# Controller seam — one orchestrator behind every frontend

All behavior — message composition, tool approval, compaction, session persistence, memory injection — is added to `control.Controller`, never to individual frontends. The three frontends (CLI TUI, HTTP/SSE serve, Wails desktop) share one Controller instance, making every feature available everywhere with a single implementation.

**Why:** Without this seam, each frontend would independently reimplement the agent loop, leading to divergent behavior, duplicated bugs, and features that work in the CLI but not the desktop (or vice versa). The Controller is the single source of truth for the agent's runtime behavior.

**Considered alternatives:**
- **Per-frontend features**: Add behavior directly in the TUI (`internal/cli/`), serve handler (`internal/serve/`), and desktop app (`desktop/`). Rejected — leads to 3× implementation burden and inevitable drift.
- **Shared library called by each frontend**: Better than per-frontend, but still allows each frontend to call the library differently. The Controller pattern ensures the call _order_ and _state_ are invariant.
- **Plugin hooks**: Each frontend registers callbacks. Rejected — adds complexity without benefit; the Controller already covers every customization point (tool approval, compaction, session routing).
