package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"reasonix/internal/boot"
	"reasonix/internal/control"
	"reasonix/internal/event"
)

// GatewayConfig holds configuration for the BotGateway.
type GatewayConfig struct {
	Model         string
	MaxSteps      int
	WorkspaceRoot string
	Channels      map[Platform]ChannelConfig
	Allowlist     AllowlistConfig
	Enabled       map[Platform]bool
	Debounce      time.Duration

	// SessionIdleTimeout is how long a session can be idle before eviction.
	// Zero or negative disables eviction. Default: 30 minutes.
	SessionIdleTimeout time.Duration

	// ModelPrefsPath is the filesystem path to persist per-session model
	// preferences (set by /model). Empty disables persistence.
	ModelPrefsPath string
}

// ChannelConfig overrides gateway defaults for one IM channel.
type ChannelConfig struct {
	Model         string
	WorkspaceRoot string
}

// AllowlistConfig controls which users/groups may use the bot.
type AllowlistConfig struct {
	Enabled  bool
	AllowAll bool
	Users    map[Platform][]string
	Groups   map[Platform][]string
}

// BotGateway is the Reasonix bot message gateway. It manages Controller
// lifecycles, session concurrency, event rendering, and platform adapters.
type BotGateway struct {
	cfg      GatewayConfig
	adapters map[Platform]Adapter
	sessions *SessionManager

	mu             sync.Mutex
	controllers    map[string]*sessionState // session key -> active state
	modelPrefs     map[string]string        // session key -> model override (survives controller recreation)
	modelPrefsPath string                   // filesystem path for persistence; empty = no persistence
	allowlist      map[Platform]map[string]bool
	groupAllowlist map[Platform]map[string]bool

	logger *slog.Logger
}

type sessionState struct {
	ctrl        *control.Controller
	sink        *sessionEventSink
	cancel      context.CancelFunc
	pendingAsks map[string][]event.AskQuestion
	createdAt   time.Time
	lastActive  time.Time
}

type sessionEventSink struct {
	mu     sync.RWMutex
	target event.Sink
}

type pendingReactionAdapter interface {
	AddPendingReaction(ctx context.Context, messageID string) error
}

func (s *sessionEventSink) setTarget(target event.Sink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.target = target
}

func (s *sessionEventSink) Emit(e event.Event) {
	s.mu.RLock()
	target := s.target
	s.mu.RUnlock()
	if target != nil {
		target.Emit(e)
	}
}

// NewGateway creates a new BotGateway.
func NewGateway(cfg GatewayConfig, adapters map[Platform]Adapter, logger *slog.Logger) *BotGateway {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Debounce <= 0 {
		cfg.Debounce = 1500 * time.Millisecond
	}
	gw := &BotGateway{
		cfg:            cfg,
		adapters:       adapters,
		sessions:       NewSessionManager(cfg.Debounce),
		controllers:    make(map[string]*sessionState),
		modelPrefs:     make(map[string]string),
		modelPrefsPath: cfg.ModelPrefsPath,
		allowlist:      make(map[Platform]map[string]bool),
		groupAllowlist: make(map[Platform]map[string]bool),
		logger:         logger.With("component", "bot_gateway"),
	}
	gw.loadModelPrefs()
	gw.buildAllowlist()
	return gw
}

func (gw *BotGateway) buildAllowlist() {
	for _, plat := range []Platform{PlatformQQ, PlatformFeishu, PlatformWeixin, PlatformDiscord} {
		gw.allowlist[plat] = make(map[string]bool)
		if !gw.cfg.Allowlist.Enabled {
			continue
		}
		for _, uid := range gw.cfg.Allowlist.Users[plat] {
			gw.allowlist[plat][uid] = true
		}
		gw.groupAllowlist[plat] = make(map[string]bool)
		for _, gid := range gw.cfg.Allowlist.Groups[plat] {
			gw.groupAllowlist[plat][gid] = true
		}
	}
}

// defaultSessionIdleTimeout is the default idle timeout for bot sessions
// when the config does not set one explicitly.
const defaultSessionIdleTimeout = 30 * time.Minute

// evictionCheckInterval is how often the eviction loop scans for idle sessions.
const evictionCheckInterval = 5 * time.Minute

