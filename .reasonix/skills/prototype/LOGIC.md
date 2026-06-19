# Logic Prototype
A tiny interactive terminal app that lets the user drive a state model by hand. Use this when the question is about **business logic, state transitions, or data shape** — the kind of thing that looks reasonable on paper but only feels wrong once you push it through real cases.

## When this is the right shape
- "I'm not sure if this state machine handles the edge case where X then Y."
- "Does this data model actually let me represent the case where..."
- "I want to feel out what the API should look like before writing it."
- Anything where the user wants to **press buttons and watch state change**.
If the question is "what should this look like" — wrong branch. Use [UI.md](UI.md).

## Process
### 1. State the question
Before writing code, write down what state model and what question you're prototyping. One paragraph, in the prototype's README or a comment at the top of the file.

### 2. Pick the language
Use whatever the host project uses. Match the project's existing conventions for tooling — don't add a new package manager or runtime just for the prototype.

### 3. Isolate the logic in a portable module
Put the actual logic behind a small, pure interface that could be lifted out and dropped into the real codebase later. The TUI around it is throwaway; the logic module shouldn't be.
Pick whichever shape best fits the question being asked, *not* whichever is easiest to wire to a TUI. Keep it pure: no I/O, no terminal code. The TUI imports it and calls into it; nothing flows the other direction.

### 4. Build the smallest TUI that exposes the state
Build it as a **lightweight TUI** — on every tick, clear the screen and re-render the whole frame. The user should always see one stable view, not an ever-growing scrollback.
Each frame has two parts, in this order:
1. **Current state**, pretty-printed and diff-friendly
2. **Keyboard shortcuts**, listed at the bottom: `[a] add  [d] delete  [q] quit`
Behaviour:
1. **Initialise state** — render the first frame on start.
2. **Read one keystroke (or one line)** at a time, dispatch to a handler that mutates state.
3. **Re-render** the full frame after every action — don't append, replace.
4. **Loop until quit.**
The whole frame should fit on one screen.

### 5. Make it runnable in one command
Add a script to the project's existing task runner. The user should run `go run ./prototype/...` or equivalent — never need to remember a path.

### 6. Hand it over
Give the user the run command. They'll drive it themselves; the interesting moments are when they say "wait, that shouldn't be possible" — those are the bugs in the _idea_, which is the whole point.

### 7. Capture the answer
When the prototype has done its job, the answer to the question is the only thing worth keeping. Leave a `NOTES.md` next to the prototype.

## Anti-patterns
- **Don't add tests.** A prototype that needs tests is no longer a prototype.
- **Don't wire it to the real database.** Use an in-memory store unless the question is specifically about persistence.
- **Don't generalise.** No "what if we wanted to support X later." The prototype answers one question.
- **Don't blur the logic and the TUI together.** Keep the TUI as a thin shell over a pure module.
- **Don't ship the TUI shell into production.** The shell is throwaway. The logic module behind it is the bit worth keeping.