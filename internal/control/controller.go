// Package control is the transport-agnostic session driver. A Controller owns
// the agent run loop and session lifecycle, takes commands (Send/Cancel/Approve/
// SetPlanMode/Compact/NewSession/…), and emits everything that happens —
// reasoning, tool calls, approvals, turn completion — as a typed event stream to
// a single event.Sink.
//
// The point is one orchestration layer behind every frontend: a terminal TUI, a
// desktop webview, or an HTTP/SSE server each drive the Controller identically
// (issue commands, render events) and none of them re-implement turn lifecycle,
// cancellation, or approval. The Controller depends on no frontend.
package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/checkpoint"
	"reasonix/internal/command"
	"reasonix/internal/config"
	"reasonix/internal/diff"
	"reasonix/internal/event"
	"reasonix/internal/hook"
	"reasonix/internal/jobs"
	"reasonix/internal/memory"
	"reasonix/internal/mesh"
	"reasonix/internal/nilutil"
	"reasonix/internal/permission"
	"reasonix/internal/plugin"
	"reasonix/internal/sandbox"
	"reasonix/internal/scheduler"
	"reasonix/internal/skill"
	"reasonix/internal/tool"
)

// ErrTurnRunning reports that a caller tried to start a second foreground turn
// while one is already active in the same Controller.
var ErrTurnRunning = errors.New("turn already running")

// Controller drives one chat session. Construct with New; drive with the command
// methods; observe through the Sink passed in Options.
type Controller struct {
	runner   agent.Runner
	executor *agent.Agent
	sink     event.Sink
	policy   permission.Policy

	label             string
	systemPrompt      string
	sessionDir        string
	host              *plugin.Host
	commands          []command.Command
	skills            []skill.Skill
	allSkills         []skill.Skill
	skillStore        *skill.Store
	allSkillStore     *skill.Store
	hooks             *hook.Runner // session hook runner; nil-safe (no hooks configured)
	mem               *memory.Set
	cleanup           func()
	schedule          *scheduler.Scheduler // cron task scheduler (nil = disabled)
	mesh              *mesh.Mesh          // agent-to-agent MCP delegation (nil = disabled)
	autoPlan          string
	reasoningLanguage string
	// disableColdResumePrune skips stale-tool-result elision on cold resume.
	// Zero value keeps the prune on (the cheaper default).
	disableColdResumePrune bool
	shell                  sandbox.Shell // interpreter for user-invoked "!" commands; zero = auto
	classifier             autoPlanClassifier
	startedOnce            bool                             // guards the one-shot SessionStart hook on first turn
	onRemember             func(rule string) RememberResult // set via Options; invoked when user picks "always allow"

	// balanceURL/balanceKey target the active provider's optional wallet-balance
	// endpoint (empty when the provider declares none). Captured at build so a
	// model/key switch — which rebuilds the controller — refreshes them.
	balanceURL    string
	balanceKey    string
	balanceClient *http.Client

	// jobs is the session-scoped background-job manager. The agent's background
	// tools spawn into it; Compose drains its completion notes into the next turn;
	// Close cancels its still-running jobs.
	jobs *jobs.Manager

	// reg is the live tool registry the executor reads each turn; pluginCtx is the
	// session-scoped context a hot-added stdio server binds its subprocess to.
	// Together they let AddMCPServer connect a server mid-session and have its tools
	// available on the next turn (see AddMCPServer / RemoveMCPServer).
	reg       *tool.Registry
	pluginCtx context.Context

	// Checkpoints (snapshot-based rewind). cp is the per-session store rebound when
	// the session path changes; cpRoot is the workspace root used to guard restore
	// writes. cpTurn is the monotonic turn counter (decoupled from the store so it
	// never collides after a restructure); cpBound[turn] records len(Session.Messages)
	// at that turn's start — the truncation boundary for a conversation rewind/fork.
	// Boundaries are persisted in each checkpoint and rebuilt from the store on
	// resume (so a reopened session can still rewind conversation / fork), but
	// dropped after a summarize restructures the log so those operations report
	// "unavailable" rather than mis-truncating; code rewind (file-based) is unaffected.
	cp      *checkpoint.Store
	cpRoot  string
	cpTurn  int
	cpBound map[int]int

	// promptMu serialises approval and ask prompts so at most one user decision is
	// outstanding at a time (parallel read-only tool calls don't normally gate,
	// writers run serially — but this keeps the contract explicit). Held across
	// the blocking wait, so it must never be taken by the Approve/Answer paths.
	promptMu sync.Mutex

	// mu guards the run state and approval bookkeeping; every critical section
	// under it is short and non-blocking.
	mu          sync.Mutex
	cancel      context.CancelFunc
	running     bool
	autosaveWG  sync.WaitGroup
	planMode    bool
	goal        string
	goalStatus  string
	goalTurns   int
	goalBlocks  int
	goalBlock   string
	sessionPath string
	approvals   map[string]pendingApproval
	asks        map[string]pendingAsk
	granted     map[string]bool
	nextID      int
	// turn counts model turns this session, passed to hooks in their payload.
	turn int
	// approvedPlanAutoApproveTools auto-allows writer tool calls without prompting.
	// Set only while executing a just-approved plan: approving the plan is the
	// go-ahead, so the model shouldn't re-prompt for every write of the work it
	// just got cleared to do. Deny rules still bite (those never reach the
	// approver). Reset when the execution turn returns.
	approvedPlanAutoApproveTools bool

	// toolApprovalMode is the runtime approval posture for permission-gated tool
	// calls. "ask" prompts by default, "auto" lets the policy auto-approve the
	// writer fallback while preserving ask/deny rules, and "yolo" skips every
	// tool approval prompt except plan approval. It never answers AskRequest.
	toolApprovalMode string

	// autoApproveTools is "YOLO/full access" mode: while set, every tool approval
	// request is auto-allowed for the rest of the session (writers and bash run
	// without asking). It is a deliberate, session-scoped opt-in (the
	// --dangerously-skip-permissions flag or a runtime toggle), never persisted.
	// Deny rules are unaffected — they're resolved before the approver, so a
	// denied tool is still blocked. It never answers AskRequest or plan approval:
	// those remain user decisions.
	autoApproveTools bool

	// pendingMemory holds memory notes added mid-session (via "#" quick-add or a
	// memory edit) that haven't yet been folded into a turn. Compose drains it
	// onto the next outgoing turn — never into the cache-stable system prefix — so
	// a fresh memory takes effect this session without busting the prompt cache;
	// it joins the prefix naturally on the next session.
	pendingMemory []string

	displayRecorder func(content, display string)
}

