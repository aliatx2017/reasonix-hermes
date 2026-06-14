// Package control is the transport-agnostic session driver.
package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"reasonix/internal/checkpoint"
	"reasonix/internal/config"
	"reasonix/internal/event"
	"reasonix/internal/evidence"
	"reasonix/internal/i18n"
	"reasonix/internal/memory"
	"reasonix/internal/permission"
	"reasonix/internal/provider"
)

// gateApprover adapts the Controller to permission.Approver. It is distinct
// from the public Approve command (different signature, different direction).
type gateApprover struct{ c *Controller }

func (g gateApprover) Approve(ctx context.Context, tool, subject string, args json.RawMessage) (bool, bool, error) {
	subject = approvalDisplaySubject(tool, subject, args)
	// Auto-allow without prompting while executing a just-approved plan (the plan
	// was the approval), during an explicit continuation turn of that approved
	// plan, or while YOLO/full-access tool auto-approval is on. Deny rules already
	// bit before this point, so they still block.
	g.c.mu.Lock()
	auto := g.c.approvalBypassAllowsLocked(tool)
	g.c.mu.Unlock()
	if auto {
		return true, false, nil
	}
	return g.c.requestApproval(ctx, tool, subject, args)
}

func approvalDisplaySubject(tool, subject string, args json.RawMessage) string {
	switch tool {
	case memoryRememberTool:
		return rememberApprovalSubject(subject, args)
	case memoryForgetTool:
		return forgetApprovalSubject(subject, args)
	case "move_file":
		return moveApprovalSubject(subject, args)
	default:
		return subject
	}
}

