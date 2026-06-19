# Cache-first immutable prefix

The system prompt prefix — base prompt, tool schemas, and memory files — is frozen after the first turn and never mutated mid-session. Memory blocks that arrive later ride the turn tail (appended as a user message after the last assistant response), never injected into the prefix.

**Why:** DeepSeek's automatic prefix cache gives a 50-120× cost reduction on cache hits vs. misses ($0.02/M vs. $1/M for v4-flash). Every byte change in the prefix breaks the entire cache for that turn. The turn-tail injection pattern preserves ~99.8% cache hit rates without sacrificing the ability to add context mid-session.

**Considered alternatives:**
- **Dynamic prompt assembly** (edit the system prompt every turn): cache miss on every turn → 50× cost increase. Rejected.
- **Separate memory-provider API call**: adds latency and a second round-trip per turn. Turn-tail injection is free.
- **Recompute only changed suffix**: DeepSeek's prefix cache is byte-prefix-based — any change at any position invalidates everything after it. Not viable.
