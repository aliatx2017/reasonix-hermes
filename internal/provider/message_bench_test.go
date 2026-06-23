package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

// ── Benchmarks: Message JSON marshaling ──────────────────────────────────────

func BenchmarkMarshalUserMessage(b *testing.B) {
	msg := Message{
		Role:    RoleUser,
		Content: "Please analyze the following code and suggest improvements. The file is about 500 lines long and implements a REST API handler.",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkMarshalAssistantToolCall(b *testing.B) {
	msg := Message{
		Role: RoleAssistant,
		ToolCalls: []ToolCall{
			{ID: "call_abc123", Name: "read_file", Arguments: `{"path": "/src/main.go", "offset": 0, "limit": 100}`},
			{ID: "call_def456", Name: "grep", Arguments: `{"pattern": "TODO", "path": "/src/"}`},
			{ID: "call_ghi789", Name: "bash", Arguments: `{"command": "go build ./...", "workdir": "/src/"}`},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkMarshalToolResult(b *testing.B) {
	msg := Message{
		Role:       RoleTool,
		ToolCallID: "call_abc123",
		Name:       "read_file",
		Content:    strings.Repeat("line of output from the tool call\n", 50),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkUnmarshalUserMessage(b *testing.B) {
	data := []byte(`{"role":"user","content":"Please analyze the following code and suggest improvements."}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var msg Message
		_ = json.Unmarshal(data, &msg)
	}
}

func BenchmarkUnmarshalAssistantWithToolCalls(b *testing.B) {
	data := []byte(`{"role":"assistant","tool_calls":[{"id":"call_abc","name":"read_file","arguments":"{\"path\":\"/src/main.go\"}"},{"id":"call_def","name":"bash","arguments":"{\"command\":\"go build\"}"}]}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var msg Message
		_ = json.Unmarshal(data, &msg)
	}
}

// ── Benchmarks: ToolCall argument parsing ────────────────────────────────────

func BenchmarkParseImageDataURLValid(b *testing.B) {
	url := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseImageDataURL(url)
	}
}

func BenchmarkParseImageDataURLInvalid(b *testing.B) {
	url := "https://example.com/image.png"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseImageDataURL(url)
	}
}

// ── Benchmarks: Message list serialization (full request body) ───────────────

func BenchmarkMarshalRequestMessages(b *testing.B) {
	msgs := []Message{
		{Role: RoleSystem, Content: "You are a helpful coding assistant."},
	}
	// Add 20 turns of messages.
	for i := 0; i < 20; i++ {
		msgs = append(msgs, Message{Role: RoleUser, Content: "Turn " + strings.Repeat("x", 100)})
		msgs = append(msgs, Message{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{
				{ID: "call_1", Name: "read_file", Arguments: `{"path": "/src/main.go"}`},
			},
		})
		msgs = append(msgs, Message{
			Role:       RoleTool,
			ToolCallID: "call_1",
			Name:       "read_file",
			Content:    "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n",
		})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msgs)
	}
}

func BenchmarkUnmarshalRequestMessages(b *testing.B) {
	// Build a realistic message blob once.
	msgs := []Message{
		{Role: RoleSystem, Content: "You are a helpful coding assistant."},
	}
	for i := 0; i < 20; i++ {
		msgs = append(msgs, Message{Role: RoleUser, Content: "Turn " + strings.Repeat("x", 100)})
		msgs = append(msgs, Message{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{
				{ID: "call_1", Name: "read_file", Arguments: `{"path": "/src/main.go"}`},
			},
		})
		msgs = append(msgs, Message{
			Role:       RoleTool,
			ToolCallID: "call_1",
			Name:       "read_file",
			Content:    "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n",
		})
	}
	data, _ := json.Marshal(msgs)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result []Message
		_ = json.Unmarshal(data, &result)
	}
}
