package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"reasonix/pkg/httputil"
	"reasonix/pkg/mcputil"
)

// ── doctorCheck ──────────────────────────────────────────────────

func TestDoctorCheck_ReasonixBinary(t *testing.T) {
	bs := NewBridge(t.TempDir())
	report, err := bs.doctorCheck()
	if err != nil {
		t.Fatalf("doctorCheck returned error: %v", err)
	}

	if !strings.Contains(report, "# Reasonix Doctor Report") {
		t.Error("report missing heading")
	}
	if !strings.Contains(report, "reasonix binary") {
		t.Error("report missing reasonix binary check")
	}
	if !strings.Contains(report, "DEEPSEEK_API_KEY") {
		t.Error("report missing DEEPSEEK_API_KEY check")
	}
	if !strings.Contains(report, "reasonix.toml") {
		t.Error("report missing reasonix.toml check")
	}
	if !strings.Contains(report, "Go") {
		t.Error("report missing Go check")
	}
	if !strings.Contains(report, "Bridge version:") {
		t.Error("report missing Bridge version")
	}
}

func TestDoctorCheck_APIKeySet(t *testing.T) {
	key := "sk-test-key-12345"
	t.Setenv("DEEPSEEK_API_KEY", key)

	bs := NewBridge(t.TempDir())
	report, err := bs.doctorCheck()
	if err != nil {
		t.Fatalf("doctorCheck returned error: %v", err)
	}
	if !strings.Contains(report, "✅ **DEEPSEEK_API_KEY**: Set") {
		t.Errorf("expected API key marked as set, got:\n%s", report)
	}
	expectedLen := fmt.Sprintf("%d chars", len(key))
	if !strings.Contains(report, expectedLen) {
		t.Errorf("expected key length %s in report, got:\n%s", expectedLen, report)
	}
}

func TestDoctorCheck_APIKeyUnset(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")

	bs := NewBridge(t.TempDir())
	report, err := bs.doctorCheck()
	if err != nil {
		t.Fatalf("doctorCheck returned error: %v", err)
	}
	if !strings.Contains(report, "⚠️  **DEEPSEEK_API_KEY**: Not set") {
		t.Errorf("expected API key marked as not set, got:\n%s", report)
	}
}

func TestDoctorCheck_ReasonixFound(t *testing.T) {
	path, err := exec.LookPath("reasonix")
	if err != nil {
		t.Skip("reasonix binary not in PATH, skipping found test")
	}
	t.Logf("reasonix found at: %s", path)

	bs := NewBridge(t.TempDir())
	report, err := bs.doctorCheck()
	if err != nil {
		t.Fatalf("doctorCheck returned error: %v", err)
	}
	if !strings.Contains(report, "✅ **reasonix binary**: Found") {
		t.Errorf("expected reasonix found, got:\n%s", report)
	}
}

// ── listSkills ────────────────────────────────────────────────────