type approvalReply struct {
	allow   bool
	session bool
	persist bool // true = write "always allow" rule to config
}

type pendingApproval struct {
	tool      string
	subject   string
	autoDrain bool
	reply     chan approvalReply
}

// pendingAsk is an in-flight ask question batch. questions is retained so the
// AskRequest can be re-emitted to a frontend that reconnected after the original
// event (see ReplayPendingPrompts).
type pendingAsk struct {
	questions []event.AskQuestion
	reply     chan []event.AskAnswer
}

const (
	ToolApprovalAsk  = "ask"
	ToolApprovalAuto = "auto"
	ToolApprovalYolo = "yolo"
)

const (
	memoryRememberTool = "remember"
	memoryForgetTool   = "forget"
)

const (
	maxGoalAutoTurns = 50
	goalContinueTurn = "Continue pursuing the active goal. If it is complete, provide the concise final result and end with [goal:complete]. If it is truly blocked on a user-owned decision after trying sensible defaults, end with [goal:blocked:<short reason>]. Otherwise do the next useful work and end with [goal:continue]."
)

// RememberResult describes what happened when an approval rule was persisted.
type RememberResult struct {
	Rule      string
	Path      string
	Saved     bool
	CoveredBy string
	Err       error
}

// Options carries the already-built pieces setup assembles. Lifecycle metadata
// lets the controller mint and rotate session files; Host/Commands are surfaced
// to frontends that resolve MCP prompts and slash commands.
type Options struct {
	Runner        agent.Runner
	Executor      *agent.Agent
	Sink          event.Sink
	Policy        permission.Policy
	Label         string
	SystemPrompt  string
	SessionDir    string
	SessionPath   string
	Host          *plugin.Host
	Commands      []command.Command
	Skills        []skill.Skill
	AllSkills     []skill.Skill
	SkillStore    *skill.Store
	AllSkillStore *skill.Store
	Hooks         *hook.Runner
	Memory        *memory.Set
	Cleanup       func()
	// BalanceURL/BalanceKey wire the active provider's optional wallet-balance
	// endpoint and bearer key; empty when the provider declares no balance_url.
	BalanceURL    string
	BalanceKey    string
	BalanceClient *http.Client
	// Jobs is the session-scoped background-job manager (nil disables background jobs).
	Jobs *jobs.Manager
	// Registry is the executor's live tool set, and PluginCtx the session-scoped
	// context; both are needed for hot-adding MCP servers via AddMCPServer.
	Registry  *tool.Registry
	PluginCtx context.Context
	// WorkspaceRoot is the project root checkpoint restores are confined to ("" =
	// no confinement). Frontends pass the cwd they launched the session in.
	WorkspaceRoot string
	AutoPlan      string
	// ReasoningLanguage controls visible reasoning language preference. Empty/auto
	// means no transient injection because the stable language policy already
	// follows the conversation language.
	ReasoningLanguage string
	// DisableColdResumePrune skips the stale-tool-result elision that otherwise
	// runs when a session resumes past the provider cache window. Zero value
	// keeps the prune on (the cheaper default).
	DisableColdResumePrune bool
	// Shell is the interpreter user-invoked "!" commands run under, so /shell
	// matches the agent's configured [tools.shell] choice. Zero value = auto.
	Shell      sandbox.Shell
	Classifier autoPlanClassifier
	// OnRemember, when set, is invoked with a new allow rule the user chose to
	// persist to disk (e.g. "Bash(go test:*)"). The callback is wired into the
	// permission Gate on EnableInteractiveApproval.
	OnRemember func(rule string) RememberResult
	// Schedule, when non-nil, starts the cron scheduler goroutine on New().
	Schedule *scheduler.Scheduler
	// ScheduleConfig, when non-empty, builds a scheduler in New() that sends tasks
	// back through the controller. The scheduler is started as a background goroutine.
	ScheduleConfig *scheduler.Config
}