// Start starts all enabled platform adapters and begins message processing.
func (gw *BotGateway) Start(ctx context.Context) error {
	for plat, adapter := range gw.adapters {
		if !gw.cfg.Enabled[plat] {
			gw.logger.Info("platform disabled, skipping", "platform", plat)
			continue
		}
		gw.logger.Info("starting adapter", "platform", plat)
		if err := adapter.Start(ctx); err != nil {
			return fmt.Errorf("start adapter %s: %w", plat, err)
		}
	}

	// Start session eviction loop if timeout is configured (or use default).
	timeout := gw.cfg.SessionIdleTimeout
	if timeout <= 0 {
		timeout = defaultSessionIdleTimeout
	}
	go gw.evictLoop(ctx, timeout)

	// Merge message channels from all adapters.
	for plat, adapter := range gw.adapters {
		if !gw.cfg.Enabled[plat] {
			continue
		}
		go gw.dispatchLoop(ctx, plat, adapter)
	}

	return nil
}

// Stop stops all adapters and closes every session.
func (gw *BotGateway) Stop() {
	gw.mu.Lock()
	for key, state := range gw.controllers {
		if state.cancel != nil {
			state.cancel()
		}
		state.ctrl.Close()
		delete(gw.controllers, key)
	}
	gw.mu.Unlock()

	for _, adapter := range gw.adapters {
		if err := adapter.Stop(); err != nil {
			gw.logger.Warn("error stopping adapter", "err", err)
		}
	}
}

// evictLoop periodically scans active sessions and removes those idle longer than
// timeout. It runs until ctx is cancelled.
func (gw *BotGateway) evictLoop(ctx context.Context, timeout time.Duration) {
	ticker := time.NewTicker(evictionCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gw.evictIdleSessions(timeout)
		}
	}
}

// evictIdleSessions closes and removes sessions whose lastActive is older than
// now - timeout. Must NOT hold gw.mu.
func (gw *BotGateway) evictIdleSessions(timeout time.Duration) {
	now := time.Now()
	var stale []struct {
		key   string
		state *sessionState
	}

	gw.mu.Lock()
	for key, state := range gw.controllers {
		if now.Sub(state.lastActive) > timeout {
			stale = append(stale, struct {
				key   string
				state *sessionState
			}{key, state})
		}
	}
	// Remove from map under lock.
	for _, s := range stale {
		delete(gw.controllers, s.key)
	}
	gw.mu.Unlock()

	// Close controllers outside the lock to avoid deadlocks.
	for _, s := range stale {
		if s.state.cancel != nil {
			s.state.cancel()
		}
		s.state.ctrl.Close()
		gw.logger.Info("evicted idle session", "session", s.key[:8])
	}
}

func (gw *BotGateway) dispatchLoop(ctx context.Context, plat Platform, adapter Adapter) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-adapter.Messages():
			if !ok {
				return
			}
			gw.handleMessage(ctx, plat, adapter, msg)
		}
	}
}

func (gw *BotGateway) handleMessage(ctx context.Context, plat Platform, adapter Adapter, msg InboundMessage) {
	msg.Platform = plat

	// Allowlist check.
	if !gw.checkAllowlist(plat, msg) {
		gw.logger.Info("user not in allowlist", "platform", plat, "user", hashID(msg.UserID))
		_ = gw.sendText(ctx, adapter, msg, "Sorry, you are not authorized to use this bot.")
		return
	}

	src := msg.Session()
	key := BuildSessionKey(src)

	// Slash command handling.
	if IsSlashBypass(msg.Text) {
		gw.handleSlashCommand(ctx, adapter, key, msg)
		return
	}

	gw.addPendingReaction(ctx, plat, adapter, msg)

	// Session concurrency control.
	acquired, merged := gw.sessions.TryAcquire(key, msg)
	if merged {
		gw.logger.Debug("message merged to pending queue", "session", key[:8])
		return
	}
	if !acquired {
		// Session is busy; message was queued in TryAcquire.
		gw.logger.Debug("session busy, queued", "session", key[:8])
		return
	}

	gw.runTurn(ctx, adapter, key, msg)
}