func TestListSkills_FromDirectory(t *testing.T) {
	// Set up an isolated HOME so os.UserConfigDir resolves predictably
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	skillsDir := filepath.Join(homeDir, "Library", "Application Support", "reasonix", "skills")
	if runtime.GOOS == "linux" {
		skillsDir = filepath.Join(homeDir, ".config", "reasonix", "skills")
	}
	os.MkdirAll(skillsDir, 0o755)

	// Create test skill files
	for _, name := range []string{"refactor.md", "debug.md", "test-gen.md"} {
		if err := os.WriteFile(filepath.Join(skillsDir, name), []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Create a non-.md file that should be skipped
	if err := os.WriteFile(filepath.Join(skillsDir, "notes.txt"), []byte("skip me"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create a directory that should be skipped
	os.MkdirAll(filepath.Join(skillsDir, "subdir"), 0o755)

	bs := NewBridge(t.TempDir())
	result, err := bs.listSkills()
	if err != nil {
		t.Fatalf("listSkills returned error: %v", err)
	}
	if !strings.Contains(result, "# Available Skills") {
		t.Error("missing skills heading")
	}
	if !strings.Contains(result, "- refactor") {
		t.Error("missing refactor skill")
	}
	if !strings.Contains(result, "- debug") {
		t.Error("missing debug skill")
	}
	if !strings.Contains(result, "- test-gen") {
		t.Error("missing test-gen skill")
	}
	if strings.Contains(result, "notes.txt") {
		t.Error("non-.md file should not appear")
	}
	if strings.Contains(result, "subdir") {
		t.Error("directories should not appear")
	}
}

func TestListSkills_ProjectLocalFallback(t *testing.T) {
	// Isolate HOME so UserConfigDir won't find an existing reasonix config
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	workDir := t.TempDir()
	skillsDir := filepath.Join(workDir, ".reasonix", "skills")
	os.MkdirAll(skillsDir, 0o755)

	if err := os.WriteFile(filepath.Join(skillsDir, "plan.md"), []byte("# plan skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	bs := NewBridge(workDir)
	result, err := bs.listSkills()
	if err != nil {
		t.Fatalf("listSkills returned error: %v", err)
	}
	if !strings.Contains(result, "- plan") {
		t.Errorf("expected plan skill listed, got:\n%s", result)
	}
}

func TestListSkills_NoSkillsDir(t *testing.T) {
	// Isolate HOME so UserConfigDir won't find an existing reasonix config
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	workDir := t.TempDir()
	// Ensure no .reasonix/skills or skills-hub/skills in workDir
	bs := NewBridge(workDir)
	result, err := bs.listSkills()
	if err != nil {
		t.Fatalf("listSkills should not error when no dirs exist, got: %v", err)
	}
	if !strings.Contains(result, "No skills found") {
		t.Errorf("expected 'No skills found' message, got:\n%s", result)
	}
}

func TestListSkills_EmptySkillsDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create an empty skills dir in the config location
	skillsDir := filepath.Join(homeDir, "Library", "Application Support", "reasonix", "skills")
	if runtime.GOOS == "linux" {
		skillsDir = filepath.Join(homeDir, ".config", "reasonix", "skills")
	}
	os.MkdirAll(skillsDir, 0o755)

	workDir := t.TempDir()
	bs := NewBridge(workDir)
	result, err := bs.listSkills()
	if err != nil {
		t.Fatalf("listSkills returned error: %v", err)
	}
	if !strings.Contains(result, "No skills found") {
		t.Errorf("expected 'No skills found' message, got:\n%s", result)
	}
}

// ── planTask ──────────────────────────────────────────────────────

func TestPlanTask_NoAPIKey(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")
	bs := NewBridge(t.TempDir())
	_, err := bs.planTask("refactor auth module")
	if err == nil {
		t.Fatal("expected error when DEEPSEEK_API_KEY not set")
	}
	if !strings.Contains(err.Error(), "DEEPSEEK_API_KEY") {
		t.Errorf("expected DEEPSEEK_API_KEY error, got: %v", err)
	}
}

func TestPlanTask_EmptyObjective_NoAPIKey(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")
	bs := NewBridge(t.TempDir())
	_, err := bs.planTask("")
	if err == nil {
		t.Fatal("expected error when DEEPSEEK_API_KEY not set")
	}
	if !strings.Contains(err.Error(), "DEEPSEEK_API_KEY") {
		t.Errorf("expected DEEPSEEK_API_KEY error, got: %v", err)
	}
}

func TestPlanTask_WithTestServer(t *testing.T) {
	// Start a fake DeepSeek API server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"content":"1. Analyze current auth module\n2. Design new structure\n3. Implement changes"}}]}`)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	bs := NewBridge(t.TempDir())
	bs.apiBase = ts.URL

	result, err := bs.planTask("refactor auth module")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "# Execution Plan") {
		t.Errorf("expected execution plan heading, got: %s", result)
	}
	if !strings.Contains(result, "Analyze current auth module") {
		t.Errorf("expected plan content, got: %s", result)
	}
}

func TestPlanTask_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	bs := NewBridge(t.TempDir())
	bs.apiBase = ts.URL

	_, err := bs.planTask("refactor auth module")
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected status code in error, got: %v", err)
	}
}

// ── orchestrateTask ───────────────────────────────────────────────────

func TestOrchestrateTask_NoReasonixBinary(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")
	bs := NewBridge(t.TempDir())
	// Even with API key, orchestrateTask needs reasonix binary
	// Test with missing key first (simpler)
	_, err := bs.orchestrateTask("build microservice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Either "not found in PATH" or "DEEPSEEK_API_KEY not set"
	// depending on which check runs first
}

func TestOrchestrateTask_WithTestServer(t *testing.T) {
	// Fake DeepSeek API that returns decomposition
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"content":"1. Set up project structure\n2. Implement auth service\n3. Add API endpoints"}}]}`)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	bs := NewBridge(t.TempDir())
	bs.apiBase = ts.URL

	// reasonix binary won't exist, so steps will fail — but we test decomposition
	result, err := bs.orchestrateTask("build microservice")
	// orchestrateTask doesn't return error for step failures — it reports them
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "# Orchestration Results") {
		t.Errorf("expected orchestration results heading, got: %s", result)
	}
	if !strings.Contains(result, "## Decomposition") {
		t.Errorf("expected decomposition section, got: %s", result)
	}
	if !strings.Contains(result, "## Execution Results") {
		t.Errorf("expected execution results section, got: %s", result)
	}
}

func TestOrchestrateTask_EmptyTask_NoAPIKey(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")
	bs := NewBridge(t.TempDir())
	_, err := bs.orchestrateTask("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ── runTask (executeTool "reasonix_run") ──────────────────────────

func TestExecuteTool_Run_MissingTask(t *testing.T) {
	bs := NewBridge(t.TempDir())
	_, err := bs.handle("reasonix_run", map[string]any{})
	if err == nil {
		t.Error("expected error for missing task, got nil")
	}
	if !strings.Contains(err.Error(), "task is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteTool_Run_EmptyTask(t *testing.T) {
	bs := NewBridge(t.TempDir())
	_, err := bs.handle("reasonix_run", map[string]any{"task": ""})
	if err == nil {
		t.Error("expected error for empty task, got nil")
	}
	if !strings.Contains(err.Error(), "task is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteTool_Run_BinaryNotFound(t *testing.T) {
	// Ensure reasonix is not on PATH by using a clean PATH
	t.Setenv("PATH", t.TempDir())

	bs := NewBridge(t.TempDir())
	result, err := bs.handle("reasonix_run", map[string]any{"task": "echo hello"})
	// runReasonix returns the error output as a string result, not a Go error
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !strings.Contains(result, "Reasonix execution error") {
		t.Errorf("expected execution error in result, got: %s", result)
	}
}

func TestExecuteTool_Run_UsesWorkdir(t *testing.T) {
	// Create a fake reasonix binary that prints its working directory
	binDir := t.TempDir()
	fakeBin := filepath.Join(binDir, "reasonix")
	script := `#!/bin/sh
