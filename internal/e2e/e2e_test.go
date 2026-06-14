package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"reasonix/internal/provider"
)

// saveFixture writes a minimal session file for testing.
func saveFixture(dir, name string, msgs []provider.Message) (string, error) {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, m := range msgs {
		if err := enc.Encode(m); err != nil {
			return "", err
		}
	}
	return path, nil
}

func TestSessionInputs(t *testing.T) {
	dir := t.TempDir()
	path, err := saveFixture(dir, "test.json", []provider.Message{
		{Role: provider.RoleSystem, Content: "You are a test agent."},
		{Role: provider.RoleUser, Content: "Hello"},
		{Role: provider.RoleAssistant, Content: "Hi there!"},
		{Role: provider.RoleUser, Content: "Do something"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{Name: "bash", Arguments: `{"command":"ls"}`}}},
		{Role: provider.RoleTool, Content: "file1.go\nfile2.go"},
	})
	if err != nil {
		t.Fatal(err)
	}

	inputs, err := SessionInputs(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 2 {
		t.Errorf("expected 2 inputs, got %d: %v", len(inputs), inputs)
	}
	if inputs[0] != "Hello" || inputs[1] != "Do something" {
		t.Errorf("unexpected inputs: %v", inputs)
	}
}

func TestSessionTools(t *testing.T) {
	dir := t.TempDir()
	path, err := saveFixture(dir, "test.json", []provider.Message{
		{Role: provider.RoleUser, Content: "test"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
			{Name: "bash", Arguments: `{"command":"ls"}`},
			{Name: "read_file", Arguments: `{"path":"x.go"}`},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	tools, err := SessionTools(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d: %v", len(tools), tools)
	}
}

func TestTurnCount(t *testing.T) {
	dir := t.TempDir()
	path, err := saveFixture(dir, "test.json", []provider.Message{
		{Role: provider.RoleUser, Content: "first"},
		{Role: provider.RoleAssistant, Content: "ok"},
		{Role: provider.RoleUser, Content: "second"},
		{Role: provider.RoleAssistant, Content: "ok"},
		{Role: provider.RoleUser, Content: "<compaction-summary>\nsummary\n</compaction-summary>"},
	})
	if err != nil {
		t.Fatal(err)
	}

	n, err := TurnCount(path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2 turns (compaction summary excluded), got %d", n)
	}
}

func TestAnalyze(t *testing.T) {
	dir := t.TempDir()
	path, err := saveFixture(dir, "test.json", []provider.Message{
		{Role: provider.RoleUser, Content: "read main.go"},
		{Role: provider.RoleAssistant, Content: "ok", ToolCalls: []provider.ToolCall{
			{Name: "read_file", Arguments: `{"path":"main.go"}`},
		}},
		{Role: provider.RoleTool, Content: "package main\n\nfunc main() {}"},
	})
	if err != nil {
		t.Fatal(err)
	}

	stats, err := Analyze(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Turns != 1 {
		t.Errorf("expected 1 turn, got %d", stats.Turns)
	}
	if stats.Tools != 1 {
		t.Errorf("expected 1 tool, got %d", stats.Tools)
	}
	if len(stats.ToolNames) != 1 || stats.ToolNames[0] != "read_file" {
		t.Errorf("unexpected tools: %v", stats.ToolNames)
	}
}

func TestAnalyzeFileExtraction(t *testing.T) {
	dir := t.TempDir()
	path, err := saveFixture(dir, "test.json", []provider.Message{
		{Role: provider.RoleUser, Content: "test"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
			{Name: "bash", Arguments: `{"command":"ls"}`},
		}},
		{Role: provider.RoleTool, Content: "internal/agent/agent.go\ncmd/reasonix/main.go"},
	})
	if err != nil {
		t.Fatal(err)
	}

	stats, err := Analyze(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Files < 2 {
		t.Errorf("expected at least 2 files extracted from tool output, got %d", stats.Files)
	}
}

func TestHarnessAssertTools(t *testing.T) {
	dir := t.TempDir()
	path, err := saveFixture(dir, "test.json", []provider.Message{
		{Role: provider.RoleUser, Content: "test"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
			{Name: "bash"},
			{Name: "read_file"},
			{Name: "write_file"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	h := NewHarness(t, Options{SessionsDir: dir})
	h.AssertTools(path, "bash", "read_file") // these exist — should pass

	// This would fail if uncommented (write_file exists, not missing)
	// h.AssertTools(path, "nonexistent_tool") // would error
}

func TestListSessions(t *testing.T) {
	dir := t.TempDir()
	_, _ = saveFixture(dir, "a.json", []provider.Message{{Role: provider.RoleUser, Content: "a"}})
	_, _ = saveFixture(dir, "b.json", []provider.Message{{Role: provider.RoleUser, Content: "b"}})
	// non-json file should be ignored
	_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)

	h := NewHarness(t, Options{SessionsDir: dir})
	paths, err := h.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 sessions, got %d: %v", len(paths), paths)
	}
}