// New builds a Controller. A nil Sink is replaced with event.Discard.
func New(opts Options) *Controller {
	sink := opts.Sink
	if nilutil.IsNil(sink) {
		sink = event.Discard
	}
	classifier := opts.Classifier
	if nilutil.IsNil(classifier) {
		classifier = nil
	}
	pluginCtx := opts.PluginCtx
	if pluginCtx == nil {
		pluginCtx = context.Background()
	}
	c := &Controller{
		runner:                 opts.Runner,
		executor:               opts.Executor,
		sink:                   sink,
		policy:                 opts.Policy,
		label:                  opts.Label,
		systemPrompt:           opts.SystemPrompt,
		sessionDir:             opts.SessionDir,
		sessionPath:            opts.SessionPath,
		host:                   opts.Host,
		commands:               opts.Commands,
		skills:                 opts.Skills,
		allSkills:              opts.AllSkills,
		skillStore:             opts.SkillStore,
		allSkillStore:          opts.AllSkillStore,
		hooks:                  opts.Hooks,
		mem:                    opts.Memory,
		cleanup:                opts.Cleanup,
		autoPlan:               normalizeAutoPlan(opts.AutoPlan),
		reasoningLanguage:      config.NormalizeReasoningLanguage(opts.ReasoningLanguage),
		disableColdResumePrune: opts.DisableColdResumePrune,
		shell:                  opts.Shell,
		classifier:             classifier,
		onRemember:             opts.OnRemember,
		schedule:               opts.Schedule,
		balanceURL:             opts.BalanceURL,
		balanceKey:             opts.BalanceKey,
		balanceClient:          opts.BalanceClient,
		jobs:                   opts.Jobs,
		reg:                    opts.Registry,
		pluginCtx:              pluginCtx,
		cpRoot:                 opts.WorkspaceRoot,
		toolApprovalMode:       ToolApprovalAsk,
		approvals:              map[string]pendingApproval{},
		asks:                   map[string]pendingAsk{},
		granted:                map[string]bool{},
	}
	// Checkpoints: bind a store to the session and route writer pre-edits into it.
	c.rebindCheckpoints(opts.SessionPath)
	c.setActiveJobSession(opts.SessionPath)
	if c.executor != nil {
		c.executor.SetPreEditHook(func(ch diff.Change) {
			if c.cp != nil {
				c.cp.Snapshot(ch)
			}
		})
		c.executor.SetMemoryQueue(c)
	}
	// Start cron scheduler if configured.
	if c.schedule != nil {
		go c.schedule.Start(context.Background())
	}
	// Build scheduler from config if provided (no pre-built one).
	if c.schedule == nil && opts.ScheduleConfig != nil {
		ctrl := c // capture for closure
		c.schedule = scheduler.New(*opts.ScheduleConfig, scheduler.SenderFunc(func(ctx context.Context, text string) error {
			ctrl.Send(text)
			return nil
		}), slog.Default())
		if c.schedule != nil {
			go c.schedule.Start(context.Background())
		}
	}
	return c
}

// SetDisplayRecorder installs an optional hook used by frontends that persist a
// Approve answers a pending ApprovalRequest by ID: allow runs the call, session
// also remembers a grant for the rest of the session so the same approval scope
// is not re-prompted. Unknown/expired IDs are ignored.
func (c *Controller) Approve(id string, allow, session, persist bool) {
	c.mu.Lock()
	pending := c.approvals[id]
	delete(c.approvals, id)
	c.mu.Unlock()
	if pending.reply != nil {
		pending.reply <- approvalReply{allow: allow, session: session, persist: persist} // buffered, never blocks
	}
}

// EnableInteractiveApproval swaps the executor's gate for one that routes
// approval decisions to the frontend via ApprovalRequest events, and wires the
// controller in as the executor's Asker so the `ask` tool can question the user.
// Interactive frontends (chat, desktop) call this; the headless run keeps the
// silent gate and a nil asker from setup.
func (c *Controller) EnableInteractiveApproval() {
	if c.executor != nil {
		c.executor.SetGate(c.newInteractiveGate())
		c.executor.SetAsker(c)
	}
}

func normalizeToolApprovalMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ToolApprovalAuto, "approve", "allow":
		return ToolApprovalAuto
	case ToolApprovalYolo, "full", "full-access", "bypass":
		return ToolApprovalYolo
	default:
		return ToolApprovalAsk
	}
}

func (c *Controller) newInteractiveGate() *permission.Gate {
	policy := c.policy
	c.mu.Lock()
	mode := normalizeToolApprovalMode(c.toolApprovalMode)
	c.mu.Unlock()
	switch mode {
	case ToolApprovalAuto, ToolApprovalYolo:
		policy.Mode = permission.Allow
	default:
		policy.Mode = permission.Ask
	}
	policy.Ask = append(policy.Ask,
		permission.Rule{Tool: memoryRememberTool},
		permission.Rule{Tool: memoryForgetTool},
	)
	gate := permission.NewGate(policy, gateApprover{c})
	gate.OnRemember = func(rule string) {
		if c.onRemember != nil {
			_ = c.onRemember(rule)
		}
	}
	return gate
}