func moveApprovalSubject(fallback string, args json.RawMessage) string {
	if len(args) == 0 {
		return fallback
	}
	var in struct {
		SourcePath      string `json:"source_path"`
		DestinationPath string `json:"destination_path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return fallback
	}
	if in.SourcePath == "" || in.DestinationPath == "" {
		return fallback
	}
	return in.SourcePath + " -> " + in.DestinationPath
}

func rememberApprovalSubject(fallback string, args json.RawMessage) string {
	if len(args) == 0 {
		return fallback
	}
	var in struct {
		Name        string `json:"name"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Body        string `json:"body"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return fallback
	}
	name := approvalCompactText(firstNonEmpty(in.Name, in.Title))
	desc := approvalTruncate(approvalCompactText(in.Description), 180)
	body := approvalTruncate(approvalCompactText(in.Body), 240)
	typ := string(memory.NormalizeType(in.Type))

	var b strings.Builder
	b.WriteString("Save/update memory")
	if name != "" {
		fmt.Fprintf(&b, " %q", name)
	}
	if typ != "" {
		fmt.Fprintf(&b, " [%s]", typ)
	}
	if desc != "" {
		b.WriteString(": ")
		b.WriteString(desc)
	}
	if body != "" {
		if desc == "" {
			b.WriteString(": ")
		} else {
			b.WriteString(" | ")
		}
		b.WriteString("body: ")
		b.WriteString(body)
	}
	if b.Len() == len("Save/update memory") && fallback != "" {
		return fallback
	}
	return b.String()
}

func forgetApprovalSubject(fallback string, args json.RawMessage) string {
	if len(args) == 0 {
		return fallback
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return fallback
	}
	name := approvalCompactText(in.Name)
	if name == "" {
		return fallback
	}
	return fmt.Sprintf("Archive memory %q", name)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func approvalCompactText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func approvalTruncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

type seedTodo struct {
	Content string `json:"content"`
	Status  string `json:"status"`
	Level   int    `json:"level,omitempty"`
}

// seedPlanTodos turns an approved plan into a starter task list and emits it as a
// synthetic todo_write event, so the live task panel populates the instant the
// user approves — a structural guarantee, not a prompt the model might ignore.
// The model still flips item status as it works (only it knows its own
// progress); this just makes the list exist. No-op when the plan has no list.
func (c *Controller) seedPlanTodos(plan string) string {
	args := PlanTodosJSON(plan)
	if args == "" {
		return ""
	}
	t := event.Tool{ID: "plan-seed", Name: "todo_write", Args: args, ReadOnly: true}
	c.sink.Emit(event.Event{Kind: event.ToolDispatch, Tool: t})
	t.Output = "task list seeded from the approved plan"
	c.sink.Emit(event.Event{Kind: event.ToolResult, Tool: t})
	c.seedAgentTodoState(args)
	return args
}

func (c *Controller) seedAgentTodoState(args string) {
	if c.executor == nil {
		return
	}
	todos := agentTodoStateFromArgs(args)
	if len(todos) == 0 {
		return
	}
	c.executor.SeedTodoState(todos)
}

func (c *Controller) completePlanTodos(args string) {
	if args == "" {
		return
	}
	done := completedPlanTodosJSON(args)
	if done == "" {
		return
	}
	t := event.Tool{ID: "plan-seed", Name: "todo_write", Args: done, ReadOnly: true}
	c.sink.Emit(event.Event{Kind: event.ToolDispatch, Tool: t})
	t.Output = "approved plan finished"
	c.sink.Emit(event.Event{Kind: event.ToolResult, Tool: t})
	c.replaceAgentTodoState(done)
}

func (c *Controller) replaceAgentTodoState(args string) {
	if c.executor == nil {
		return
	}
	todos := agentTodoStateFromArgs(args)
	if len(todos) == 0 {
		return
	}
	c.executor.ReplaceTodoState(todos)
}

func agentTodoStateFromArgs(args string) []evidence.TodoItem {
	var payload struct {
		Todos []evidence.TodoItem `json:"todos"`
	}
	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return nil
	}
	return payload.Todos
}

// PlanTodosJSON parses an approved plan's markdown into todo_write-shaped args
// JSON ({"todos":[...]}), or "" when the plan has no list items. The exit_plan_mode
// path seeds via seedPlanTodos (an event); a frontend whose own approval flow
// bypasses exit_plan_mode (the chat TUI's text-plan approval) calls this directly
// to render the same starter checklist. Shared parsing keeps the two consistent.
func PlanTodosJSON(plan string) string {
	items := parsePlanTodos(plan)
	if len(items) == 0 {
		return ""
	}
	b, err := json.Marshal(map[string]any{"todos": items})
	if err != nil {
		return ""
	}
	return string(b)
}

func completedPlanTodosJSON(args string) string {
	var p struct {
		Todos []seedTodo `json:"todos"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil || len(p.Todos) == 0 {
		return ""
	}
	for i := range p.Todos {
		p.Todos[i].Status = "completed"
	}
	b, err := json.Marshal(map[string]any{"todos": p.Todos})
	if err != nil {
		return ""
	}
	return string(b)
}

// parsePlanTodos extracts a starter task list from an approved plan's markdown
// list items (bulleted or numbered): the first is in_progress, the rest pending,
// capped so a long plan can't flood the panel. It understands ONLY markdown lists
// — an unambiguous, standard structure — and deliberately does not guess at prose,
// tables, or arrow sequences (those need brittle, language-specific heuristics).
// The plan-mode marker steers the model to present its plan as a list, so this
// catches the normal case; anything it misses is covered by the model's own
// todo_write calls as it executes.
func parsePlanTodos(plan string) []seedTodo {
	var todos []seedTodo
	for _, raw := range strings.Split(plan, "\n") {
		item, level, ok := listItem(raw)
		if !ok {
			continue
		}
		status := "pending"
		if len(todos) == 0 {
			status = "in_progress"
		}
		todos = append(todos, seedTodo{Content: item, Status: status, Level: level})
		if len(todos) >= 20 {
			break
		}
	}
	return todos
}

func (c *Controller) sessionMessageCount() int {
	if c.executor == nil {
		return 0
	}
	return len(c.executor.Session().Messages)
}

// hasTodoUpdateSince reports whether the model emitted its own todo_write after
// index start, so the seeded plan todos aren't auto-completed over the model's
// own bookkeeping.
func (c *Controller) hasTodoUpdateSince(start int) bool {
	if c.executor == nil {
		return false
	}
	msgs := c.executor.Session().Messages
	if start < 0 || start > len(msgs) {
		start = len(msgs)
	}
	_, ok := latestTodoArgsSince(msgs, start)
	return ok
}

func latestTodoArgsSince(msgs []provider.Message, start int) (string, bool) {
	for i := len(msgs) - 1; i >= start; i-- {
		for j := len(msgs[i].ToolCalls) - 1; j >= 0; j-- {
			tc := msgs[i].ToolCalls[j]
			if tc.Name == "todo_write" {
				return tc.Arguments, true
			}
		}
	}
	return "", false
}

// listItem parses a markdown list line ("- x", "* x", "1. x", "2) x") into its
// task text and a nesting level derived from leading indentation (0 for a
// top-level item, 1 for an indented sub-step — capped at 1 since the plan is
// two-level). ok is false when the line isn't a list item. Light inline-markdown
// stripping keeps the checklist readable.
func listItem(line string) (content string, level int, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return "", 0, false
	}
	indent := 0
	for _, c := range line[:len(line)-len(trimmed)] {
		if c == '\t' {
			indent += 4
		} else {
			indent++
		}
	}
	s := trimmed
	// A numbered markdown heading ("### 1. Add the loader") is how models often
	// write a phase even when asked for a list; strip the heading marker and
	// treat it as a top-level phase. A heading without a number (a section
	// title like "## Plan") falls through and is ignored.
	heading := false
	if h := strings.TrimLeft(s, "#"); h != s && strings.HasPrefix(h, " ") {
		heading = true
		s = strings.TrimSpace(h)
	}
	switch {
	case strings.HasPrefix(s, "- "), strings.HasPrefix(s, "* "), strings.HasPrefix(s, "+ "):
		s = s[2:]
	default:
		// numbered: leading digits, then "." or ")", then a space
		i := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == 0 || i+1 >= len(s) || (s[i] != '.' && s[i] != ')') || s[i+1] != ' ' {
			return "", 0, false
		}
		s = s[i+2:]
	}
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[ ] ")
	s = strings.TrimPrefix(s, "[x] ")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, false
	}
	if heading {
		return s, 0, true // a heading is always a top-level phase
	}
	if indent >= 2 {
		return s, 1, true
	}
	return s, 0, true
}

