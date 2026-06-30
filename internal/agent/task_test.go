package agent

import (
	"context"
	"errors"
<<<<<<< HEAD
	"os"
=======
	"path/filepath"
>>>>>>> upstream/main-v2
	"strings"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
	toolbuiltin "reasonix/internal/tool/builtin"
)

func testTaskContext() context.Context {
	return WithParentSession(context.Background(), "parent-session")
}

// TestTaskToolReturnsSubAgentFinalAnswer runs a task against a mock provider
// that emits a single text turn, and verifies the tool returns that text with a
// transcript reference — sub-agent intermediate state isn't supposed to leak.
func TestTaskToolReturnsSubAgentFinalAnswer(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "found 3 callers of Foo"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	task := newTestTaskTool(t, sub, parentReg, "test-sys-prompt", "", "", nil)

	out, err := task.Execute(testTaskContext(), []byte(`{"prompt":"find callers of Foo"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	_ = subagentRefFromOutput(t, out)
	if !strings.Contains(out, "found 3 callers of Foo") {
		t.Errorf("got %q, want sub-agent final answer", out)
	}
	if !strings.Contains(out, "To continue this same subagent transcript in a later call, pass this ref as `continue_from`. Start a fresh subagent when the next task is independent.") {
		t.Errorf("got %q, want continuation guidance", out)
	}

	// The sub-agent must have received the prompt as its user message and
	// the configured system prompt at the top — proving the session was
	// fresh, not the parent's.
	if sys := sub.lastReq.Messages[0]; sys.Role != provider.RoleSystem || sys.Content != "test-sys-prompt" {
		t.Errorf("first message = %+v, want system 'test-sys-prompt'", sys)
	}
	if got := lastUser(sub.lastReq); got != "find callers of Foo" {
		t.Errorf("sub-agent user = %q, want the prompt verbatim", got)
	}
}

func TestTaskToolCancelDuringStuckProviderReturnsPromptly(t *testing.T) {
	task := newTestTaskTool(t, stuckStreamProvider{}, tool.NewRegistry(), "sys", "", "", nil)

	ctx, cancel := context.WithCancel(testTaskContext())
	done := make(chan error, 1)
	go func() {
		_, err := task.Execute(ctx, []byte(`{"prompt":"wait on stuck provider"}`))
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Execute returned nil after context cancellation")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Execute error = %v, want context cancellation", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("TaskTool.Execute did not return promptly after cancellation")
	}
}

func TestTaskToolSchemaExposesOnlyContinueFromForPersistence(t *testing.T) {
	task := NewTaskTool(&mockProvider{name: "sub"}, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil)
	schema := string(task.Schema())
	if !strings.Contains(schema, `"continue_from"`) {
		t.Fatalf("task schema = %s, want continue_from", schema)
	}
	if strings.Contains(schema, "fork_from") {
		t.Fatalf("task schema = %s, want no fork_from", schema)
	}
}

func TestParallelTasksSchemaDoesNotExposePersistentContinuation(t *testing.T) {
	task := NewTaskTool(&mockProvider{name: "sub"}, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil)
	parallel := NewParallelTasksTool(task, tool.NewRegistry())
	schema := string(parallel.Schema())
	if strings.Contains(schema, "continue_from") || strings.Contains(schema, "fork_from") {
		t.Fatalf("parallel_tasks schema = %s, want no persistent continuation fields", schema)
	}
}

func TestTaskToolInheritsReasoningLanguageFromContext(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "done"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)

	ctx := WithReasoningLanguagePreference(testTaskContext(), "zh")
	if _, err := task.Execute(ctx, []byte(`{"prompt":"inspect auth"}`)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := lastUser(sub.lastReq)
	if !strings.HasPrefix(got, "<reasoning-language>") || !strings.Contains(got, "Simplified Chinese") || !strings.HasSuffix(got, "inspect auth") {
		t.Fatalf("sub-agent user = %q, want reasoning-language-prefixed prompt", got)
	}
}

// TestTaskToolFiltersTools verifies the whitelist behaviour: when the caller
// names a subset of tools, the sub-agent's registry contains exactly that set
// with subagent/skill meta-tools stripped to prevent recursive delegation.
func TestTaskToolFiltersTools(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "read_file", readOnly: true})
	parentReg.Add(fakeTool{name: "write_file", readOnly: false})
	parentReg.Add(fakeTool{name: "bash", readOnly: false})
	task := newTestTaskTool(t, sub, parentReg, "sys", "", "", nil)
	parentReg.Add(task) // simulate the wiring in cli.setup
	parentReg.Add(fakeTool{name: "run_skill", readOnly: false})
	parentReg.Add(fakeTool{name: "read_only_skill", readOnly: true})
	parentReg.Add(fakeTool{name: "research", readOnly: false})

	args := []byte(`{"prompt":"x","tools":["read_file","task","write_file","run_skill","read_only_skill","research"]}`)
	if _, err := task.Execute(testTaskContext(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// The sub-agent's tool schemas should reflect the whitelist minus meta-tools.
	got := map[string]bool{}
	for _, s := range sub.lastReq.Tools {
		got[s.Name] = true
	}
	if !got["read_file"] || !got["write_file"] || got["task"] || got["run_skill"] || got["read_only_skill"] || got["research"] || got["bash"] {
		t.Errorf("sub-agent tools = %v, want {read_file, write_file} (meta-tools stripped, bash not requested)", got)
	}
}

// TestTaskToolDefaultsToParentToolsWithoutMetaTools covers the no-whitelist
// path: the sub-agent inherits parent tools except subagent/skill meta-tools.
func TestTaskToolDefaultsToParentToolsWithoutMetaTools(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "read_file", readOnly: true})
	parentReg.Add(fakeTool{name: "grep", readOnly: true})
	task := newTestTaskTool(t, sub, parentReg, "sys", "", "", nil)
	parentReg.Add(task)
	parentReg.Add(fakeTool{name: "run_skill", readOnly: false})
	parentReg.Add(fakeTool{name: "read_only_skill", readOnly: true})
	parentReg.Add(fakeTool{name: "explore", readOnly: false})
	parentReg.Add(fakeTool{name: "research", readOnly: false})
	parentReg.Add(fakeTool{name: "review", readOnly: false})
	parentReg.Add(fakeTool{name: "security_review", readOnly: false})
	parentReg.Add(fakeTool{name: "remember", readOnly: false})

	if _, err := task.Execute(testTaskContext(), []byte(`{"prompt":"x"}`)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := map[string]bool{}
	for _, s := range sub.lastReq.Tools {
		got[s.Name] = true
	}
	if !got["read_file"] || !got["grep"] || !got["remember"] ||
		got["task"] || got["run_skill"] || got["read_only_skill"] || got["explore"] || got["research"] || got["review"] || got["security_review"] {
		t.Errorf("default sub-agent tools = %v, want normal tools inherited and meta-tools stripped", got)
	}
}

func TestTaskToolUsesConfiguredProfileForExecution(t *testing.T) {
	parent := &mockProvider{name: "parent", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "parent answer"},
		{Type: provider.ChunkDone},
	}}
	resolved := &mockProvider{name: "resolved", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "resolved answer"},
		{Type: provider.ChunkDone},
	}}
	var gotModel, gotEffort string
	resolve := func(model, effort string) (provider.Provider, *provider.Pricing, int, error) {
		gotModel, gotEffort = model, effort
		return resolved, nil, 0, nil
	}
	task := newTestTaskTool(t, parent, tool.NewRegistry(), "sys", "deepseek-pro", "max", resolve)

	out, err := task.Execute(testTaskContext(), []byte(`{"prompt":"x"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "resolved answer") {
		t.Fatalf("sub-agent did not use resolved provider, got %q", out)
	}
	if gotModel != "deepseek-pro" || gotEffort != "max" {
		t.Fatalf("resolved profile = %q/%q, want deepseek-pro/max", gotModel, gotEffort)
	}
}

func TestTaskToolReturnsProfileResolutionErrors(t *testing.T) {
	parent := &mockProvider{name: "parent", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "parent answer"},
		{Type: provider.ChunkDone},
	}}
	resolve := func(string, string) (provider.Provider, *provider.Pricing, int, error) {
		return nil, nil, 0, errors.New("bad effort")
	}
	task := newTestTaskTool(t, parent, tool.NewRegistry(), "sys", "", "", resolve)

	_, err := task.Execute(testTaskContext(), []byte(`{"prompt":"x","effort":"turbo"}`))
	if err == nil || !strings.Contains(err.Error(), "bad effort") {
		t.Fatalf("Execute error = %v, want profile resolution error", err)
	}
}

