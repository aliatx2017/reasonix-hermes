package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"reasonix/internal/mesh"
	"reasonix/internal/tool"
)

func TestCouncilJudgeFallback(t *testing.T) {
	ctx := context.Background()
	tool, ok := tool.LookupBuiltin("council_judge")
	if !ok {
		t.Fatal("council_judge not registered as built-in")
	}

	_, err := tool.Execute(ctx, json.RawMessage(`{"task":"evaluate this"}`))
	if err == nil {
		t.Fatal("expected error from unconfigured council tool, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("error should mention not configured, got: %v", err)
	}
}

func TestCouncilJudgeMissingTask(t *testing.T) {
	ctx := context.Background()
	m := fakeMesh(t)
	cj := councilJudge{m: m}

	_, err := cj.Execute(ctx, json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "task is required") {
		t.Fatalf("expected 'task is required' error, got: %v", err)
	}

	_, err = cj.Execute(ctx, json.RawMessage(`{"task":""}`))
	if err == nil || !strings.Contains(err.Error(), "task is required") {
		t.Fatalf("expected 'task is required' error, got: %v", err)
	}
}

func TestCouncilJudgeBadArgs(t *testing.T) {
	ctx := context.Background()
	m := fakeMesh(t)
	cj := councilJudge{m: m}

	_, err := cj.Execute(ctx, json.RawMessage(`not json`))
	if err == nil || !strings.Contains(err.Error(), "invalid args") {
		t.Fatalf("expected 'invalid args' error, got: %v", err)
	}
}

func TestCouncilJudgeConveneErrorNoPeers(t *testing.T) {
	ctx := context.Background()
	// Create a mesh with no active peers — Convene should fail.
	// An empty mesh with no peers will return an error from Broadcast.
	m := mesh.New(mesh.Config{Enabled: true})
	if m == nil {
		t.Skip("New(nil peers) returns nil — test not applicable")
	}
	cj := councilJudge{m: m}

	_, err := cj.Execute(ctx, json.RawMessage(`{"task":"hello"}`))
	if err == nil {
		t.Fatal("expected error from convene with no peers, got nil")
	}
}

// fakeMesh creates a Mesh with a fake local HTTP server that echoes a response.
// The mesh stores peer URLs; we create one with a fake that responds "echo".
func fakeMesh(t *testing.T) *mesh.Mesh {
	t.Helper()
	// Use a Mesh that is "active" but has a fake localhost URL.
	// The Broadcast will try to reach the URL and fail, but the code path
	// through Convene → Broadcast is exercised.
	return mesh.New(mesh.Config{
		Enabled: true,
		Peers: []mesh.PeerConfig{
			{Name: "test-peer", URL: "http://127.0.0.1:19999/mcp", Enabled: true},
		},
	})
}

func TestCouncilJudgeSchema(t *testing.T) {
	cj := councilJudge{}
	schema := cj.Schema()
	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}
	props, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}
	if _, ok := props["task"]; !ok {
		t.Fatal("schema missing task property")
	}
	req, ok := parsed["required"].([]any)
	if !ok || len(req) != 1 || req[0] != "task" {
		t.Fatal("task must be required")
	}
}

func TestConfineCouncil(t *testing.T) {
	cj := ConfineCouncil(nil)
	if cj.Name() != "council_judge" {
		t.Fatalf("name = %q, want council_judge", cj.Name())
	}
	_, err := cj.Execute(context.Background(), json.RawMessage(`{"task":"test"}`))
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected not configured error with nil mesh, got: %v", err)
	}
}
