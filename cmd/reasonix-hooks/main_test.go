package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRetain_NoiseToolSkipped(t *testing.T) {
	for _, tool := range []string{"read_file", "write_file", "edit_file", "bash", "search", "glob", ""} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("noise tool %q should not trigger HTTP call", tool)
		}))
		payload := hookPayload{ToolName: tool, ToolInput: json.RawMessage(`{}`)}
		doRetain(ts.URL, "", 5*time.Second, payload)
		ts.Close()
	}
}

func TestRetain_MeaningfulToolSent(t *testing.T) {
	var gotMethod, gotTool string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		gotMethod = req["method"].(string)
		params := req["params"].(map[string]any)
		gotTool = params["name"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	payload := hookPayload{ToolName: "run_skill", ToolInput: json.RawMessage(`{"name":"test"}`)}
	doRetain(ts.URL, "", 5*time.Second, payload)

	if gotMethod != "tools/call" {
		t.Errorf("method = %q, want tools/call", gotMethod)
	}
	if gotTool != "hindsight_retain" {
		t.Errorf("tool = %q, want hindsight_retain", gotTool)
	}
}

func TestRetain_ContentIncludesToolName(t *testing.T) {
	var gotArgs map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		params := req["params"].(map[string]any)
		gotArgs = params["arguments"].(map[string]any)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	payload := hookPayload{ToolName: "run_skill"}
	doRetain(ts.URL, "", 5*time.Second, payload)

	content := gotArgs["content"].(string)
	if !strings.Contains(content, "run_skill") {
		t.Errorf("content = %q, should contain tool name", content)
	}
	tags := gotArgs["tags"].([]interface{})
	if len(tags) < 2 {
		t.Errorf("tags should contain at least 2 entries, got %d", len(tags))
	}
}

func TestReflect_SendsCorrectTool(t *testing.T) {
	var gotTool, gotSession string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		params := req["params"].(map[string]any)
		gotTool = params["name"].(string)
		args := params["arguments"].(map[string]any)
		gotSession = args["session_id"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	payload := hookPayload{SessionID: "test-session-123"}
	doReflect(ts.URL, "", 5*time.Second, payload)

	if gotTool != "hindsight_reflect" {
		t.Errorf("tool = %q, want hindsight_reflect", gotTool)
	}
	if gotSession != "test-session-123" {
		t.Errorf("session = %q, want test-session-123", gotSession)
	}
}

func TestReflect_EmptySessionDefaultsToLatest(t *testing.T) {
	var gotSession string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		args := req["params"].(map[string]any)["arguments"].(map[string]any)
		gotSession = args["session_id"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	payload := hookPayload{} // no session ID
	doReflect(ts.URL, "", 5*time.Second, payload)

	if gotSession != "latest" {
		t.Errorf("session = %q, want 'latest'", gotSession)
	}
}

func TestPostJSON_AuthHeader(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	err := postJSON(ts.URL, "secret-key", 5*time.Second, []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer secret-key" {
		t.Errorf("auth = %q, want 'Bearer secret-key'", gotAuth)
	}
}

func TestPostJSON_NoAuthWhenKeyEmpty(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	postJSON(ts.URL, "", 5*time.Second, []byte(`{}`))
	if gotAuth != "" {
		t.Errorf("no auth header expected when key is empty, got %q", gotAuth)
	}
}

func TestPostJSON_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	err := postJSON(ts.URL, "", 5*time.Second, []byte(`{}`))
	if err == nil {
		t.Error("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention 500, got: %v", err)
	}
}

func TestJSONRPCRequest_Structure(t *testing.T) {
	req := jsonrpcRequest("test_tool", map[string]any{"arg": "value"})
	if req["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v", req["jsonrpc"])
	}
	if req["method"] != "tools/call" {
		t.Errorf("method = %v", req["method"])
	}
	params := req["params"].(map[string]any)
	if params["name"] != "test_tool" {
		t.Errorf("name = %v", params["name"])
	}
	args := params["arguments"].(map[string]any)
	if args["arg"] != "value" {
		t.Errorf("arg = %v", args["arg"])
	}
}

func TestEnvDuration_Parse(t *testing.T) {
	t.Setenv("HINDSIGHT_TIMEOUT", "10")
	if d := envDuration("HINDSIGHT_TIMEOUT", 5*time.Second); d != 10*time.Second {
		t.Errorf("expected 10s, got %v", d)
	}
}

func TestEnvDuration_Default(t *testing.T) {
	if d := envDuration("NONEXISTENT", 5*time.Second); d != 5*time.Second {
		t.Errorf("expected 5s default, got %v", d)
	}
}

func TestEnvDuration_Invalid(t *testing.T) {
	t.Setenv("HINDSIGHT_TIMEOUT", "not-a-number")
	if d := envDuration("HINDSIGHT_TIMEOUT", 5*time.Second); d != 5*time.Second {
		t.Errorf("invalid should fall back to default 5s, got %v", d)
	}
}