// parseRewind parses the arguments after "/rewind". The user may provide:
//
//	/rewind              → latest checkpoint, both
//	/rewind <turn>       → that turn, both
//	/rewind <turn> <scope> → that turn, code|conversation|both
//
// If no turn is given, the latest checkpoint is used. If no scope is given, Both is assumed.
func parseRewind(args string, cps []checkpoint.Meta) (int, RewindScope, error) {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		if len(cps) == 0 {
			return 0, RewindBoth, fmt.Errorf("no checkpoints available")
		}
		return cps[len(cps)-1].Turn, RewindBoth, nil
	}
	turn, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, RewindBoth, fmt.Errorf("invalid turn: %w", err)
	}
	scope := RewindBoth
	if len(fields) >= 2 {
		switch strings.ToLower(fields[1]) {
		case "code":
			scope = RewindCode
		case "conversation":
			scope = RewindConversation
		case "both":
			scope = RewindBoth
		default:
			return 0, RewindBoth, fmt.Errorf("unknown scope %q", fields[1])
		}
	}
	return turn, scope, nil
}

// requestApproval emits an ApprovalRequest and blocks until Approve(ID, …)
// answers or ctx is cancelled. A prior session grant for the same approval scope
// short-circuits. promptMu serialises outstanding prompts.
func (c *Controller) requestApproval(ctx context.Context, tool, subject string, args json.RawMessage) (bool, bool, error) {
	c.mu.Lock()
	// YOLO/full access and the just-approved-plan execution window auto-allow
	// approval-gated tools without prompting. Plan approval is a user decision,
	// not a tool permission, so it deliberately stays interactive.
	if c.approvalBypassAllowsLocked(tool) || c.sessionGrantAllowsLocked(tool, subject) {
		c.mu.Unlock()
		return true, false, nil
	}
	c.mu.Unlock()

	c.promptMu.Lock()
	defer c.promptMu.Unlock()

	// Re-check the grant: a session grant may have landed while we queued behind
	// another prompt for the same subject.
	c.mu.Lock()
	if c.approvalBypassAllowsLocked(tool) || c.sessionGrantAllowsLocked(tool, subject) {
		c.mu.Unlock()
		return true, false, nil
	}
	c.nextID++
	id := strconv.Itoa(c.nextID)
	reply := make(chan approvalReply, 1)
	c.approvals[id] = pendingApproval{tool: tool, subject: subject, autoDrain: c.autoApprovalWouldAllowLocked(tool, subject), reply: reply}
	c.mu.Unlock()

	c.sink.Emit(event.Event{Kind: event.ApprovalRequest, Approval: event.Approval{ID: id, Tool: tool, Subject: subject}})
	if hookSubject, hookArgs, ok := permissionRequestHookPayload(tool, subject, args); ok {
		go c.hooks.PermissionRequest(ctx, tool, hookSubject, hookArgs)
	}
	// The agent now needs the user's attention; a Notification hook can ping an
	// external channel (desktop notice, phone) while the run blocks on the reply.
	go c.hooks.Notification(ctx, approvalNotificationText(tool, subject))

	select {
	case r := <-reply:
		// Plan approvals are one-shot — never persist a session grant for them, or
		// every future plan would auto-approve.
		if r.allow && r.session && !requiresFreshApprovalTool(tool) {
			rule := permission.SessionGrantRuleForScope(tool, subject)
			c.mu.Lock()
			c.granted[rule] = true
			c.mu.Unlock()
		}
		if r.allow && r.persist && !requiresFreshApprovalTool(tool) && c.onRemember != nil {
			c.emitRememberResult(c.onRemember(permission.RememberRuleForScope(tool, subject)))
		}
		return r.allow, false, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.approvals, id)
		c.mu.Unlock()
		return false, false, ctx.Err()
	}
}

