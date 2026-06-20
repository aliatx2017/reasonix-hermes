# Reasonix-Hermes

A config-driven, plugin-driven coding agent platform — a thin harness that drives multiple LLM providers through a transport-agnostic Controller, with all capabilities supplied by configuration and plugins.

## Language

### Agent loop

**Turn**:
One conversational cycle: a user message, the agent's full response, and any tool calls the agent makes before producing its final answer.
_Avoid_: Round, iteration, request/response cycle

**Turn tail**:
The position after the last message of a turn, used by the Controller to inject memory blocks and context without mutating the system prompt. Preserves the prefix cache.
_Avoid_: Append slot, injection point

**Compaction**:
Reducing conversation history by summarizing or discarding old turns when the context window nears its limit. Triggered by `compact_ratio` / `compact_force_ratio` thresholds.
_Avoid_: Truncation, summarization (compaction includes both)

**Subagent**:
An isolated child agent spawned by the parent for a focused sub-task. Its tool calls and reasoning never enter the parent's context — only its final answer returns.
_Avoid_: Worker, child task, delegate

**Plan Mode**:
A read-only agent posture that blocks writer tools. Four tool names are unconditionally denied: `write_file`, `edit_file`, `multi_edit`, `apply_patch`. Every other non-ReadOnly tool is also blocked with a generic message (the agent can request `plan_mode_allowed_tools` exceptions in config). Bash commands with chaining/redirection metacharacters are blocked even if the prefix matches a safe command. Plan mode is activated by `/plan` or auto-detected from ambiguous input. Prepends a `[PLAN MODE]` marker to user turns so the LLM knows writes will be blocked.
_Avoid_: Dry run, preview mode

**Goal**:
An active-task finite-state machine that tracks a multi-step objective across turns. Statuses: `running`, `complete`, `blocked`, `stopped`. Supports research sub-modes where the agent auto-outlines before executing. Managed by the GoalMachine — a strict leaf that never calls back into the Controller.
_Avoid_: Task, objective, mission

**Tool**:
A capability the agent invokes (bash, read, write, edit, etc.). Registered via `init()` for built-ins or via MCP for external plugins.
_Avoid_: Function, action, command

**Skill**:
A reusable playbook loaded on demand. Inline skills fold into the parent turn; subagent skills spawn an isolated child. Stored as `SKILL.md` files.
_Avoid_: Prompt, template, rule

### Architecture

**Controller**:
The concrete implementation behind the SessionAPI port. Constructs the agent, wires tools, and orchestrates turns. Frontends never couple to Controller directly — they depend on the SessionAPI interface.
_Avoid_: Handler, dispatcher, engine

**Port / SessionAPI**:
The driving port — a typed, segregated interface surface that frontends consume instead of coupling to the concrete Controller. Composed of 14 sub-ports (Lifecycle, TurnControl, Approvals, Goals, SessionHistory, MemoryControl, Capabilities, Status, SessionPersistence, Input, Settings, MeshState, LearnState, ScheduleState). Each frontend depends only on the sub-ports it uses.
_Avoid_: API, facade, wrapper

**Prefix**:
The byte-stable front of the context window — system prompt, tool schemas, and memory files. Frozen after the first turn so DeepSeek's automatic prefix cache stays warm across turns. Never mutated mid-session.
_Avoid_: System prompt (the system prompt is part of the prefix, not the whole thing)

**Provider**:
An LLM API backend implementing the `Provider` interface. Distinct from a _model_ — a provider exposes multiple models (e.g., `openai` provider exposes `deepseek-v4-flash`, `deepseek-v4-pro`).
_Avoid_: Backend, API, service

**Frontend**:
A transport-specific UI layer. The three canonical frontends: CLI TUI (`reasonix chat`), HTTP/SSE (`reasonix serve`), Wails desktop (`reasonix-desktop`). All share one Controller.
_Avoid_: Client, interface, UI

**Registry**:
The interface-based plugin system. Providers and tools self-register via Go `init()` functions. Resolution is by name from configuration, not by hardcoded switch statements.
_Avoid_: Plugin system, module system

**Coordinator**:
A two-model collaboration runner where a distinct planner model (with its own session) wraps an executor agent. The planner gets a read-only research tool set to inspect rules and code without side effects. Keeps each model's prompt cache stable by isolating sessions. Activated via `planner_model` in config.
_Avoid_: Planner, orchestrator, two-stage pipeline

### Multi-agent & bots

**Mesh**:
Agent-to-agent communication over MCP. Operations: delegate a task to a peer, broadcast to all peers, query a peer's status, convene a council for multi-perspective analysis.
_Avoid_: Federation, cluster, swarm

**Gateway**:
The multi-platform bot message router. Receives inbound messages from platform adapters, routes them to Controllers, and sends responses back through the correct adapter.
_Avoid_: Hub, dispatcher, broker

**Adapter**:
A platform-specific bot integration implementing the `Adapter` interface. One adapter per platform connection (Discord, Telegram, LINE, Slack, Feishu, WeChat, QQ).
_Avoid_: Connector, bridge, driver

### Governance

**Constitution**:
Structured project invariants stored in `.reasonix/constitution.json`. Encodes principles, constraints, and rules — the project's behavioral contract. Enforced structurally rather than by convention.
_Avoid_: Rules, guidelines, policy (the constitution is above guidelines)

**Sandbox**:
OS-level isolation for tool execution. macOS uses Seatbelt profiles; Linux uses bubblewrap (bwrap). Read-only root filesystem, writable workspace only, network isolation. Falls back gracefully when the sandbox binary is missing.
_Avoid_: Container, jail, chroot

**Checkpoint**:
A saved point in a session that can be rewound to via `Rewind()`. Tracks files created/modified since the checkpoint. Distinguished from _branch_ (a divergent session path) and _snapshot_ (a read-only capture).
_Avoid_: Savepoint, rollback point

### Persistence

**Session**:
A single conversation with the agent, persisted to disk as a transcript directory under `.reasonix/projects/`. Contains message history (`.jsonl`), tool outputs, and sidecar metadata (`.sessionstats`, `.meta`, `.learning`, `.ckpt`).
_Avoid_: Chat, conversation, transcript (the session includes metadata beyond the transcript)

**Memory**:
Cross-session persistent storage with TTL-based decay and vector search. Implemented by the Hindsight memory server (SQLite). Two recall modes: sparse (keyword BM25) and dense (embedding cosine similarity).
_Avoid_: Knowledge base, long-term memory, storage