func (c *Controller) refreshInteractiveGate() {
	if c.executor != nil {
		c.executor.SetGate(c.newInteractiveGate())
	}
}

// Steer queues mid-turn guidance without interrupting the in-flight request.
func (c *Controller) Steer(text string) {
	c.mu.Lock()
	exec := c.executor
	running := c.running
	c.mu.Unlock()
	if exec == nil {
		return
	}
	if running {
		exec.Steer(text)
		return
	}
	// Agent not running — frontend's runningRef was stale.
	// Convert to a new turn so the user gets a response.
	go func() { c.SubmitDisplay(text, text) }()
}

// SteerConsumed returns true when the steer queue is empty after the last consume.
func (c *Controller) SteerConsumed() bool {
	c.mu.Lock()
	exec := c.executor
	c.mu.Unlock()
	if exec != nil {
		return exec.SteerConsumed()
	}
	return true
}

// Ask implements agent.Asker: it emits an AskRequest and blocks until
// AnswerQuestion(ID, …) answers or ctx is cancelled. promptMu serialises it
// against tool-approval prompts so at most one user prompt is outstanding.
// Unlike tool-approval gates, Ask is NOT bypassed in YOLO mode — the `ask`
// tool exists to get a genuine user decision, and YOLO only auto-approves
// tool calls; it must not answer the user's questions for them.
func (c *Controller) Ask(ctx context.Context, questions []event.AskQuestion) ([]event.AskAnswer, error) {
	c.promptMu.Lock()
	defer c.promptMu.Unlock()

	c.mu.Lock()
	c.nextID++
	id := strconv.Itoa(c.nextID)
	reply := make(chan []event.AskAnswer, 1)
	c.asks[id] = pendingAsk{questions: questions, reply: reply}
	c.mu.Unlock()

	c.sink.Emit(event.Event{Kind: event.AskRequest, Ask: event.Ask{ID: id, Questions: questions}})

	select {
	case ans := <-reply:
		return ans, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.asks, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

// AnswerQuestion resolves a pending AskRequest by ID with the user's selections.
// Unknown/expired IDs are ignored.
func (c *Controller) AnswerQuestion(id string, answers []event.AskAnswer) {
	c.mu.Lock()
	pending, ok := c.asks[id]
	delete(c.asks, id)
	c.mu.Unlock()
	if ok {
		pending.reply <- answers // buffered, never blocks
	}
}

// ReplayPendingPrompts re-emits the ApprovalRequest / AskRequest event for every
// prompt currently blocking the run loop. A frontend that reconnected or reloaded
// after the original event has no way to rebuild its approval/ask modal otherwise,
// so the blocked gate goroutine stays stuck forever while the session shows a
// "waiting" status with no actionable prompt. promptMu serialises Ask and
// requestApproval, so in practice at most one prompt is outstanding; the loops
// stay general so a future concurrent prompt would still replay correctly.
func (c *Controller) ReplayPendingPrompts() {
	c.mu.Lock()
	approvals := make([]event.Approval, 0, len(c.approvals))
	for id, p := range c.approvals {
		approvals = append(approvals, event.Approval{ID: id, Tool: p.tool, Subject: p.subject})
	}
	asks := make([]event.Ask, 0, len(c.asks))
	for id, p := range c.asks {
		asks = append(asks, event.Ask{ID: id, Questions: p.questions})
	}
	c.mu.Unlock()
	for _, a := range approvals {
		c.sink.Emit(event.Event{Kind: event.ApprovalRequest, Approval: a})
	}
	for _, a := range asks {
		c.sink.Emit(event.Event{Kind: event.AskRequest, Ask: a})
	}
}

// SetPlanMode flips the executor's read-only gate without touching the
// cache-stable prompt prefix, and remembers the state so Compose can prepend the
// plan-mode marker to outgoing turns.
func (c *Controller) SetPlanMode(v bool) {
	c.mu.Lock()
	c.planMode = v
	c.mu.Unlock()
	if c.executor != nil {
		c.executor.SetPlanMode(v)
	}
}

// SetAutoPlan updates the interactive auto-plan gate for subsequent turns.
func (c *Controller) SetAutoPlan(mode string) {
	c.mu.Lock()
	c.autoPlan = normalizeAutoPlan(mode)
	c.mu.Unlock()
}

// SetReasoningLanguage updates the visible reasoning language preference for
// subsequent turns.
func (c *Controller) SetReasoningLanguage(lang string) {
	mode := config.NormalizeReasoningLanguage(lang)
	c.mu.Lock()
	c.reasoningLanguage = mode
	c.mu.Unlock()
	if setter, ok := c.runner.(interface{ SetReasoningLanguage(string) }); ok {
		setter.SetReasoningLanguage(mode)
	} else if c.executor != nil {
		c.executor.SetReasoningLanguage(mode)
	}
}

// PlanMode reports whether outgoing turns currently receive the plan-mode
// marker. Frontends use it after Compose because auto-plan may flip the mode.
func (c *Controller) PlanMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.planMode
}

// SetGoal stores a session-scoped active goal. Compose injects it into outgoing
// user turns, not the system prompt or tool schema, so it does not disturb the
// cache-stable prefix.
func (c *Controller) SetGoal(goal string) {
	goal = strings.TrimSpace(goal)
	c.mu.Lock()
	defer c.mu.Unlock()
	if goal == "" {
		c.goal = ""
		c.goalStatus = GoalStatusStopped
		c.goalTurns = 0
		c.goalBlocks = 0
		c.goalBlock = ""
		return
	}
	if c.goal == goal && c.goalStatus == GoalStatusRunning {
		return
	}
	c.goal = goal
	c.goalStatus = GoalStatusRunning
	c.goalTurns = 0
	c.goalBlocks = 0
	c.goalBlock = ""
}

func (c *Controller) ClearGoal() {
	c.SetGoal("")
}

func (c *Controller) Goal() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.goal
}