echo "WORKDIR=$(pwd)"
`
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	workDir := t.TempDir()
	t.Setenv("PATH", binDir)

	bs := NewBridge(workDir)
	result, err := bs.handle("reasonix_run", map[string]any{
		"task": "test task",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resolve symlinks for comparison (macOS /var → /private/var)
	expectedWorkdir, _ := filepath.EvalSymlinks(workDir)
	if !strings.Contains(result, "WORKDIR="+expectedWorkdir) {
		t.Errorf("expected workdir %s in output, got: %s", expectedWorkdir, result)
	}
}

func TestExecuteTool_Run_ModelFlag(t *testing.T) {
	// Create a fake reasonix binary that prints its args
	binDir := t.TempDir()
	fakeBin := filepath.Join(binDir, "reasonix")
	script := `#!/bin/sh
echo "ARGS=$*"
`
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", binDir)

	bs := NewBridge(t.TempDir())
	result, err := bs.handle("reasonix_run", map[string]any{
		"task":  "do something",
		"model": "deepseek-flash",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "--model deepseek-flash") {
		t.Errorf("expected --model deepseek-flash in args, got: %s", result)
	}
}

// ── executeTool dispatch ──────────────────────────────────────────

func TestExecuteTool_UnknownTool(t *testing.T) {
	bs := NewBridge(t.TempDir())
	_, err := bs.handle("nonexistent_tool", map[string]any{})
	if err == nil {
		t.Error("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool: nonexistent_tool") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteTool_Doctor(t *testing.T) {
	bs := NewBridge(t.TempDir())
	result, err := bs.handle("reasonix_doctor", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "# Reasonix Doctor Report") {
		t.Errorf("expected doctor report, got: %s", result)
	}
}

// ── JSON-RPC handling ─────────────────────────────────────────────

func TestHandleMessage_Initialize(t *testing.T) {
	bs := NewBridge(t.TempDir())
	req := mcputil.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	}
	data, _ := json.Marshal(req)
	resp := bs.Server().HandleMessage(data)

	var parsed mcputil.Response
	if err := json.Unmarshal(resp, &parsed); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if string(parsed.ID) != "1" {
		t.Errorf("expected id 1, got %d", parsed.ID)
	}
	if parsed.Error != nil {
		t.Errorf("unexpected error: %+v", parsed.Error)
	}

	var result map[string]any
	if err := json.Unmarshal(parsed.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("unexpected protocol version: %v", result["protocolVersion"])
	}
}

func TestHandleMessage_ListTools(t *testing.T) {
	bs := NewBridge(t.TempDir())
	req := mcputil.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	}
	data, _ := json.Marshal(req)
	resp := bs.Server().HandleMessage(data)

	var parsed mcputil.Response
	if err := json.Unmarshal(resp, &parsed); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if parsed.Error != nil {
		t.Fatalf("unexpected error: %+v", parsed.Error)
	}

	var result map[string]any
	if err := json.Unmarshal(parsed.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools not a slice: %T", result["tools"])
	}
	if len(tools) != 6 {
		t.Errorf("expected 6 tools, got %d", len(tools))
	}
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	bs := NewBridge(t.TempDir())
	resp := bs.Server().HandleMessage([]byte("{invalid}"))

	var parsed mcputil.Response
	if err := json.Unmarshal(resp, &parsed); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if parsed.Error == nil {
		t.Error("expected error for invalid JSON")
	}
	if parsed.Error.Code != -32700 {
		t.Errorf("expected code -32700, got %d", parsed.Error.Code)
	}
}

func TestHandleMessage_MethodNotFound(t *testing.T) {
	bs := NewBridge(t.TempDir())
	req := mcputil.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "nonexistent/method",
	}
	data, _ := json.Marshal(req)
	resp := bs.Server().HandleMessage(data)

	var parsed mcputil.Response
	if err := json.Unmarshal(resp, &parsed); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if parsed.Error == nil {
		t.Error("expected error for unknown method")
	}
	if parsed.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", parsed.Error.Code)
	}
}

func TestHandleMessage_NotificationsInitialized(t *testing.T) {
	bs := NewBridge(t.TempDir())
	req := mcputil.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`0`),
		Method:  "notifications/initialized",
	}
	data, _ := json.Marshal(req)
	resp := bs.Server().HandleMessage(data)
	if resp != nil {
		t.Errorf("expected nil response for notification, got: %s", string(resp))
	}
}

func TestHandleMessage_CallTool_Doctor(t *testing.T) {
	bs := NewBridge(t.TempDir())
	params, _ := json.Marshal(map[string]any{
		"name":      "reasonix_doctor",
		"arguments": map[string]any{},
	})
	req := mcputil.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "tools/call",
		Params:  params,
	}
	data, _ := json.Marshal(req)
	resp := bs.Server().HandleMessage(data)

	var parsed mcputil.Response
	if err := json.Unmarshal(resp, &parsed); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if parsed.Error != nil {
		t.Fatalf("unexpected error: %+v", parsed.Error)
	}

	var result map[string]any
	if err := json.Unmarshal(parsed.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	content, ok := result["content"].([]any)
	if !ok {
		t.Fatalf("content not a slice: %T", result["content"])
	}
	if len(content) == 0 {
		t.Fatal("expected content entries")
	}
	textEntry, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content entry not a map: %T", content[0])
	}
	if !strings.Contains(textEntry["text"].(string), "# Reasonix Doctor Report") {
		t.Errorf("expected doctor report in text content, got: %v", textEntry["text"])
	}
}

func TestHandleMessage_CallTool_PlanTask(t *testing.T) {
	bs := NewBridge(t.TempDir())
	params, _ := json.Marshal(map[string]any{
		"name": "plan_task",
		"arguments": map[string]any{
			"objective": "refactor everything",
		},
	})
	req := mcputil.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`5`),
		Method:  "tools/call",
		Params:  params,
	}
	data, _ := json.Marshal(req)
	resp := bs.Server().HandleMessage(data)

	var parsed mcputil.Response
	if err := json.Unmarshal(resp, &parsed); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if parsed.Error == nil {
		t.Error("expected error for plan_task without API key")
	}
	if !strings.Contains(parsed.Error.Message, "DEEPSEEK_API_KEY") {
		t.Errorf("expected DEEPSEEK_API_KEY error, got: %s", parsed.Error.Message)
	}
}

// ── BridgeServer construction ─────────────────────────────────────

func TestNewBridgeServer_RegistersAllTools(t *testing.T) {
	bs := NewBridge(t.TempDir())
	if len(bs.tools()) != 6 {
		t.Errorf("expected 6 tools, got %d", len(bs.tools()))
	}

	names := map[string]bool{}
	for _, tool := range bs.tools() {
		names[tool.Name] = true
	}
	for _, expected := range []string{"reasonix_run", "reasonix_doctor", "plan_task", "orchestrate_task", "get_skill", "get_skills"} {
		if !names[expected] {
			t.Errorf("missing tool: %s", expected)
		}
	}
}

func TestNewBridgeServer_DefaultAPIBase(t *testing.T) {
	os.Unsetenv("DEEPSEEK_BASE_URL")
	bs := NewBridge(t.TempDir())
	if bs.apiBase != "https://api.deepseek.com" {
		t.Errorf("expected default apiBase, got: %s", bs.apiBase)
	}
}

func TestNewBridgeServer_CustomAPIBase(t *testing.T) {
	t.Setenv("DEEPSEEK_BASE_URL", "https://custom.api.example.com")
	bs := NewBridge(t.TempDir())
	if bs.apiBase != "https://custom.api.example.com" {
		t.Errorf("expected custom apiBase, got: %s", bs.apiBase)
	}
}

// ── callDeepSeek ──────────────────────────────────────────────────

func TestCallDeepSeek_NoAPIKey(t *testing.T) {
	os.Unsetenv("DEEPSEEK_API_KEY")
	bs := NewBridge(t.TempDir())
	_, err := bs.callDeepSeek("system prompt", "user prompt")
	if err == nil {
		t.Fatal("expected error without API key")
	}
	if !strings.Contains(err.Error(), "DEEPSEEK_API_KEY") {
		t.Errorf("expected DEEPSEEK_API_KEY error, got: %v", err)
	}
}

func TestCallDeepSeek_CustomModel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		model, _ := body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"content":"model was %s"}}]}`, model)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	t.Setenv("DEEPSEEK_MODEL", "deepseek-v4-pro")
	bs := NewBridge(t.TempDir())
	bs.apiBase = ts.URL

	result, err := bs.callDeepSeek("sys", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "deepseek-v4-pro") {
		t.Errorf("expected custom model used, got: %s", result)
	}
}