func TestTaskToolRequiresTranscriptStore(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := NewTaskTool(sub, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil)

	_, err := task.Execute(testTaskContext(), []byte(`{"prompt":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "transcript store is required") {
		t.Fatalf("Execute error = %v, want transcript store requirement", err)
	}
}

// TestTaskToolRunsEphemerallyWithoutParentSession mirrors headless `reasonix run`:
// the store is wired but the context carries no parent session, so the sub-agent
// must run without persistence and return its plain answer (no transcript ref).
func TestTaskToolRunsEphemerallyWithoutParentSession(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "headless answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)

	out, err := task.Execute(context.Background(), []byte(`{"prompt":"x"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "headless answer") {
		t.Fatalf("got %q, want sub-agent final answer", out)
	}
	if strings.Contains(out, "Subagent reference") {
		t.Fatalf("ephemeral run should not emit a transcript reference: %q", out)
	}
}

func TestReadOnlyTaskToolRunsEphemerallyWithReadOnlyRegistry(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "read-only findings"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "read_file", readOnly: true})
	parentReg.Add(fakeTool{name: "write_file", readOnly: false})
	parentReg.Add(fakeTool{name: "todo_write", readOnly: true})
	parentReg.Add(fakeTool{name: "complete_step", readOnly: true})
	parentReg.Add(fakeTool{name: "connect_tool_source", readOnly: true})
	parentReg.Add(fakeTool{name: "read_only_skill", readOnly: true})
	parentReg.Add(fakeTool{name: "bash", readOnly: false})
	task := newTestTaskTool(t, sub, parentReg, "writer sys", "", "", nil)
	readonly := NewReadOnlyTaskTool(task)
	parentReg.Add(task)
	parentReg.Add(readonly)

	out, err := readonly.Execute(testTaskContext(), []byte(`{"prompt":"inspect callers"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "read-only findings") {
		t.Fatalf("output = %q, want final answer", out)
	}
	if strings.Contains(out, "Subagent reference") {
		t.Fatalf("read_only_task should not persist transcript refs: %q", out)
	}
	if sys := sub.lastReq.Messages[0]; sys.Role != provider.RoleSystem || sys.Content != DefaultReadOnlyTaskSystemPrompt {
		t.Fatalf("read_only_task system prompt = %+v, want read-only prompt", sys)
	}

	got := map[string]bool{}
	for _, s := range sub.lastReq.Tools {
		got[s.Name] = true
	}
	for _, want := range []string{"read_file", "bash"} {
		if !got[want] {
			t.Fatalf("read_only_task sub-agent missing %q; tools=%v", want, toolSchemaNames(sub.lastReq.Tools))
		}
	}
	for _, hidden := range []string{"write_file", "todo_write", "complete_step", "connect_tool_source", "task", "read_only_task", "read_only_skill"} {
		if got[hidden] {
			t.Fatalf("read_only_task sub-agent should hide %q; tools=%v", hidden, toolSchemaNames(sub.lastReq.Tools))
		}
	}
}

func TestTaskToolRejectsContinuationWithoutParentSession(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)

	_, err := task.Execute(context.Background(), []byte(`{"prompt":"x","continue_from":"sa_whatever"}`))
	if err == nil || !strings.Contains(err.Error(), "persisted session") {
		t.Fatalf("Execute error = %v, want persisted-session requirement", err)
	}
}

func TestTaskToolPersistsAndContinuesTranscript(t *testing.T) {
	sub := &mockProvider{name: "sub", streams: [][]provider.Chunk{
		{
			{Type: provider.ChunkText, Text: "first answer"},
			{Type: provider.ChunkDone},
		},
		{
			{Type: provider.ChunkText, Text: "second answer"},
			{Type: provider.ChunkDone},
		},
	}}
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	store := NewSubagentStore(t.TempDir())
	task := newTestTaskTool(t, sub, reg, "sys", "", "", nil).
		WithTranscripts(store, t.TempDir(), "base-model", "base-effort")

	first, err := task.Execute(testTaskContext(), []byte(`{"prompt":"first task"}`))
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	ref := subagentRefFromOutput(t, first)
	meta, err := store.LoadMeta(ref)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if meta.ParentSession != "parent-session" {
		t.Fatalf("parent session = %q, want parent-session", meta.ParentSession)
	}
	if !strings.Contains(first, "first answer") {
		t.Fatalf("first output = %q, want answer", first)
	}

	second, err := task.Execute(testTaskContext(), []byte(`{"prompt":"second task","continue_from":"`+ref+`"}`))
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if !strings.Contains(second, "second answer") {
		t.Fatalf("second output = %q, want answer", second)
	}
	if len(sub.requests) != 2 {
		t.Fatalf("provider requests = %d, want 2", len(sub.requests))
	}
	msgs := sub.requests[1].Messages
	if len(msgs) < 4 {
		t.Fatalf("continued request messages = %+v, want prior transcript plus new task", msgs)
	}
	if msgs[1].Content != "first task" || msgs[2].Content != "first answer" || lastUser(sub.requests[1]) != "second task" {
		t.Fatalf("continued request messages = %+v, want first task/answer then second task", msgs)
	}
}

func TestTaskToolContinueFromAncestorReturnsCopiedReferenceGuidance(t *testing.T) {
	sub := &mockProvider{name: "sub", streams: [][]provider.Chunk{
		{
			{Type: provider.ChunkText, Text: "root answer"},
			{Type: provider.ChunkDone},
		},
		{
			{Type: provider.ChunkText, Text: "child answer"},
			{Type: provider.ChunkDone},
		},
	}}
	sessionDir := t.TempDir()
	store := NewSubagentStore(filepath.Join(sessionDir, "subagents"))
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(store, t.TempDir(), "base-model", "base-effort")

	rootCtx := WithParentSession(context.Background(), "root")
	first, err := task.Execute(rootCtx, []byte(`{"prompt":"root task"}`))
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	rootRef := subagentRefFromOutput(t, first)

	if err := SaveBranchMeta(filepath.Join(sessionDir, "root.jsonl"), BranchMeta{}); err != nil {
		t.Fatalf("SaveBranchMeta root: %v", err)
	}
	if err := SaveBranchMeta(filepath.Join(sessionDir, "child.jsonl"), BranchMeta{ParentID: "root"}); err != nil {
		t.Fatalf("SaveBranchMeta child: %v", err)
	}

	childCtx := WithParentSession(context.Background(), "child")
	second, err := task.Execute(childCtx, []byte(`{"prompt":"child task","continue_from":"`+rootRef+`"}`))
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	childRef := subagentRefFromOutput(t, second)
	if childRef == rootRef {
		t.Fatalf("child ref = source ref %q, want copied ref", childRef)
	}
	if !strings.Contains(second, "Forked from: "+rootRef) {
		t.Fatalf("second output = %q, want Forked from source ref", second)
	}
	if !strings.Contains(second, "The requested ref resolves to an ancestor conversation transcript") {
		t.Fatalf("second output = %q, want ancestor-copy guidance", second)
	}
	if !strings.Contains(second, "Final answer:\nchild answer") {
		t.Fatalf("second output = %q, want final answer", second)
	}
}

func TestTaskToolLegacyForkFromAncestorConvertsToCopiedReference(t *testing.T) {
	sub := &mockProvider{name: "sub", streams: [][]provider.Chunk{
		{
			{Type: provider.ChunkText, Text: "root answer"},
			{Type: provider.ChunkDone},
		},
		{
			{Type: provider.ChunkText, Text: "child answer"},
			{Type: provider.ChunkDone},
		},
	}}
	sessionDir := t.TempDir()
	store := NewSubagentStore(filepath.Join(sessionDir, "subagents"))
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(store, t.TempDir(), "base-model", "base-effort")

	rootCtx := WithParentSession(context.Background(), "root")
	first, err := task.Execute(rootCtx, []byte(`{"prompt":"root task"}`))
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	rootRef := subagentRefFromOutput(t, first)

	if err := SaveBranchMeta(filepath.Join(sessionDir, "root.jsonl"), BranchMeta{}); err != nil {
		t.Fatalf("SaveBranchMeta root: %v", err)
	}
	if err := SaveBranchMeta(filepath.Join(sessionDir, "child.jsonl"), BranchMeta{ParentID: "root"}); err != nil {
		t.Fatalf("SaveBranchMeta child: %v", err)
	}

	childCtx := WithParentSession(context.Background(), "child")
	second, err := task.Execute(childCtx, []byte(`{"prompt":"child task","fork_from":"`+rootRef+`"}`))
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	childRef := subagentRefFromOutput(t, second)
	if childRef == rootRef {
		t.Fatalf("child ref = source ref %q, want copied ref", childRef)
	}
	if !strings.Contains(second, "Forked from: "+rootRef) ||
		!strings.Contains(second, "Final answer:\nchild answer") {
		t.Fatalf("second output = %q, want copied reference guidance and final answer", second)
	}
}

func TestTaskToolRejectsLegacyForkFromCurrentSession(t *testing.T) {
	sub := &mockProvider{name: "sub", streams: [][]provider.Chunk{
		{
			{Type: provider.ChunkText, Text: "first answer"},
			{Type: provider.ChunkDone},
		},
		{
			{Type: provider.ChunkText, Text: "should not run"},
			{Type: provider.ChunkDone},
		},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")

	first, err := task.Execute(testTaskContext(), []byte(`{"prompt":"first task"}`))
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	ref := subagentRefFromOutput(t, first)
	_, err = task.Execute(testTaskContext(), []byte(`{"prompt":"second task","fork_from":"`+ref+`"}`))
	if err == nil || !strings.Contains(err.Error(), "cannot be safely converted") {
		t.Fatalf("legacy fork error = %v, want unsafe conversion rejection", err)
	}
	if len(sub.requests) != 1 {
		t.Fatalf("provider requests = %d, want only first run", len(sub.requests))
	}
}

func TestTaskToolFailedForegroundContinuationPersistsAndRejectsReuse(t *testing.T) {
	sub := &mockProvider{name: "sub", streams: [][]provider.Chunk{
		{
			{Type: provider.ChunkText, Text: "first answer"},
			{Type: provider.ChunkDone},
		},
		{
			{Type: provider.ChunkError, Err: errors.New("provider failed")},
		},
	}}
	store := NewSubagentStore(t.TempDir())
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(store, t.TempDir(), "base-model", "base-effort")

	first, err := task.Execute(testTaskContext(), []byte(`{"prompt":"first task"}`))
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	ref := subagentRefFromOutput(t, first)

	_, err = task.Execute(testTaskContext(), []byte(`{"prompt":"second task","continue_from":"`+ref+`"}`))
	if err == nil || !strings.Contains(err.Error(), "provider failed") {
		t.Fatalf("second Execute error = %v, want provider failure", err)
	}
	meta, err := store.LoadMeta(ref)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if meta.Status != SubagentFailed {
		t.Fatalf("status = %q, want failed", meta.Status)
	}
	loaded, err := LoadSession(store.sessionPath(ref))
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	msgs := loaded.Snapshot()
	if len(msgs) != 4 || msgs[1].Content != "first task" || msgs[2].Content != "first answer" || msgs[3].Content != "second task" {
		t.Fatalf("failed continuation transcript = %+v, want first task/answer plus second task", msgs)
	}
	if _, err := task.Execute(testTaskContext(), []byte(`{"prompt":"third task","continue_from":"`+ref+`"}`)); err == nil || !strings.Contains(err.Error(), "failed and cannot be continued") {
		t.Fatalf("reuse error = %v, want failed ref rejection", err)
	}
}

func TestTaskToolBackgroundPanicPersistsFailedMetadata(t *testing.T) {
	sub := panicProvider{name: "panic-sub"}
	store := NewSubagentStore(t.TempDir())
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(store, t.TempDir(), "base-model", "base-effort")

	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ctx := testTaskContext()
	ctx = jobs.WithSession(ctx, "parent-session")
	ctx = jobs.WithManager(ctx, jm)
	out, err := task.Execute(ctx, []byte(`{"prompt":"panic task","run_in_background":true}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	ref := subagentRefFromOutput(t, out)
	jobID := extractJobID(out)
	if jobID == "" {
		t.Fatalf("no background job id in output:\n%s", out)
	}
	res := jm.WaitForSession(context.Background(), "parent-session", []string{jobID}, 5)
	if len(res) != 1 || res[0].Status != jobs.Failed {
		t.Fatalf("background job result = %+v, want failed", res)
	}
	if !strings.Contains(res[0].Output, "Subagent reference (failed): "+ref) {
		t.Fatalf("job output = %q, want failed subagent ref %s", res[0].Output, ref)
	}
	meta, err := store.LoadMeta(ref)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if meta.Status != SubagentFailed {
		t.Fatalf("status = %q, want failed", meta.Status)
	}
	if _, err := task.Execute(testTaskContext(), []byte(`{"prompt":"again","continue_from":"`+ref+`"}`)); err == nil || !strings.Contains(err.Error(), "failed and cannot be continued") {
		t.Fatalf("reuse error = %v, want failed continuation rejection", err)
	}
}

func TestTaskToolBackgroundResultIncludesReferenceGuidance(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "background answer"},
		{Type: provider.ChunkDone},
	}}
	store := NewSubagentStore(t.TempDir())
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(store, t.TempDir(), "base-model", "base-effort")

	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ctx := testTaskContext()
	ctx = jobs.WithSession(ctx, "parent-session")
	ctx = jobs.WithManager(ctx, jm)
	out, err := task.Execute(ctx, []byte(`{"prompt":"background task","run_in_background":true}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	ref := subagentRefFromOutput(t, out)
	if !strings.Contains(out, "To continue this same subagent transcript in a later call") {
		t.Fatalf("start output = %q, want reference guidance", out)
	}
	jobID := extractJobID(out)
	if jobID == "" {
		t.Fatalf("no background job id in output:\n%s", out)
	}
	res := jm.WaitForSession(context.Background(), "parent-session", []string{jobID}, 5)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("background job result = %+v, want succeeded", res)
	}
	if !strings.Contains(res[0].Output, "Subagent reference: "+ref) ||
		!strings.Contains(res[0].Output, "To continue this same subagent transcript in a later call") ||
		!strings.Contains(res[0].Output, "Final answer:\nbackground answer") {
		t.Fatalf("job output = %q, want reference guidance and final answer", res[0].Output)
	}
}

func TestTaskToolBackgroundAncestorContinuationIncludesForkGuidance(t *testing.T) {
	sub := &mockProvider{name: "sub", streams: [][]provider.Chunk{
		{
			{Type: provider.ChunkText, Text: "root answer"},
			{Type: provider.ChunkDone},
		},
		{
			{Type: provider.ChunkText, Text: "child background answer"},
			{Type: provider.ChunkDone},
		},
	}}
	sessionDir := t.TempDir()
	store := NewSubagentStore(filepath.Join(sessionDir, "subagents"))
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(store, t.TempDir(), "base-model", "base-effort")

	rootCtx := WithParentSession(context.Background(), "root")
	rootOut, err := task.Execute(rootCtx, []byte(`{"prompt":"root task"}`))
	if err != nil {
		t.Fatalf("root Execute: %v", err)
	}
	rootRef := subagentRefFromOutput(t, rootOut)
	if err := SaveBranchMeta(filepath.Join(sessionDir, "root.jsonl"), BranchMeta{}); err != nil {
		t.Fatalf("SaveBranchMeta root: %v", err)
	}
	if err := SaveBranchMeta(filepath.Join(sessionDir, "child.jsonl"), BranchMeta{ParentID: "root"}); err != nil {
		t.Fatalf("SaveBranchMeta child: %v", err)
	}

	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	childCtx := WithParentSession(context.Background(), "child")
	childCtx = jobs.WithSession(childCtx, "child")
	childCtx = jobs.WithManager(childCtx, jm)
	startOut, err := task.Execute(childCtx, []byte(`{"prompt":"child task","continue_from":"`+rootRef+`","run_in_background":true}`))
	if err != nil {
		t.Fatalf("child Execute: %v", err)
	}
	childRef := subagentRefFromOutput(t, startOut)
	if childRef == rootRef {
		t.Fatalf("child ref = source ref %q, want copied ref", childRef)
	}
	if !strings.Contains(startOut, "Forked from: "+rootRef) ||
		!strings.Contains(startOut, "The requested ref resolves to an ancestor conversation transcript") ||
		strings.Contains(startOut, "Final answer:") {
		t.Fatalf("start output = %q, want fork guidance without final answer", startOut)
	}
	jobID := extractJobID(startOut)
	if jobID == "" {
		t.Fatalf("no background job id in output:\n%s", startOut)
	}
	res := jm.WaitForSession(context.Background(), "child", []string{jobID}, 5)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("background job result = %+v, want succeeded", res)
	}
	if !strings.Contains(res[0].Output, "Subagent reference: "+childRef) ||
		!strings.Contains(res[0].Output, "Forked from: "+rootRef) ||
		!strings.Contains(res[0].Output, "The requested ref resolves to an ancestor conversation transcript") ||
		!strings.Contains(res[0].Output, "Final answer:\nchild background answer") {
		t.Fatalf("job output = %q, want copied ref guidance and final answer", res[0].Output)
	}
}

func TestTaskToolRejectsMismatchedContinuationProfile(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "")

	out, err := task.Execute(testTaskContext(), []byte(`{"prompt":"first task"}`))
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	ref := subagentRefFromOutput(t, out)
	_, err = task.Execute(testTaskContext(), []byte(`{"prompt":"second task","continue_from":"`+ref+`","model":"other-model"}`))
	if err == nil || !strings.Contains(err.Error(), "model/effort") {
		t.Fatalf("mismatched model error = %v, want compatibility failure", err)
	}
}

func subagentRefFromOutput(t *testing.T, out string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Subagent reference: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Subagent reference: "))
		}
	}
	t.Fatalf("no subagent reference in output:\n%s", out)
	return ""
}

func TestSubSinkForwardsUsageToParent(t *testing.T) {
	var got []event.Event
	parent := event.FuncSink(func(e event.Event) {
		got = append(got, e)
	})
	subSinkFor("task_1", parent).Emit(event.Event{
		Kind:        event.Usage,
		Usage:       &provider.Usage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12},
		UsageSource: event.UsageSourceSubagent,
	})
	if len(got) != 1 || got[0].Usage == nil || got[0].UsageSource != event.UsageSourceSubagent {
		t.Fatalf("forwarded events = %+v, want subagent usage", got)
	}
}

func TestTaskToolCarriesRecentKeepIntoSubsessions(t *testing.T) {
	task := NewTaskTool(&mockProvider{name: "sub"}, nil, tool.NewRegistry(), 20, 0, 7, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil)
	if task.recentKeep != 7 {
		t.Fatalf("recentKeep = %d, want 7", task.recentKeep)
	}
}

func newTestTaskTool(t *testing.T, prov provider.Provider, reg *tool.Registry, sysPrompt, subagentModel, subagentEffort string, resolve func(string, string) (provider.Provider, *provider.Pricing, int, error)) *TaskTool {
	t.Helper()
	return NewTaskTool(prov, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", sysPrompt, nil, 0, subagentModel, subagentEffort, resolve).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
}


// TestTaskToolBatchDispatch verifies that batch mode spawns multiple background
// tasks and returns a summary with job IDs.
func TestTaskToolBatchDispatch(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)

	jm := jobs.NewManager(nil)
	ctx := jobs.WithManager(testTaskContext(), jm)

	out, err := task.Execute(ctx, []byte(`{"batch":[{"prompt":"task A","description":"alpha"},{"prompt":"task B","description":"beta"}]}`))
	if err != nil {
		t.Fatalf("Execute batch: %v", err)
	}
	if !strings.Contains(out, "Started 2 parallel background tasks") {
		t.Errorf("expected batch summary, got: %s", out)
	}
	if !strings.Contains(out, `"alpha"`) || !strings.Contains(out, `"beta"`) {
		t.Errorf("expected both task labels in output, got: %s", out)
	}

	// Wait for background jobs to finish so the temp dir can be cleaned up.
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	jm.Close()
}

// TestTaskToolBatchRejectsPromptWithBatch verifies mutual exclusion.
func TestTaskToolBatchRejectsPromptWithBatch(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)

	jm := jobs.NewManager(nil)
	ctx := jobs.WithManager(testTaskContext(), jm)

	_, err := task.Execute(ctx, []byte(`{"prompt":"x","batch":[{"prompt":"y"}]}`))
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutual exclusion error, got: %v", err)
	}
}

// TestTaskToolBatchRejectsEmptyPrompt verifies batch items require prompt.
func TestTaskToolBatchRejectsEmptyPrompt(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)

	jm := jobs.NewManager(nil)
	ctx := jobs.WithManager(testTaskContext(), jm)

	_, err := task.Execute(ctx, []byte(`{"batch":[{"prompt":" "}]}`))
	if err == nil || !strings.Contains(err.Error(), "prompt is required") {
		t.Fatalf("expected prompt-required error, got: %v", err)
	}
}

// TestTaskToolBatchRequiresJobsContext verifies batch fails without jobs manager.
func TestTaskToolBatchRequiresJobsContext(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)

	_, err := task.Execute(testTaskContext(), []byte(`{"batch":[{"prompt":"x"}]}`))
	if err == nil || !strings.Contains(err.Error(), "background execution is not available") {
		t.Fatalf("expected jobs-context error, got: %v", err)
	}
}

// TestTaskToolBatchHonorsTopLevelMaxSteps verifies the top-level max_steps is
// accepted alongside batch and passed through to executeBatch as the default
// for items that don't specify their own max_steps.
func TestTaskToolBatchHonorsTopLevelMaxSteps(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)
	jm := jobs.NewManager(nil)
	ctx := jobs.WithManager(testTaskContext(), jm)

	// max_steps=42 at top level should be accepted alongside batch.
	out, err := task.Execute(ctx, []byte(`{"batch":[{"prompt":"task A","description":"alpha"},{"prompt":"task B","description":"beta"}],"max_steps":42}`))
	if err != nil {
		t.Fatalf("Execute batch with top-level max_steps: %v", err)
	}
	if !strings.Contains(out, "Started 2 parallel background tasks") {
		t.Errorf("expected batch summary, got: %s", out)
	}

	// Cleanup.
	jm.Close()
}

// TestTaskToolBatchItemMaxStepsOverridesTopLevel verifies per-item max_steps
// overrides the top-level max_steps.
func TestTaskToolBatchItemMaxStepsOverridesTopLevel(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)
	jm := jobs.NewManager(nil)
	ctx := jobs.WithManager(testTaskContext(), jm)

	// Top-level max_steps=10, but item specifies max_steps=50 — item should win.
	out, err := task.Execute(ctx, []byte(`{"batch":[{"prompt":"task A","description":"alpha","max_steps":50}],"max_steps":10}`))
	if err != nil {
		t.Fatalf("Execute batch with item-level max_steps override: %v", err)
	}
	if !strings.Contains(out, "Started 1 parallel background task") {
		t.Errorf("expected batch summary, got: %s", out)
	}

	jm.Close()
}

// TestTaskToolBatchE2EWriteFile verifies end-to-end: batch sub-agents with a
// multi-turn provider can call write_file and produce persistent output.
func TestTaskToolBatchE2EWriteFile(t *testing.T) {
	ws := t.TempDir()
	resultPath := ws + "/results/item.json"

	// Build a tool registry with workspace-bound write_file.
	reg := tool.NewRegistry()
	wsTools := toolbuiltin.Workspace{Dir: ws}.Tools()
	for _, tl := range wsTools {
		reg.Add(tl)
	}

	// Multi-turn mock: turn 1 = write_file, turn 2 = final answer.
	sub := &mockProvider{
		name: "sub",
		streams: [][]provider.Chunk{
			// Turn 1: call write_file with the JSON result.
			{
				{Type: provider.ChunkToolCallStart, ToolCall: &provider.ToolCall{ID: "c1", Name: "write_file"}},
				{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{
					ID: "c1", Name: "write_file",
					Arguments: `{"path":"results/item.json","content":"{\"name\":\"item1\",\"status\":\"done\"}"}`,
				}},
				{Type: provider.ChunkDone},
			},
			// Turn 2: final text answer.
			{
				{Type: provider.ChunkText, Text: "JSON written to results/item.json"},
				{Type: provider.ChunkDone},
			},
		},
	}
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")

	jm := jobs.NewManager(nil)
	ctx := jobs.WithManager(testTaskContext(), jm)

	out, err := task.Execute(ctx, []byte(`{"batch":[{"prompt":"research item1"}],"max_steps":30}`))
	if err != nil {
		t.Fatalf("Execute batch: %v", err)
	}
	if !strings.Contains(out, "Started 1 parallel background task") {
		t.Errorf("expected batch summary, got: %s", out)
	}

	// Wait for the background job to finish.
	res := jm.Wait(context.Background(), nil, 5)
	if len(res) != 1 {
		t.Fatalf("expected 1 job result, got %d", len(res))
	}
	if res[0].Status != jobs.Done {
		t.Fatalf("job status = %v, want Done. Output: %s", res[0].Status, res[0].Output)
	}

	// Verify the file was actually written.
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", resultPath, err)
	}
	if string(data) != `{"name":"item1","status":"done"}` {
		t.Errorf("file content = %q, want JSON payload", string(data))
	}

	jm.Close()
}

// TestTaskToolBatchE2EHeadlessWriteFile verifies batch sub-agents in headless
// mode (no parent session, like `reasonix run`) can call write_file and produce output.
func TestTaskToolBatchE2EHeadlessWriteFile(t *testing.T) {
	ws := t.TempDir()
	resultPath := ws + "/results/item.json"

	reg := tool.NewRegistry()
	wsTools := toolbuiltin.Workspace{Dir: ws}.Tools()
	for _, tl := range wsTools {
		reg.Add(tl)
	}

	// Multi-turn mock: turn 1 = write_file, turn 2 = final answer.
	sub := &mockProvider{
		name: "sub",
		streams: [][]provider.Chunk{
			{
				{Type: provider.ChunkToolCallStart, ToolCall: &provider.ToolCall{ID: "c1", Name: "write_file"}},
				{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{
					ID: "c1", Name: "write_file",
					Arguments: `{"path":"results/item.json","content":"{\"name\":\"item1\"}"}`,
				}},
				{Type: provider.ChunkDone},
			},
			{
				{Type: provider.ChunkText, Text: "done"},
				{Type: provider.ChunkDone},
			},
		},
	}
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), ws, "base-model", "base-effort")

	jm := jobs.NewManager(nil)
	// Headless: no parent session set on context.
	ctx := jobs.WithManager(context.Background(), jm)

	out, err := task.Execute(ctx, []byte(`{"batch":[{"prompt":"research item1"}],"max_steps":30}`))
	if err != nil {
		t.Fatalf("Execute batch headless: %v", err)
	}
	if !strings.Contains(out, "Started 1 parallel background task") {
		t.Errorf("expected batch summary, got: %s", out)
	}

	res := jm.Wait(context.Background(), nil, 5)
	if len(res) != 1 {
		t.Fatalf("expected 1 job result, got %d", len(res))
	}
	if res[0].Status != jobs.Done {
		t.Fatalf("job status = %v, want Done. Output: %s", res[0].Status, res[0].Output)
	}

	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", resultPath, err)
	}
	if string(data) != `{"name":"item1"}` {
		t.Errorf("file content = %q, want {\"name\":\"item1\"}", string(data))
	}

	jm.Close()
}

// TestTaskToolBatchWaitFindsSessionScopedJobs verifies that batch jobs started
// with session scoping are visible to wait when filtering by the same session.
// This is the regression test for the bug where jm.Start("task",...) (empty
// session) meant wait(jobs.SessionFromContext(ctx)) couldn't find them.
func TestTaskToolBatchWaitFindsSessionScopedJobs(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "answer"},
		{Type: provider.ChunkDone},
	}}
	task := newTestTaskTool(t, sub, tool.NewRegistry(), "sys", "", "", nil)
	jm := jobs.NewManager(nil)
	ctx := jobs.WithManager(testTaskContext(), jm)
	// Mimic the controller path: set a session on the context.
	ctx = jobs.WithSession(ctx, "my-session")

	out, err := task.Execute(ctx, []byte(`{"batch":[{"prompt":"task A","description":"alpha"}]}`))
	if err != nil {
		t.Fatalf("Execute batch with session: %v", err)
	}
	if !strings.Contains(out, "Started 1 parallel background task") {
		t.Errorf("expected batch summary, got: %s", out)
	}

	// The batch jobs should be scoped to "my-session" and found by wait.
	res := jm.WaitForSession(context.Background(), "my-session", nil, 5)
	if len(res) != 1 {
		t.Fatalf("expected 1 job for session my-session, got %d", len(res))
	}
	if res[0].Status != jobs.Done {
		t.Fatalf("job status = %v, want Done", res[0].Status)
	}

	// But a different session should see none.
	resOther := jm.WaitForSession(context.Background(), "other-session", nil, 1)
	if len(resOther) != 0 {
		t.Fatalf("expected 0 jobs for other-session, got %d", len(resOther))
	}

	jm.Close()
}

type panicProvider struct{ name string }

func (p panicProvider) Name() string { return p.name }

func (p panicProvider) Stream(context.Context, provider.Request) (<-chan provider.Chunk, error) {
	panic("subagent boom")
}
