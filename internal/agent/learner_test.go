package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"reasonix/internal/learn"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// fakeExecTool is a simple tool that always succeeds.
type fakeExecTool struct{ name string }

func (f fakeExecTool) Name() string            { return f.name }
func (f fakeExecTool) Description() string     { return "" }
func (f fakeExecTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (f fakeExecTool) ReadOnly() bool          { return false }
func (f fakeExecTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return f.name + " done", nil
}

// TestLearnerObserveIntegration verifies the full agent→learner chain:
// agent.Run() → tool execution → learner.Observe() → patterns detected.
func TestLearnerObserveIntegration(t *testing.T) {
	lr := learn.New(learn.Config{Enabled: true, MinConfidence: 2, MaxObservations: 20})

	reg := tool.NewRegistry()
	reg.Add(fakeExecTool{name: "edit_file"})
	reg.Add(fakeExecTool{name: "bash"})

	// Run turn 1: edit_file then bash (edit-then-test pattern)
	prov1 := &mockProvider{name: "mp", chunks: []provider.Chunk{
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c1", Name: "edit_file", Arguments: `{"path":"x.go"}`}},
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c2", Name: "bash", Arguments: `{"command":"go test"}`}},
		{Type: provider.ChunkDone},
	}}
	a := New(prov1, reg, NewSession("sys"), Options{Learner: lr, MaxSteps: 1}, nil)
	_ = a.Run(context.Background(), "edit and test x.go")

	// Run turn 2: same pattern
	prov2 := &mockProvider{name: "mp", chunks: []provider.Chunk{
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c3", Name: "edit_file", Arguments: `{"path":"y.go"}`}},
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c4", Name: "bash", Arguments: `{"command":"go test"}`}},
		{Type: provider.ChunkDone},
	}}
	a = New(prov2, reg, NewSession("sys"), Options{Learner: lr, MaxSteps: 1}, nil)
	_ = a.Run(context.Background(), "edit and test y.go")

	// Run turn 3: same pattern (hits min confidence of 2)
	prov3 := &mockProvider{name: "mp", chunks: []provider.Chunk{
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c5", Name: "edit_file", Arguments: `{"path":"z.go"}`}},
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c6", Name: "bash", Arguments: `{"command":"go test"}`}},
		{Type: provider.ChunkDone},
	}}
	a = New(prov3, reg, NewSession("sys"), Options{Learner: lr, MaxSteps: 1}, nil)
	_ = a.Run(context.Background(), "edit and test z.go")

	// Verify observations were collected
	obs := lr.Observations()
	if len(obs) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(obs))
	}
	for i, o := range obs {
		if len(o.ToolCalls) != 2 {
			t.Errorf("turn %d: expected 2 tool calls, got %d", i+1, len(o.ToolCalls))
		}
		if o.ToolCalls[0].Name != "edit_file" {
			t.Errorf("turn %d, call 1: expected edit_file, got %s", i+1, o.ToolCalls[0].Name)
		}
		if o.ToolCalls[1].Name != "bash" {
			t.Errorf("turn %d, call 2: expected bash, got %s", i+1, o.ToolCalls[1].Name)
		}
		// Verify Success is populated (the gap we fixed)
		if !o.ToolCalls[0].Success {
			t.Errorf("turn %d, call 1: expected Success=true", i+1)
		}
		if !o.ToolCalls[1].Success {
			t.Errorf("turn %d, call 2: expected Success=true", i+1)
		}
	}

	// Verify patterns detected
	pats := lr.Patterns()
	if len(pats) == 0 {
		t.Fatal("expected at least one pattern (edit-then-test with 3 observations)")
	}
	foundAuto := false
	for _, p := range pats {
		if p.Confidence >= 2 {
			foundAuto = true
			if p.Trigger != "after editing files" {
				t.Errorf("expected trigger 'after editing files', got %q", p.Trigger)
			}
			break
		}
	}
	if !foundAuto {
		t.Error("no auto-verify pattern found")
	}
}

// TestLearnerObserveFailureTracking verifies that tool failures are tracked
// via the Success field.
func TestLearnerObserveFailureTracking(t *testing.T) {
	lr := learn.New(learn.Config{Enabled: true, MaxObservations: 10})

	reg := tool.NewRegistry()
	reg.Add(fakeExecTool{name: "bash"})
	// "broken" tool always returns an error
	reg.Add(fakeTool{name: "broken_tool", readOnly: false, err: errors.New("broken")})

	prov := &mockProvider{name: "mp", chunks: []provider.Chunk{
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c1", Name: "bash", Arguments: `{"command":"echo ok"}`}},
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c2", Name: "broken_tool", Arguments: `{}`}},
		{Type: provider.ChunkDone},
	}}
	a := New(prov, reg, NewSession("sys"), Options{Learner: lr, MaxSteps: 1}, nil)
	_ = a.Run(context.Background(), "test with broken tool")

	obs := lr.Observations()
	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(obs))
	}
	if len(obs[0].ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(obs[0].ToolCalls))
	}
	if !obs[0].ToolCalls[0].Success {
		t.Error("bash should succeed")
	}
	if obs[0].ToolCalls[1].Success {
		t.Error("broken_tool should fail")
	}
}

// TestLearnerObserveDisabled verifies learner is a no-op when disabled.
func TestLearnerObserveDisabled(t *testing.T) {
	lr := learn.New(learn.Config{Enabled: false})

	reg := tool.NewRegistry()
	reg.Add(fakeExecTool{name: "bash"})

	prov := &mockProvider{name: "mp", chunks: []provider.Chunk{
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c1", Name: "bash", Arguments: `{"command":"echo hi"}`}},
		{Type: provider.ChunkDone},
	}}
	a := New(prov, reg, NewSession("sys"), Options{Learner: lr, MaxSteps: 1}, nil)
	_ = a.Run(context.Background(), "hello")

	if len(lr.Observations()) != 0 {
		t.Error("disabled learner should collect no observations")
	}
}
