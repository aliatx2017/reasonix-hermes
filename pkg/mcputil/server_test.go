package mcputil

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestHandleMessage_Initialize(t *testing.T) {
	srv := &Server{
		Name:    "test-server",
		Version: "1.0.0",
		Tools:   []Tool{{Name: "test_tool", Description: "a test"}},
		Handle:  func(name string, args map[string]any) (string, error) { return "ok", nil },
	}

	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	resp := srv.HandleMessage([]byte(req))
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	var r Response
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}

	var result map[string]any
	json.Unmarshal(r.Result, &result)
	info, _ := result["serverInfo"].(map[string]any)
	if info["name"] != "test-server" {
		t.Errorf("expected server name 'test-server', got %v", info["name"])
	}
}

func TestHandleMessage_StringID(t *testing.T) {
	srv := &Server{
		Name:    "test",
		Version: "1.0.0",
		Tools:   []Tool{},
		Handle:  func(name string, args map[string]any) (string, error) { return "ok", nil },
	}

	// Claude Code sends string IDs
	req := `{"jsonrpc":"2.0","id":"msg_01XYZ","method":"tools/list","params":{}}`
	resp := srv.HandleMessage([]byte(req))
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	var r Response
	json.Unmarshal(resp, &r)

	// ID must be echoed back verbatim as string
	if string(r.ID) != `"msg_01XYZ"` {
		t.Errorf("expected ID \"msg_01XYZ\", got %s", string(r.ID))
	}
}

func TestHandleMessage_NumericID(t *testing.T) {
	srv := &Server{
		Name:    "test",
		Version: "1.0.0",
		Tools:   []Tool{},
		Handle:  func(name string, args map[string]any) (string, error) { return "ok", nil },
	}

	req := `{"jsonrpc":"2.0","id":42,"method":"tools/list","params":{}}`
	resp := srv.HandleMessage([]byte(req))

	var r Response
	json.Unmarshal(resp, &r)

	if string(r.ID) != "42" {
		t.Errorf("expected ID 42, got %s", string(r.ID))
	}
}

func TestHandleMessage_NullID(t *testing.T) {
	srv := &Server{
		Name:    "test",
		Version: "1.0.0",
		Tools:   []Tool{},
		Handle:  func(name string, args map[string]any) (string, error) { return "ok", nil },
	}

	req := `{"jsonrpc":"2.0","id":null,"method":"tools/list","params":{}}`
	resp := srv.HandleMessage([]byte(req))

	var r Response
	json.Unmarshal(resp, &r)

	if string(r.ID) != "null" {
		t.Errorf("expected ID null, got %s", string(r.ID))
	}
}

func TestHandleMessage_Notification(t *testing.T) {
	srv := &Server{
		Name:    "test",
		Version: "1.0.0",
		Tools:   []Tool{},
		Handle:  func(name string, args map[string]any) (string, error) { return "ok", nil },
	}

	req := `{"jsonrpc":"2.0","id":1,"method":"notifications/initialized","params":{}}`
	resp := srv.HandleMessage([]byte(req))
	if resp != nil {
		t.Errorf("expected nil for notification, got %s", string(resp))
	}
}

func TestHandleMessage_ToolCall(t *testing.T) {
	srv := &Server{
		Name:    "test",
		Version: "1.0.0",
		Tools:   []Tool{{Name: "greet", Description: "say hello"}},
		Handle: func(name string, args map[string]any) (string, error) {
			who, _ := args["name"].(string)
			return "hello " + who, nil
		},
	}

	req := `{"jsonrpc":"2.0","id":"abc","method":"tools/call","params":{"name":"greet","arguments":{"name":"world"}}}`
	resp := srv.HandleMessage([]byte(req))

	var r Response
	json.Unmarshal(resp, &r)

	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}

	var result map[string]any
	json.Unmarshal(r.Result, &result)
	content := result["content"].([]any)
	first := content[0].(map[string]any)
	if first["text"] != "hello world" {
		t.Errorf("expected 'hello world', got %v", first["text"])
	}
}

func TestHandleMessage_ToolCallError(t *testing.T) {
	srv := &Server{
		Name:    "test",
		Version: "1.0.0",
		Tools:   []Tool{{Name: "fail", Description: "always fails"}},
		Handle: func(name string, args map[string]any) (string, error) {
			return "", fmt.Errorf("intentional failure")
		},
	}

	req := `{"jsonrpc":"2.0","id":99,"method":"tools/call","params":{"name":"fail","arguments":{}}}`
	resp := srv.HandleMessage([]byte(req))

	var r Response
	json.Unmarshal(resp, &r)

	if r.Error == nil {
		t.Fatal("expected error response")
	}
	if r.Error.Code != -32000 {
		t.Errorf("expected code -32000, got %d", r.Error.Code)
	}
}

func TestHandleMessage_MethodNotFound(t *testing.T) {
	srv := &Server{
		Name:    "test",
		Version: "1.0.0",
		Tools:   []Tool{},
		Handle:  func(name string, args map[string]any) (string, error) { return "", nil },
	}

	req := `{"jsonrpc":"2.0","id":1,"method":"nonexistent","params":{}}`
	resp := srv.HandleMessage([]byte(req))

	var r Response
	json.Unmarshal(resp, &r)

	if r.Error == nil || r.Error.Code != -32601 {
		t.Errorf("expected -32601 error, got %+v", r.Error)
	}
}

func TestHandleMessage_ParseError(t *testing.T) {
	srv := &Server{
		Name:    "test",
		Version: "1.0.0",
		Tools:   []Tool{},
		Handle:  func(name string, args map[string]any) (string, error) { return "", nil },
	}

	resp := srv.HandleMessage([]byte("not json"))

	var r Response
	json.Unmarshal(resp, &r)

	if r.Error == nil || r.Error.Code != -32700 {
		t.Errorf("expected -32700 parse error, got %+v", r.Error)
	}
}