func (gw *BotGateway) addPendingReaction(ctx context.Context, plat Platform, adapter Adapter, msg InboundMessage) {
	if strings.TrimSpace(msg.MessageID) == "" {
		return
	}
	reactor, ok := adapter.(pendingReactionAdapter)
	if !ok {
		return
	}
	if err := reactor.AddPendingReaction(ctx, msg.MessageID); err != nil {
		gw.logger.Warn("pending reaction failed", "platform", plat, "err", err)
	}
}

func (gw *BotGateway) checkAllowlist(plat Platform, msg InboundMessage) bool {
	if gw.cfg.Allowlist.AllowAll {
		return true
	}
	if !gw.cfg.Allowlist.Enabled {
		return false
	}
	if !gw.allowlist[plat][msg.UserID] {
		return false
	}
	groups := gw.groupAllowlist[plat]
	if chatUsesGroupAllowlist(msg.ChatType) && len(groups) > 0 && !groups[msg.ChatID] {
		return false
	}
	return true
}

func chatUsesGroupAllowlist(chatType ChatType) bool {
	switch chatType {
	case ChatGroup, ChatGuild, ChatThread:
		return true
	default:
		return false
	}
}

func (gw *BotGateway) handleSlashCommand(ctx context.Context, adapter Adapter, key string, msg InboundMessage) {
	switch {
	case strings.HasPrefix(msg.Text, "/stop"):
		gw.mu.Lock()
		state, ok := gw.controllers[key]
		gw.mu.Unlock()
		if ok && state.cancel != nil {
			state.cancel()
		}
		gw.sessions.ForceRelease(key)
		_ = gw.sendText(ctx, adapter, msg, "Stopped current task.")

	case strings.HasPrefix(msg.Text, "/new") || strings.HasPrefix(msg.Text, "/reset"):
		gw.mu.Lock()
		state, ok := gw.controllers[key]
		hasModelPref := gw.modelPrefs[key] != ""
		if ok {
			if state.cancel != nil {
				state.cancel()
			}
			// If a model override is set, drop the old controller so the next
			// turn creates a fresh one with the new model.
			if hasModelPref {
				delete(gw.controllers, key)
			} else {
				if err := state.ctrl.NewSession(); err != nil {
					gw.logger.Warn("new session failed", "err", err)
				}
			}
		}
		gw.mu.Unlock()
		gw.sessions.ForceRelease(key)
		if hasModelPref {
			_ = gw.sendText(ctx, adapter, msg,
				fmt.Sprintf("Started new session (model: %s)", gw.modelPrefs[key]))
		} else {
			_ = gw.sendText(ctx, adapter, msg, "Started new session.")
		}

	case strings.HasPrefix(msg.Text, "/approve"):
		// Parse approval ID from message.
		parts := strings.Fields(msg.Text)
		if len(parts) < 2 {
			_ = gw.sendText(ctx, adapter, msg, "Usage: /approve <id>")
			return
		}
		gw.mu.Lock()
		state, ok := gw.controllers[key]
		gw.mu.Unlock()
		if ok {
			state.ctrl.Approve(parts[1], true, false, false)
			_ = gw.sendText(ctx, adapter, msg, "Approved.")
		}

	case strings.HasPrefix(msg.Text, "/deny"):
		parts := strings.Fields(msg.Text)
		if len(parts) < 2 {
			_ = gw.sendText(ctx, adapter, msg, "Usage: /deny <id>")
			return
		}
		gw.mu.Lock()
		state, ok := gw.controllers[key]
		gw.mu.Unlock()
		if ok {
			state.ctrl.Approve(parts[1], false, false, false)
			_ = gw.sendText(ctx, adapter, msg, "Denied.")
		}

	case strings.HasPrefix(msg.Text, "/answer"):
		parts := strings.Fields(msg.Text)
		if len(parts) < 3 {
			_ = gw.sendText(ctx, adapter, msg, "Usage: /answer <id> <option or q1=option;q2=option>")
			return
		}
		askID := parts[1]
		rawAnswer := strings.TrimSpace(strings.Join(parts[2:], " "))
		gw.mu.Lock()
		state, ok := gw.controllers[key]
		var questions []event.AskQuestion
		if ok {
			questions = state.pendingAsks[askID]
			delete(state.pendingAsks, askID)
		}
		gw.mu.Unlock()
		if !ok || state.ctrl == nil {
			_ = gw.sendText(ctx, adapter, msg, "No active session found.")
			return
		}
		answers := parseAskAnswers(questions, rawAnswer)
		state.ctrl.AnswerQuestion(askID, answers)
		_ = gw.sendText(ctx, adapter, msg, "Answer submitted.")

	case strings.HasPrefix(msg.Text, "/status"):
		active := gw.sessions.ActiveCount()
		gw.mu.Lock()
		sessions := len(gw.controllers)
		gw.mu.Unlock()
		_ = gw.sendText(ctx, adapter, msg, fmt.Sprintf("Active tasks: %d\nRetained sessions: %d", active, sessions))

	case strings.HasPrefix(msg.Text, "/model"):
		modelName := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/model"))
		// Model aliases for convenience
		aliases := map[string]string{
			"flash":    "deepseek-flash",
			"pro":      "deepseek-pro",
			"mimo":     "mimo-pro",
			"deepseek": "deepseek-flash",
		}
		if resolved, ok := aliases[modelName]; ok {
			modelName = resolved
		}
		// Show current model
		if modelName == "" {
			gw.mu.Lock()
			current := gw.cfg.Model
			if pref, ok := gw.modelPrefs[key]; ok {
				current = pref
			}
			gw.mu.Unlock()
			_ = gw.sendText(ctx, adapter, msg,
				fmt.Sprintf("Current model: %s\nAvailable: /model flash | pro | mimo\nUse /new after switching to apply.", current))
			return
		}
		// Store preference — takes effect on next /new (controller recreation)
		gw.mu.Lock()
		gw.modelPrefs[key] = modelName
		gw.mu.Unlock()
		gw.saveModelPrefs()
		_ = gw.sendText(ctx, adapter, msg,
			fmt.Sprintf("Model switched to %s (takes effect after /new)", modelName))

	case strings.HasPrefix(msg.Text, "/goal"):
		goalText := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/goal"))
		// Show current goal status
		if goalText == "" || goalText == "status" {
			gw.mu.Lock()
			state, ok := gw.controllers[key]
			gw.mu.Unlock()
			if ok && state.ctrl.Goal() != "" {
				_ = gw.sendText(ctx, adapter, msg,
					fmt.Sprintf("Current goal: %s\nStatus: %s", state.ctrl.Goal(), state.ctrl.GoalStatus()))
			} else {
				_ = gw.sendText(ctx, adapter, msg,
					"No active goal.\nUse /goal <objective> to start an autonomous task.")
			}
			return
		}
		// Clear goal
		if goalText == "clear" || goalText == "stop" || goalText == "off" || goalText == "done" {
			gw.mu.Lock()
			state, ok := gw.controllers[key]
			gw.mu.Unlock()
			if ok {
				state.ctrl.ClearGoal()
				_ = gw.sendText(ctx, adapter, msg, "Goal cleared.")
			} else {
				_ = gw.sendText(ctx, adapter, msg, "No active session.")
			}
			return
		}
		// Set goal and run the goal loop: release any existing lock, prepare the
		// session, then delegate to runTurn — which calls RunTurn → the controller's
		// built-in continueGoal loop handles autonomous continuation.
		gw.sessions.ForceRelease(key)
		state := gw.getOrCreateSession(ctx, key, msg)
		if state == nil || state.ctrl == nil {
			_ = gw.sendText(ctx, adapter, msg, "Internal error: could not create session.")
			return
		}
		state.ctrl.SetGoal(goalText)
		_ = gw.sendText(ctx, adapter, msg, fmt.Sprintf("🎯 Starting autonomous goal: %s", goalText))
		// Replace message text with the raw goal so the turn knows what to pursue.
		msg.Text = goalText
		gw.runTurn(ctx, adapter, key, msg)
		return

	case strings.HasPrefix(msg.Text, "/help"):
		help := "Available commands:\n" +
			"/stop - Stop current task\n" +
			"/new - Start new session\n" +
			"/reset - Reset session\n" +
			"/model [flash|pro|mimo] - Switch model\n" +
			"/goal <objective> - Start autonomous goal\n" +
			"/goal status - Show goal status\n" +
			"/goal clear - Clear goal\n" +
			"/approve <id> - Approve action\n" +
			"/deny <id> - Deny action\n" +
			"/answer <id> <option> - Answer question\n" +
			"/status - Show status\n" +
			"/help - Show help"
		_ = gw.sendText(ctx, adapter, msg, help)
	}
}