func (c *Controller) GoalStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.TrimSpace(c.goal) == "" && c.goalStatus == "" {
		return GoalStatusStopped
	}
	if c.goalStatus == "" {
		return GoalStatusStopped
	}
	return c.goalStatus
}

// GoalTurns returns the number of turns spent on the current goal.
func (c *Controller) GoalTurns() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.goalTurns
}

// GoalBlocks returns the consecutive block count for the current goal.
func (c *Controller) GoalBlocks() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.goalBlocks
}

// Compact runs one compaction pass on the executor's session on demand.
// instructions is optional `/compact <focus>` guidance steering what to keep.
func (c *Controller) Compact(ctx context.Context, instructions string) error {
	if c.executor == nil {
		return nil
	}
	return c.executor.CompactNow(ctx, instructions)
}

// maybeSessionStart fires the SessionStart hook exactly once per session, lazily
// on the first turn — by then the sink/notify is wired, and a resumed session
// fires it too (its first post-resume turn).
func (c *Controller) maybeSessionStart(ctx context.Context) {
	c.mu.Lock()
	if c.startedOnce {
		c.mu.Unlock()
		return
	}
	c.startedOnce = true
	c.mu.Unlock()
	c.hooks.SessionStart(ctx)
}

// NewSession snapshots the current conversation, rotates to a fresh file, and
// resets the executor to a clean session carrying the same system prompt. It
// ends the old session and starts the new one for lifecycle hooks.
func (c *Controller) NewSession() error {
	if c.executor == nil {
		return nil
	}
	if err := c.Snapshot(); err != nil {
		return err
	}
	c.hooks.SessionEnd(context.Background())
	if c.sessionDir != "" {
		c.mu.Lock()
		c.sessionPath = agent.NewSessionPath(c.sessionDir, c.label)
		c.mu.Unlock()
	}
	c.setActiveJobSession(c.SessionPath())
	c.executor.SetSession(agent.NewSession(c.systemPrompt))
	c.rebindCheckpoints(c.SessionPath())
	c.mu.Lock()
	c.startedOnce = true // NewSession fires SessionStart itself; don't re-fire on the next turn
	c.mu.Unlock()
	c.hooks.SessionStart(context.Background())
	return nil
}

// ClearSession discards the current conversation without preserving it in
// resume/history, then rotates to a clean session carrying the same system prompt.
func (c *Controller) ClearSession() error {
	if c.executor == nil {
		return nil
	}
	c.mu.Lock()
	running := c.running
	oldPath := c.sessionPath
	c.mu.Unlock()
	if running {
		return fmt.Errorf("cannot clear while a turn is running")
	}
	destroy := c.BeginDestroySession(oldPath)
	if !destroy.Async {
		if err := removeSessionArtifacts(oldPath); err != nil {
			destroy.Finish()
			return err
		}
		destroy.Finish()
	}
	c.hooks.SessionEnd(context.Background())
	if c.sessionDir != "" {
		c.mu.Lock()
		c.sessionPath = agent.NewSessionPath(c.sessionDir, c.label)
		c.mu.Unlock()
	}
	c.setActiveJobSession(c.SessionPath())
	c.executor.SetSession(agent.NewSession(c.systemPrompt))
	c.rebindCheckpoints(c.SessionPath())
	c.mu.Lock()
	c.startedOnce = true
	c.mu.Unlock()
	c.hooks.SessionStart(context.Background())
	if destroy.Async {
		go func() {
			destroy.Wait()
			if err := removeSessionArtifacts(oldPath); err != nil {
				c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: "clear session cleanup failed: " + err.Error()})
			}
			destroy.Finish()
		}()
	}
	return nil
}