func approvalNotificationText(tool, subject string) string {
	if requiresFreshApprovalTool(tool) {
		return "approval needed: " + tool
	}
	if subject == "" {
		return "approval needed: " + tool
	}
	return "approval needed: " + tool + " " + subject
}

func permissionRequestHookPayload(tool, subject string, args json.RawMessage) (string, json.RawMessage, bool) {
	switch tool {
	case planApprovalTool:
		return "", nil, false
	case memoryRememberTool, memoryForgetTool:
		return "", nil, true
	default:
		return subject, args, true
	}
}

func (c *Controller) approvalBypassAllowsLocked(tool string) bool {
	if requiresFreshApprovalTool(tool) {
		return false
	}
	return c.toolApprovalMode == ToolApprovalYolo ||
		c.approvedPlanAutoApproveTools
}

func (c *Controller) autoApprovalWouldAllowLocked(tool, subject string) bool {
	if requiresFreshApprovalTool(tool) {
		return false
	}
	policy := c.policy
	policy.Mode = permission.Allow
	return policy.DecideSubject(tool, false, subject) == permission.Allow
}

func (c *Controller) sessionGrantAllowsLocked(tool, subject string) bool {
	if requiresFreshApprovalTool(tool) {
		return false
	}
	for rule := range c.granted {
		if permission.RuleMatchesString(rule, tool, subject) {
			return true
		}
	}
	return false
}

func requiresFreshApprovalTool(tool string) bool {
	switch tool {
	case planApprovalTool, memoryRememberTool, memoryForgetTool:
		return true
	default:
		return false
	}
}

func (c *Controller) emitRememberResult(r RememberResult) {
	if r.Err != nil {
		c.sink.Emit(event.Event{
			Kind:  event.Notice,
			Level: event.LevelWarn,
			Text:  fmt.Sprintf(i18n.M.PermissionSaveFailedFmt, r.Rule, r.Err),
		})
		return
	}
	switch {
	case r.Saved:
		c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(i18n.M.PermissionSavedFmt, r.Path, r.Rule)})
	case strings.TrimSpace(r.CoveredBy) != "":
		c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(i18n.M.PermissionAlreadyAllowedFmt, r.Path, r.CoveredBy)})
	}
}

// ToggleSound turns the completion-sound setting on or off and persists the change
// to the user config file. Returns true on success (setting changed and saved).
func (c *Controller) ToggleSound(on bool) bool {
	cfg, err := config.Load()
	if err != nil {
		c.notice("sound: " + err.Error())
		return false
	}
	cfg.Notifications.Sound = on
	if err := cfg.Save(); err != nil {
		c.notice("sound: save failed: " + err.Error())
		return false
	}
	if on {
		c.notice("completion sound on — bell chimes when each turn finishes")
	} else {
		c.notice("completion sound off")
	}
	return true
}

// beep writes an ASCII bell character to stdout when the completion-sound config
// flag is enabled. Called after each turn completes.
func (c *Controller) beep() {
	cfg, err := config.Load()
	if err != nil {
		return
	}
	if cfg.Notifications.Sound {
		fmt.Print("\a")
	}
}