func (gw *BotGateway) runTurn(ctx context.Context, adapter Adapter, key string, msg InboundMessage) {
	defer func() {
		// Check for queued messages after the turn completes.
		next := gw.sessions.Release(key)
		if next != nil {
			gw.runTurn(ctx, adapter, key, *next)
			return
		}
	}()

	// Build the input text. In groups, prepend the sender's name.
	input := msg.Text
	if msg.ChatType == ChatGroup {
		input = fmt.Sprintf("[%s] %s", msg.UserName, msg.Text)
	}

	// Get or create the Controller.
	state := gw.getOrCreateSession(ctx, key, msg)
	if state == nil || state.ctrl == nil {
		_ = gw.sendText(ctx, adapter, msg, "Internal error: could not create session.")
		return
	}

	// Send "user is typing" indicator.
	_ = adapter.SendTyping(ctx, msg.ChatID)

	// Create an event render sink for this turn.
	sink := newRenderSink(ctx, adapter, msg.ChatID, msg.ChatType, msg.MessageID, gw.logger, func(ask event.Ask) {
		gw.mu.Lock()
		if state.pendingAsks == nil {
			state.pendingAsks = make(map[string][]event.AskQuestion)
		}
		state.pendingAsks[ask.ID] = ask.Questions
		gw.mu.Unlock()
	})
	state.sink.setTarget(sink)
	defer state.sink.setTarget(nil)

	// Create a cancellable context for this turn.
	turnCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	gw.mu.Lock()
	state.cancel = cancel
	state.lastActive = time.Now()
	gw.mu.Unlock()

	// Run one turn of the conversation.
	sink.ctrl = state.ctrl
	err := state.ctrl.RunTurn(turnCtx, input)
	sink.Emit(event.Event{Kind: event.TurnDone, Err: err})
	if err != nil {
		gw.logger.Warn("turn error", "session", key[:8], "err", err)
	}
}