func removeSessionArtifacts(path string) error {
	if path == "" {
		return nil
	}
	for _, p := range []string{path, agent.BranchMetaPath(path)} {
		if p == "" {
			continue
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if dir := ckptDir(path); dir != "" {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := agent.DeleteSubagentsByParent(filepath.Dir(path), agent.BranchID(path)); err != nil {
		return err
	}
	return nil
}

// SummarizeFrom compresses the conversation from turn onward into one summary;
// SummarizeUpTo compresses everything before it. Both are Claude Code's "summarize
// from/up to here" — they restructure the message log (keeping code untouched), so
// afterwards the per-turn boundaries no longer map and conversation rewind/fork
// report "unavailable" until new turns rebuild them (code rewind, file-based, is
// unaffected). Refused while a turn runs; need the live boundary.
func (c *Controller) SummarizeFrom(ctx context.Context, turn int) error {
	return c.summarizeAt(ctx, turn, true)
}

func (c *Controller) SummarizeUpTo(ctx context.Context, turn int) error {
	return c.summarizeAt(ctx, turn, false)
}

func (c *Controller) summarizeAt(ctx context.Context, turn int, from bool) error {
	if c.executor == nil {
		return c.rewindFail(fmt.Errorf("checkpoints unavailable"))
	}
	c.mu.Lock()
	running := c.running
	boundary, hasBound := c.cpBound[turn]
	c.mu.Unlock()
	if running {
		return c.rewindFail(fmt.Errorf("cannot summarize while a turn is running"))
	}
	if !hasBound {
		return c.rewindFail(fmt.Errorf("summarize unavailable for turn %d (resumed session)", turn))
	}
	var err error
	if from {
		err = c.executor.SummarizeFrom(ctx, boundary)
	} else {
		err = c.executor.SummarizeUpTo(ctx, boundary)
	}
	if err != nil {
		return c.rewindFail(err)
	}
	// The log was restructured; existing boundaries no longer map. Drop them (keep
	// cpTurn monotonic so new turns don't collide with the store) — conversation
	// rewind degrades to "unavailable" until fresh turns rebuild boundaries.
	c.mu.Lock()
	c.cpBound = map[int]int{}
	c.mu.Unlock()
	if err := c.Snapshot(); err != nil {
		slog.Warn("controller: post-summarize snapshot", "err", err)
	}
	return nil
}

// Resume seeds the session from a loaded transcript and pins the active file to
// its path so auto-save keeps appending there.
func (c *Controller) Resume(s *agent.Session, path string) {
	if c.executor != nil {
		c.executor.SetSession(s)
	}
	c.mu.Lock()
	c.sessionPath = path
	c.mu.Unlock()
	c.setActiveJobSession(path)
	c.rebindCheckpoints(path)
	c.maybeColdResumePrune(path)
}

// cacheColdAfter approximates how long the provider keeps a prompt prefix
// cached. A session idle longer than this resumes against a cold cache, so a
// history rewrite at that moment costs no extra cache misses — it only shrinks
// the full-price first request. Deliberately conservative: too small burns a
// live cache (~4× the miss tokens, measured), too large only forgoes a prune.
// Tighten from benchmarks/cache-ttl-probe data, never below measured retention.
var cacheColdAfter = 24 * time.Hour

// maybeColdResumePrune elides stale tool results when a resumed session has
// been idle past the provider's cache retention, then persists the pruned
// transcript so the saved file and the prompt stay in sync.
func (c *Controller) maybeColdResumePrune(path string) {
	if c.disableColdResumePrune || c.executor == nil || path == "" {
		return
	}
	// Idle time comes from branch meta only — every session the controller has
	// ever snapshotted carries one. A meta-less transcript (e.g. a legacy import
	// not yet saved) skips the prune until its first snapshot creates the meta.
	m, ok, err := agent.LoadBranchMeta(path)
	if err != nil || !ok || m.UpdatedAt.IsZero() {
		return
	}
	last := m.UpdatedAt
	if time.Since(last) < cacheColdAfter {
		return
	}
	st, err := c.executor.PruneStaleToolResults()
	if err != nil || st.Results == 0 {
		return
	}
	c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(
		"resumed after %s idle (provider cache expired) — elided %d stale tool results to cheapen the cold restart",
		time.Since(last).Round(time.Minute), st.Results)})
	if err := c.Snapshot(); err != nil {
		slog.Warn("controller: post-prune snapshot", "err", err)
	}
}

// Snapshot writes the executor's conversation to the active session file. No-op
// when persistence is unavailable or the session has never been used (no user
// interaction). Called after every turn so a crash loses at most one in-flight
// prompt.
func (c *Controller) Snapshot() error {
	return c.snapshot(false)
}

// SnapshotActivity writes the active conversation and marks the session as
// recently active. Use it only after a real user/model turn changes the
// transcript; switch/close snapshots should call Snapshot so they do not reorder
// recent-session pickers.
func (c *Controller) SnapshotActivity() error {
	return c.snapshot(true)
}

// midTurnSnapshotInterval is atomic (nanoseconds) so a test shrinking it
// cannot race a previous test's still-parking autosave goroutine.
var midTurnSnapshotInterval atomic.Int64

func init() { midTurnSnapshotInterval.Store(int64(30 * time.Second)) }

// autosaveWhileRunning snapshots the session periodically while a turn runs,
// so an abrupt kill (SSH drop, force-quit) loses at most one interval of a
// long turn instead of all of it (#3772). Session.Save copies under the lock
// and replaces the file atomically, so racing the turn's appends is safe.
func (c *Controller) autosaveWhileRunning(ctx context.Context) {
	t := time.NewTicker(time.Duration(midTurnSnapshotInterval.Load()))
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.snapshot(false); err != nil {
				slog.Warn("controller: mid-turn snapshot", "err", err)
			}
		}
	}
}