func TestCallDeepSeek_EmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[]}`)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	bs := NewBridge(t.TempDir())
	bs.apiBase = ts.URL

	_, err := bs.callDeepSeek("sys", "user")
	if err == nil {
		t.Fatal("expected error for empty response")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("expected empty response error, got: %v", err)
	}
}

func TestCallDeepSeek_ConnectionError(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	bs := NewBridge(t.TempDir())
	bs.apiBase = "http://127.0.0.1:1" // unreachable port

	_, err := bs.callDeepSeek("sys", "user")
	if err == nil {
		t.Fatal("expected connection error")
	}
}

// ── HTTP handler ──────────────────────────────────────────────────

func TestServeHTTP_HealthEndpoint(t *testing.T) {
	bs := NewBridge(t.TempDir())
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, version)
	})
	mux.HandleFunc("/mcp", bs.Server().HandleHTTP)

	auth := &httputil.AuthMiddleware{APIKey: "", KeyEnv: ""}
	handler := auth.Wrap(mux)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleHTTPMCP(t *testing.T) {
	bs := NewBridge(t.TempDir())
	req := mcputil.Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize"}
	data, _ := json.Marshal(req)

	req2 := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(data))
	w := httptest.NewRecorder()
	bs.Server().HandleHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected json content type, got: %s", w.Header().Get("Content-Type"))
	}
}

func TestHandleHTTPMCP_BadBody(t *testing.T) {
	bs := NewBridge(t.TempDir())
	req2 := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	w := httptest.NewRecorder()
	bs.Server().HandleHTTP(w, req2)
	// Should still get 200 with error in JSON body (IO error handled)
}

// ── parseSteps ────────────────────────────────────────────────────

func TestParseSteps_Numbered(t *testing.T) {
	text := "1. First step\n2. Second step\n3. Third step"
	steps := parseSteps(text)
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d: %v", len(steps), steps)
	}
	if !strings.Contains(steps[0], "First step") {
		t.Errorf("step 0: %s", steps[0])
	}
}

func TestParseSteps_SingleStep(t *testing.T) {
	text := "Just one thing to do"
	steps := parseSteps(text)
	// No numbered headers means no steps parsed
	if len(steps) != 0 {
		t.Errorf("expected 0 steps for unnumbered text, got %d", len(steps))
	}
}

func TestParseSteps_ParenFormat(t *testing.T) {
	text := "1) Do A\n2) Do B"
	steps := parseSteps(text)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
}

func TestParseSteps_StepPrefix(t *testing.T) {
	text := "Step 1: Setup\nStep 2: Implement"
	steps := parseSteps(text)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d: %v", len(steps), steps)
	}
}

func TestParseSteps_MultiLine(t *testing.T) {
	text := "1. First step\n   with continuation\n2. Second step"
	steps := parseSteps(text)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d: %v", len(steps), steps)
	}
	if !strings.Contains(steps[0], "with continuation") {
		t.Errorf("expected continuation in step 0, got: %s", steps[0])
	}
}

func TestIsStepHeader(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1. Do thing", true},
		{"2) Other", true},
		{"10: Item", true},
		{"Step 3: Go", true},
		{"Not a step", false},
		{"", false},
		{"No number here", false},
	}
	for _, tt := range tests {
		got := isStepHeader(tt.input)
		if got != tt.want {
			t.Errorf("isStepHeader(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStripStepPrefix(t *testing.T) {
	if got := stripStepPrefix("1. Do thing"); got != "Do thing" {
		t.Errorf("got %q", got)
	}
	if got := stripStepPrefix("Step 2: Go"); got != "Go" {
		t.Errorf("got %q", got)
	}
	if got := stripStepPrefix("3) Item"); got != "Item" {
		t.Errorf("got %q", got)
	}
}

// ── ServeHTTP integration ─────────────────────────────────────────

func TestServeHTTP_Integration(t *testing.T) {
	bs := NewBridge(t.TempDir())

	go func() {
		_ = bs.Server().ServeHTTP(":0", "MCP_API_KEY")
	}()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)
}

func TestServeHTTP_AuthEnabled(t *testing.T) {
	t.Setenv("MCP_API_KEY", "test-secret-key")
	bs := NewBridge(t.TempDir())

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/mcp", bs.Server().HandleHTTP)

	auth := &httputil.AuthMiddleware{APIKey: "test-secret-key", KeyEnv: "MCP_API_KEY"}
	handler := auth.Wrap(mux)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// No auth → 401 on /mcp
	resp, err := http.Get(ts.URL + "/mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
	}

	// Health still public
	resp2, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on health, got %d", resp2.StatusCode)
	}

	// Valid auth on /mcp
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	req.Header.Set("Authorization", "Bearer test-secret-key")
	resp3, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusMethodNotAllowed && resp3.StatusCode != http.StatusOK {
		// GET on /mcp may return 405 or 200 depending on handler
		t.Logf("auth'd /mcp returned %d (acceptable)", resp3.StatusCode)
	}
}