func (gw *BotGateway) getOrCreateSession(ctx context.Context, key string, msg InboundMessage) *sessionState {
	gw.mu.Lock()
	if state, ok := gw.controllers[key]; ok {
		state.lastActive = time.Now()
		gw.mu.Unlock()
		return state
	}
	gw.mu.Unlock()

	// Resolve the base model: platform-level ChannelConfig overrides gateway default.
	model, workspaceRoot := gw.sessionOptionsForPlatform(msg.Platform)

	// Per-session model preference (set by /model) takes highest priority.
	gw.mu.Lock()
	if pref, ok := gw.modelPrefs[key]; ok {
		model = pref
	}
	gw.mu.Unlock()

	// Create a new Controller.
	sessionSink := &sessionEventSink{}
	ctrl, err := boot.Build(ctx, boot.Options{
		Model:         model,
		MaxSteps:      gw.cfg.MaxSteps,
		RequireKey:    true,
		Sink:          sessionSink,
		WorkspaceRoot: workspaceRoot,
	})
	if err != nil {
		gw.logger.Error("build controller failed", "err", err)
		return nil
	}
	ctrl.EnableInteractiveApproval()

	gw.mu.Lock()
	gw.controllers[key] = &sessionState{
		ctrl:        ctrl,
		sink:        sessionSink,
		pendingAsks: make(map[string][]event.AskQuestion),
		createdAt:   time.Now(),
		lastActive:  time.Now(),
	}
	state := gw.controllers[key]
	gw.mu.Unlock()

	return state
}