func (c *Controller) snapshot(markActivity bool) error {
	c.mu.Lock()
	path := c.sessionPath
	c.mu.Unlock()
	if c.executor == nil || path == "" {
		return nil
	}
	s := c.executor.Session()
	if !s.HasContent() {
		return nil
	}
	if !markActivity {
		if _, err := agent.EnsureBranchMeta(path); err != nil {
			return err
		}
	}
	// Stamp aggregate statistics from the Agent onto the Session before saving,
	// so the sidecar .meta file captures accurate cumulative totals.
	if c.executor != nil {
		hit, miss := c.executor.SessionCache()
		curr := "¥"
		if p := c.executor.Pricing(); p != nil {
			curr = p.Symbol()
		}
		s.SetMeta(
			c.executor.SessionTokensIn(),
			c.executor.SessionTokensOut(),
			c.executor.SessionTurns(),
			hit,
			miss,
			c.executor.SessionCost(),
			curr,
		)
	}
	if err := s.Save(path); err != nil {
		return err
	}
	if markActivity {
		return agent.TouchBranchMeta(path)
	}
	return nil
}

func (c *Controller) messageCount() int {
	if c.executor == nil {
		return 0
	}
	return len(c.executor.Session().Snapshot())
}

func (c *Controller) snapshotActivityIfChanged(startMessages int) {
	if c.messageCount() <= startMessages {
		return
	}
	if err := c.SnapshotActivity(); err != nil {
		slog.Warn("controller: activity snapshot", "err", err)
	}
}

// SetSessionPath pins where auto-save lands (a fresh session file minted by the
// caller when no resume path applies).
func (c *Controller) SetSessionPath(p string) {
	c.mu.Lock()
	c.sessionPath = p
	c.mu.Unlock()
	c.setActiveJobSession(p)
	c.rebindCheckpoints(p)
}

// SessionDestroyHandle separates waiting for cancelled jobs from ending the
// destroy window, so callers can move/delete persistent artifacts in between.
type SessionDestroyHandle struct {
	Wait   func()
	Finish func()
	Async  bool
}

// BeginDestroySession marks a session as leaving active use and cancels its
// background jobs. Call Wait before moving/deleting artifacts, then Finish after
// persistent cleanup/move work is complete.
func (c *Controller) BeginDestroySession(sessionPath string) SessionDestroyHandle {
	parentSession := agent.BranchID(sessionPath)
	if c.jobs == nil || parentSession == "" {
		noop := func() {}
		return SessionDestroyHandle{Wait: noop, Finish: noop}
	}
	done := c.jobs.DestroySession(parentSession)
	return SessionDestroyHandle{
		Wait: func() {
			for _, ch := range done {
				<-ch
			}
		},
		Finish: func() {
			c.jobs.FinishDestroySession(parentSession)
		},
		Async: len(done) > 0,
	}
}

// IsDestroyingSession reports whether sessionPath is currently in the destroy
// window for this controller's job manager.
func (c *Controller) IsDestroyingSession(sessionPath string) bool {
	if c.jobs == nil {
		return false
	}
	return c.jobs.IsDestroying(agent.BranchID(sessionPath))
}

func (c *Controller) setActiveJobSession(sessionPath string) {
	if c.jobs != nil {
		c.jobs.SetActiveSession(agent.BranchID(sessionPath))
	}
}

// SessionDir reports the directory new session files land in ("" disables
// InheritLifecycleFrom carries same-session lifecycle state across controller
// rebuilds, such as model switches that preserve the conversation.
func (c *Controller) InheritLifecycleFrom(prev *Controller) {
	if prev == nil {
		return
	}
	prev.mu.Lock()
	started := prev.startedOnce
	turn := prev.turn
	prev.mu.Unlock()

	c.mu.Lock()
	c.startedOnce = started
	if c.turn < turn {
		c.turn = turn
	}
	c.mu.Unlock()
}

// ReleaseResources stops plugin subprocesses and releases resources without
// firing SessionEnd. Use it only when replacing the controller for the same
// logical session.
func (c *Controller) ReleaseResources() {
	c.close(false)
}

// Close stops plugin subprocesses and releases resources. A session that ever
// started fires SessionEnd so a teardown hook runs.
func (c *Controller) Close() {
	c.close(true)
}

func (c *Controller) close(fireSessionEnd bool) {
	c.mu.Lock()
	started := c.startedOnce
	c.mu.Unlock()
	if fireSessionEnd && started {
		c.hooks.SessionEnd(context.Background())
	}
	if c.jobs != nil {
		c.jobs.Close() // cancel any still-running background jobs
	}
	if c.cleanup != nil {
		c.cleanup()
	}
}

// Jobs returns the still-running background jobs for the status bar (nil when
// background jobs are disabled).
func (c *Controller) Jobs() []jobs.View {
	if c.jobs == nil {
		return nil
	}
	return c.jobs.RunningForSession(c.parentSessionID())
}

// SetToolApprovalMode changes the runtime approval posture for permission-gated
// tools. It does not answer business asks or plan approval.
func (c *Controller) SetToolApprovalMode(mode string) {
	mode = normalizeToolApprovalMode(mode)
	var pending []chan approvalReply

	c.mu.Lock()
	c.toolApprovalMode = mode
	c.autoApproveTools = mode == ToolApprovalYolo
	switch mode {
	case ToolApprovalAuto:
		pending = c.drainApprovalsLocked(false)
	case ToolApprovalYolo:
		pending = c.drainApprovalsLocked(true)
	}
	c.mu.Unlock()

	c.refreshInteractiveGate()
	for _, reply := range pending {
		reply <- approvalReply{allow: true}
	}
}

