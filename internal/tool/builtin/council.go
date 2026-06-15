package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"reasonix/internal/mesh"
	"reasonix/internal/tool"
)

func init() { tool.RegisterBuiltin(councilJudge{}) }

// councilJudge broadcasts a task to mesh council peers and returns a
// synthesized consensus answer. The zero value (nil mesh) returns a
// descriptive error; boot.go replaces it with a confined instance via
// ConfineCouncil when mesh is configured.
type councilJudge struct {
	m *mesh.Mesh
}

func (councilJudge) Name() string { return "council_judge" }

func (councilJudge) Description() string {
	return "Ask the mesh council (multiple peer agents) to independently analyze a task and return a synthesized consensus. Useful when you need multiple perspectives on a problem, a second opinion, or a diverse set of approaches. The council broadcasts the task to all configured peers, collects their responses, and delivers a unified summary."
}

func (councilJudge) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"task":{"type":"string","description":"The task or question to send to all council peers for independent analysis."}},"required":["task"]}`)
}

func (councilJudge) ReadOnly() bool { return false }

func (c councilJudge) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("council_judge: invalid args: %w", err)
	}
	if p.Task == "" {
		return "", fmt.Errorf("council_judge: task is required")
	}

	if c.m == nil {
		return "", fmt.Errorf("council_judge: council is not configured — enable [mesh] in reasonix.toml and add at least one [[mesh.peers]] entry")
	}

	council := mesh.NewCouncil(c.m)
	if err := council.Convene(ctx, p.Task); err != nil {
		return "", fmt.Errorf("council_judge: convene failed: %w", err)
	}

	return council.Consensus(), nil
}