func (gw *BotGateway) sessionOptionsForPlatform(plat Platform) (model string, workspaceRoot string) {
	model = gw.cfg.Model
	workspaceRoot = gw.cfg.WorkspaceRoot
	if gw.cfg.Channels == nil {
		return model, workspaceRoot
	}
	channel, ok := gw.cfg.Channels[plat]
	if !ok {
		return model, workspaceRoot
	}
	if value := strings.TrimSpace(channel.Model); value != "" {
		model = value
	}
	if value := strings.TrimSpace(channel.WorkspaceRoot); value != "" {
		workspaceRoot = value
	}
	return model, workspaceRoot
}

func (gw *BotGateway) sendText(ctx context.Context, adapter Adapter, msg InboundMessage, text string) error {
	_, err := adapter.Send(ctx, OutboundMessage{
		ChatID:       msg.ChatID,
		ChatType:     msg.ChatType,
		Text:         text,
		ReplyToMsgID: msg.MessageID,
	})
	return err
}

func parseAskAnswers(questions []event.AskQuestion, raw string) []event.AskAnswer {
	raw = strings.TrimSpace(raw)
	if len(questions) == 0 {
		return []event.AskAnswer{{Selected: []string{raw}}}
	}
	byID := make(map[string]*event.AskQuestion, len(questions))
	for i := range questions {
		q := &questions[i]
		byID[q.ID] = q
		byID[fmt.Sprintf("%d", i+1)] = q
	}
	answerMap := make(map[string][]string, len(questions))
	if strings.Contains(raw, "=") {
		for _, part := range strings.Split(raw, ";") {
			k, v, ok := strings.Cut(part, "=")
			if !ok {
				continue
			}
			q := byID[strings.TrimSpace(k)]
			if q == nil {
				continue
			}
			answerMap[q.ID] = normalizeAskSelection(*q, strings.TrimSpace(v))
		}
	} else if len(questions) == 1 {
		answerMap[questions[0].ID] = normalizeAskSelection(questions[0], raw)
	}
	out := make([]event.AskAnswer, 0, len(questions))
	for _, q := range questions {
		out = append(out, event.AskAnswer{QuestionID: q.ID, Selected: answerMap[q.ID]})
	}
	return out
}

func normalizeAskSelection(q event.AskQuestion, raw string) []string {
	parts := []string{raw}
	if q.Multi && strings.Contains(raw, ",") {
		parts = strings.Split(raw, ",")
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx, err := strconv.Atoi(part); err == nil && idx >= 1 && idx <= len(q.Options) {
			out = append(out, q.Options[idx-1].Label)
			continue
		}
		out = append(out, part)
	}
	return out
}

// saveModelPrefs persists modelPrefs to a JSON file so preferences survive
// restarts. If no path is configured it is a no-op.
func (gw *BotGateway) saveModelPrefs() {
	if gw.modelPrefsPath == "" {
		return
	}
	// Copy under lock so marshal + write don't hold the mutex across I/O.
	gw.mu.Lock()
	prefsCopy := make(map[string]string, len(gw.modelPrefs))
	for k, v := range gw.modelPrefs {
		prefsCopy[k] = v
	}
	gw.mu.Unlock()

	data, err := json.Marshal(prefsCopy)
	if err != nil {
		gw.logger.Error("bot: marshal model prefs", "err", err)
		return
	}
	if err := os.WriteFile(gw.modelPrefsPath, data, 0644); err != nil {
		gw.logger.Error("bot: save model prefs", "err", err)
	}
}

// loadModelPrefs loads persisted model preferences from disk.
func (gw *BotGateway) loadModelPrefs() {
	if gw.modelPrefsPath == "" {
		return
	}
	data, err := os.ReadFile(gw.modelPrefsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			gw.logger.Error("bot: load model prefs", "err", err)
		}
		return
	}
	var prefs map[string]string
	if err := json.Unmarshal(data, &prefs); err != nil {
		gw.logger.Error("bot: unmarshal model prefs", "err", err)
		return
	}
	gw.mu.Lock()
	for k, v := range prefs {
		gw.modelPrefs[k] = v
	}
	gw.mu.Unlock()
	gw.logger.Info("bot: loaded model prefs", "count", len(prefs))
}

// ModelPrefsFilePath returns the canonical path for persisting per-session model
// preferences (~/.config/reasonix/bot-model-prefs.json). Returns "" if the OS
// config directory cannot be resolved.
func ModelPrefsFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "reasonix", "bot-model-prefs.json")
}