func (c *Controller) ToolApprovalMode() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return normalizeToolApprovalMode(c.toolApprovalMode)
}

// SetAutoApproveTools turns YOLO/full-access mode on or off for the session:
// while on, every tool approval request is auto-allowed (writers and bash run
// without asking). Ask requests and plan approval still reach the user. Deny
// rules still block. Runtime-only — never written to config.
func (c *Controller) SetAutoApproveTools(on bool) {
	if on {
		c.SetToolApprovalMode(ToolApprovalYolo)
		return
	}
	c.SetToolApprovalMode(ToolApprovalAsk)
}

// SetBypass is the legacy name for SetAutoApproveTools. Keep it for existing
// desktop/serve bindings and CLI code that still uses the bypass wording.
func (c *Controller) SetBypass(on bool) {
	c.SetAutoApproveTools(on)
}

// SetMode applies plan (read-only) and tool auto-approval together so a turn
// submitted right after a composer mode switch can't observe a half-applied
// gate. Turning tool auto-approval on drains any pending tool approval.
func (c *Controller) SetMode(plan, autoApproveTools bool) {
	c.mu.Lock()
	c.planMode = plan
	c.mu.Unlock()

	if c.executor != nil {
		c.executor.SetPlanMode(plan)
	}
	if autoApproveTools {
		c.SetToolApprovalMode(ToolApprovalYolo)
	} else {
		c.SetToolApprovalMode(ToolApprovalAsk)
	}
}

// drainApprovalsLocked removes every pending approval gate and returns their
// reply channels; caller holds c.mu and sends {allow:true} after unlocking.
func (c *Controller) drainApprovalsLocked(includeExplicitAsk bool) []chan approvalReply {
	pending := make([]chan approvalReply, 0, len(c.approvals))
	for id, approval := range c.approvals {
		if requiresFreshApprovalTool(approval.tool) {
			continue
		}
		if !includeExplicitAsk && !approval.autoDrain {
			continue
		}
		delete(c.approvals, id)
		pending = append(pending, approval.reply)
	}
	return pending
}

// AutoApproveTools reports whether YOLO/full-access tool auto-approval is on,
// for status indicators and mode persistence.
func (c *Controller) AutoApproveTools() bool {
	return c.ToolApprovalMode() == ToolApprovalYolo
}

// Bypass is the legacy name for AutoApproveTools.
func (c *Controller) Bypass() bool {
	return c.AutoApproveTools()
}


func (c *Controller) profileListText() string {
	cfg, err := config.Load()
	if err != nil {
		return "profile: " + err.Error()
	}
	if len(cfg.Profiles) == 0 {
		return "no harness profiles configured. Add [profiles.<name>] blocks to reasonix.toml or ~/.config/reasonix/config.toml."
	}
	var b strings.Builder
	active := cfg.ActiveProfile
	fmt.Fprintf(&b, "Harness profiles%s:\n", "")
	for name, p := range cfg.Profiles {
		mark := " "
		if name == active {
			mark = "*"
		}
		desc := p.Description
		if desc == "" {
			desc = p.Model
		}
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(&b, " %s %s — %s\n", mark, name, desc)
	}
	b.WriteString("switch with /profile <name>")
	return b.String()
}

// ApplyProfile activates a named harness profile and persists the choice.
// Settings from the profile (model, effort, tool-approval-mode, auto-plan,
// output-style) are applied immediately. Pass "" to deactivate.
func (c *Controller) ApplyProfile(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	name = strings.TrimSpace(name)

	if name != "" {
		p, ok := cfg.Profiles[name]
		if !ok {
			return fmt.Errorf("unknown profile %q; available: %s", name, profileNames(cfg.Profiles))
		}
		// Apply profile settings to the live session.
		if p.Model != "" {
			cfg.ActiveProfile = name
			cfg.DefaultModel = p.Model
		}
		if p.Effort != "" {
			cfg.Agent.PlannerModel = "" // profile overrides planner too
		}
		if p.ToolApproveMode != "" {
			c.SetToolApprovalMode(p.ToolApproveMode)
		}
		if p.AutoPlan != "" {
			c.SetAutoPlan(p.AutoPlan)
		}
		if p.OutputStyle != "" {
			cfg.Agent.OutputStyle = p.OutputStyle
		}
		cfg.ActiveProfile = name
	} else {
		cfg.ActiveProfile = ""
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if name != "" {
		c.notice("profile: switched to " + name + " — model: " + cfg.DefaultModel +
			", approve: " + c.ToolApprovalMode() +
			", auto-plan: " + c.PlanModeStr())
	} else {
		c.notice("profile: deactivated (using default settings)")
	}
	return nil
}

// PlanModeStr returns "on" or "off" for display.
func (c *Controller) PlanModeStr() string {
	if c.PlanMode() {
		return "on"
	}
	return "off"
}

// profileNames returns a sorted, comma-joined string of profile names.
func profileNames(profiles map[string]config.ProfileConfig) string {
	names := make([]string, 0, len(profiles))
	for n := range profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
