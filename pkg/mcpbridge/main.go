// MCP Bridge Server — connects external AI agents (Claude Code, Codex, etc.)
// to Reasonix/DeepSeek via the Model Context Protocol.
//
// Usage:
//
//	go run ./pkg/mcpbridge [--port 9090]
//
// This exposes tools like reasonix_run, reasonix_doctor, and plan_task
// that other agents can call to delegate work to Reasonix.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"reasonix/pkg/httputil"
)

const version = "1.5.0"

// ToolDefinition is an MCP tool exposed by the bridge.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// BridgeServer handles MCP requests over stdio or HTTP.
type BridgeServer struct {
	mu      sync.Mutex
	tools   []ToolDefinition
	workDir string
	apiBase string // DeepSeek API base URL, defaults to https://api.deepseek.com
}

// NewBridgeServer creates a new bridge server.
func NewBridgeServer(workDir string) *BridgeServer {
	bs := &BridgeServer{
		workDir: workDir,
		apiBase: os.Getenv("DEEPSEEK_BASE_URL"),
	}
	if bs.apiBase == "" {
		bs.apiBase = "https://api.deepseek.com"
	}
	bs.registerTools()
	return bs
}

func (bs *BridgeServer) registerTools() {
	bs.tools = []ToolDefinition{
		{
			Name:        "reasonix_run",
			Description: "Execute a one-shot task using Reasonix with DeepSeek. Provide a coding/refactoring task description.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task": map[string]any{
						"type":        "string",
						"description": "The coding task to execute (e.g. 'refactor auth module')",
					},
					"model": map[string]any{
						"type":        "string",
						"description": "Optional model override (deepseek-flash, deepseek-pro, mimo-pro)",
					},
					"workdir": map[string]any{
						"type":        "string",
						"description": "Working directory for the task",
					},
				},
				"required": []string{"task"},
			},
		},
		{
			Name:        "reasonix_doctor",
			Description: "Diagnose Reasonix configuration and connectivity. Returns setup status, model availability, and API key check.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "plan_task",
			Description: "Plan a complex multi-step task and return a structured execution plan without executing.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"objective": map[string]any{
						"type":        "string",
						"description": "The high-level objective to plan",
					},
				},
				"required": []string{"objective"},
			},
		},
		{
			Name:        "orchestrate_task",
			Description: "Decompose a complex task into parallel sub-tasks, execute them, and merge results.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task": map[string]any{
						"type":        "string",
						"description": "The complex task to orchestrate",
					},
				},
				"required": []string{"task"},
			},
		},
		{
			Name:        "get_skills",
			Description: "List all available Reasonix skills with descriptions.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

// ServeStdio runs the bridge over stdio (for Claude Code MCP integration).
func (bs *BridgeServer) ServeStdio() error {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	log.SetOutput(os.Stderr) // keep logs out of the stdio pipe

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read stdin: %w", err)
		}

		resp := bs.handleMessage(line)
		if resp == nil {
			// notifications/initialized — no response per MCP spec
			continue
		}
		resp = append(resp, '\n')
		if _, err := writer.Write(resp); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
	}
}

// ServeHTTP runs the bridge over HTTP (Streamable MCP transport).
// If MCP_API_KEY env var is set, requires Authorization: Bearer <key> header
// for all endpoints except /health.
func (bs *BridgeServer) ServeHTTP(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", bs.handleHTTPMCP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, version)
	})

	auth := &httputil.AuthMiddleware{
		APIKey: httputil.LoadAPIKey("MCP_API_KEY"),
		KeyEnv: "MCP_API_KEY",
	}
	handler := auth.Wrap(mux)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Graceful shutdown on signal
	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
		<-sc
		log.Println("Shutting down HTTP server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("MCP Bridge HTTP server listening on %s", addr)
	return srv.ListenAndServe()
}

func (bs *BridgeServer) handleHTTPMCP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	resp := bs.handleMessage(body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// ── JSON-RPC handling ──────────────────────────────────────────────

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (bs *BridgeServer) handleMessage(data []byte) []byte {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	var req jsonRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return bs.errorResponse(0, -32700, "Parse error: "+err.Error())
	}

	switch req.Method {
	case "initialize":
		return bs.handleInitialize(req)
	case "notifications/initialized":
		return nil // no response needed
	case "tools/list":
		return bs.handleListTools(req)
	case "tools/call":
		return bs.handleCallTool(req, req.Params)
	default:
		return bs.errorResponse(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func (bs *BridgeServer) handleInitialize(req jsonRPCRequest) []byte {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "reasonix-bridge",
			"version": version,
		},
		"capabilities": map[string]any{
			"tools": map[string]bool{},
		},
	}
	return bs.successResponse(req.ID, result)
}

func (bs *BridgeServer) handleListTools(req jsonRPCRequest) []byte {
	result := map[string]any{
		"tools": bs.tools,
	}
	return bs.successResponse(req.ID, result)
}

func (bs *BridgeServer) handleCallTool(req jsonRPCRequest, params json.RawMessage) []byte {
	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return bs.errorResponse(req.ID, -32602, "Invalid params: "+err.Error())
	}

	result, err := bs.executeTool(call.Name, call.Arguments)
	if err != nil {
		return bs.errorResponse(req.ID, -32000, err.Error())
	}

	content := []map[string]string{
		{"type": "text", "text": result},
	}
	res := map[string]any{"content": content}
	return bs.successResponse(req.ID, res)
}

func (bs *BridgeServer) executeTool(name string, args map[string]any) (string, error) {
	switch name {
	case "reasonix_run":
		task, _ := args["task"].(string)
		if task == "" {
			return "", fmt.Errorf("task is required")
		}
		model, _ := args["model"].(string)
		workdir, _ := args["workdir"].(string)
		if workdir == "" {
			workdir = bs.workDir
		}
		return bs.runReasonix(workdir, model, task)

	case "reasonix_doctor":
		return bs.doctorCheck()

	case "plan_task":
		objective, _ := args["objective"].(string)
		return bs.planTask(objective)

	case "orchestrate_task":
		task, _ := args["task"].(string)
		return bs.orchestrateTask(task)

	case "get_skills":
		return bs.listSkills()

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func (bs *BridgeServer) runReasonix(workdir, model, task string) (string, error) {
	args := []string{"run", task}
	if model != "" {
		args = append(args, "--model", model)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "reasonix", args...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("reasonix timed out after 5 minutes")
	}
	if err != nil {
		return fmt.Sprintf("Reasonix execution error: %v\n%s", err, string(out)), nil
	}
	return string(out), nil
}

func (bs *BridgeServer) doctorCheck() (string, error) {
	var report strings.Builder
	report.WriteString("# Reasonix Doctor Report\n\n")

	// Check if reasonix binary exists
	if _, err := exec.LookPath("reasonix"); err != nil {
		report.WriteString("❌ **reasonix binary**: Not found in PATH\n")
		report.WriteString("   Install: `npm i -g reasonix` or `brew install esengine/reasonix/reasonix`\n")
	} else {
		report.WriteString("✅ **reasonix binary**: Found\n")
	}

	// Check DeepSeek API key
	if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
		report.WriteString("✅ **DEEPSEEK_API_KEY**: Set (length: " + fmt.Sprintf("%d", len(key)) + " chars)\n")
	} else {
		report.WriteString("⚠️  **DEEPSEEK_API_KEY**: Not set\n")
	}

	// Check MCP API key (auth)
	if key := os.Getenv("MCP_API_KEY"); key != "" {
		report.WriteString("✅ **MCP_API_KEY**: Set (HTTP auth enabled)\n")
	} else {
		report.WriteString("⚠️  **MCP_API_KEY**: Not set (HTTP auth disabled — set to enable Bearer token auth)\n")
	}

	// Check config
	if _, err := os.Stat("reasonix.toml"); err == nil {
		report.WriteString("✅ **reasonix.toml**: Found in working directory\n")
	} else {
		report.WriteString("⚠️  **reasonix.toml**: Not found (run 'reasonix setup')\n")
	}

	// Check Go
	if _, err := exec.LookPath("go"); err == nil {
		report.WriteString("✅ **Go**: Available\n")
	} else {
		report.WriteString("⚠️  **Go**: Not available (optional, needed for Go projects)\n")
	}

	report.WriteString("\n---\n")
	report.WriteString(fmt.Sprintf("Bridge version: %s\n", version))
	report.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339)))

	return report.String(), nil
}

func (bs *BridgeServer) planTask(objective string) (string, error) {
	systemPrompt := "You are a task planning assistant. Given an objective, produce a structured execution plan with numbered steps. Each step should have: step number, description, files to modify (if any), and dependencies on other steps. Be concise and actionable."
	plan, err := bs.callDeepSeek(systemPrompt, objective)
	if err != nil {
		return "", fmt.Errorf("plan generation failed: %w", err)
	}
	return "# Execution Plan\n\n" + plan, nil
}

func (bs *BridgeServer) orchestrateTask(task string) (string, error) {
	// Check reasonix binary exists
	if _, err := exec.LookPath("reasonix"); err != nil {
		return "", fmt.Errorf("reasonix binary not found in PATH — required for orchestration")
	}

	// Decompose task into independent sub-tasks
	systemPrompt := "You are a task decomposition assistant. Break the given task into independent sub-tasks that can be executed in parallel. For each sub-task provide: name, description, scope (files to modify). Output as a numbered list. Only decompose if the task naturally splits into independent pieces; for simple tasks, return a single step."
	decomposition, err := bs.callDeepSeek(systemPrompt, task)
	if err != nil {
		return "", fmt.Errorf("task decomposition failed: %w", err)
	}

	// Parse steps from numbered lines like "1." or "Step 1:" etc.
	steps := parseSteps(decomposition)
	if len(steps) == 0 {
		steps = []string{task}
	}

	// Execute steps in parallel (max 3 concurrent)
	type stepResult struct {
		index int
		desc  string
		out   string
		err   error
	}

	results := make([]stepResult, len(steps))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)

	for i, step := range steps {
		wg.Add(1)
		go func(idx int, stepDesc string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			out, err := bs.runReasonix(bs.workDir, "", stepDesc)
			if err != nil {
				results[idx] = stepResult{index: idx, desc: stepDesc, err: err}
			} else {
				results[idx] = stepResult{index: idx, desc: stepDesc, out: out}
			}
		}(i, step)
	}
	wg.Wait()

	// Build summary
	var sb strings.Builder
	sb.WriteString("# Orchestration Results\n\n")
	sb.WriteString("## Decomposition\n")
	sb.WriteString(decomposition)
	sb.WriteString("\n\n## Execution Results\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("### Step %d: %s\n", r.index+1, r.desc))
		if r.err != nil {
			sb.WriteString(fmt.Sprintf("❌ Error: %v\n", r.err))
		} else {
			sb.WriteString(r.out)
			if !strings.HasSuffix(r.out, "\n") {
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// callDeepSeek sends a chat completion request to the DeepSeek API and returns
// the assistant's response content.
func (bs *BridgeServer) callDeepSeek(systemPrompt, userPrompt string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY not set — required for plan generation")
	}

	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-v4-flash"
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":  2048,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	url := strings.TrimRight(bs.apiBase, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("DeepSeek API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepSeek API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("DeepSeek API returned empty response")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// parseSteps extracts individual steps from a numbered list response.
// Handles formats like "1." "1)" "Step 1:" etc.
func parseSteps(text string) []string {
	var steps []string
	lines := strings.Split(text, "\n")

	var currentStep strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Detect numbered step headers: "1." "1)" "Step 1:" "Step 1 -" etc.
		if isStepHeader(trimmed) {
			if currentStep.Len() > 0 {
				steps = append(steps, strings.TrimSpace(currentStep.String()))
				currentStep.Reset()
			}
			// Remove the step number prefix and keep the description
			stepText := stripStepPrefix(trimmed)
			currentStep.WriteString(stepText)
		} else if currentStep.Len() > 0 {
			// Continuation of current step — only include if not another step header
			if trimmed != "" {
				currentStep.WriteString(" ")
				currentStep.WriteString(trimmed)
			}
		}
	}
	if currentStep.Len() > 0 {
		steps = append(steps, strings.TrimSpace(currentStep.String()))
	}

	return steps
}

// isStepHeader checks if a line starts a new numbered step.
func isStepHeader(line string) bool {
	if len(line) == 0 {
		return false
	}
	// Match "1." "1)" "1:" patterns
	if line[0] >= '0' && line[0] <= '9' {
		i := 1
		for i < len(line) && line[i] >= '0' && line[i] <= '9' {
			i++
		}
		if i < len(line) && (line[i] == '.' || line[i] == ')' || line[i] == ':') {
			return true
		}
	}
	// Match "Step 1:" "Step 1 -" patterns
	if strings.HasPrefix(strings.ToUpper(line), "STEP ") {
		rest := line[len("STEP "):]
		if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
			return true
		}
	}
	return false
}

// stripStepPrefix removes the step number prefix from a line.
func stripStepPrefix(line string) string {
	// Handle "Step N:" / "Step N -" / "Step N." patterns
	upper := strings.ToUpper(line)
	if strings.HasPrefix(upper, "STEP ") {
		rest := line[len("STEP "):]
		i := 0
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i < len(rest) {
			return strings.TrimSpace(rest[i+1:]) // skip the delimiter after the number
		}
		return strings.TrimSpace(rest[i:])
	}
	// Handle "N." / "N)" / "N:" patterns
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i < len(line) {
		return strings.TrimSpace(line[i+1:]) // skip the delimiter
	}
	return line
}

func (bs *BridgeServer) listSkills() (string, error) {
	// Resolve skills directory: XDG_CONFIG_HOME > ~/.config/reasonix/skills/ > .reasonix/skills/
	skillsDir := ""
	if configDir, err := os.UserConfigDir(); err == nil {
		skillsDir = filepath.Join(configDir, "reasonix", "skills")
	}
	if _, err := os.Stat(skillsDir); err != nil {
		// Fall back to project-local skills directory
		skillsDir = filepath.Join(bs.workDir, ".reasonix", "skills")
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return "", fmt.Errorf("no skills directory found at %q (create .reasonix/skills/ or ~/.config/reasonix/skills/)", skillsDir)
	}

	var out strings.Builder
	out.WriteString("# Available Skills\n\n")
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fmt.Fprintf(&out, "- %s\n", strings.TrimSuffix(e.Name(), ".md"))
	}
	if out.Len() == len("# Available Skills\n\n") {
		fmt.Fprintf(&out, "No skills found in %s\n", skillsDir)
	}
	return out.String(), nil
}

// ── JSON-RPC helpers ──────────────────────────────────────────────

func (bs *BridgeServer) successResponse(id int, result any) []byte {
	r, _ := json.Marshal(result)
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  r,
	}
	data, _ := json.Marshal(resp)
	return data
}

func (bs *BridgeServer) errorResponse(id, code int, message string) []byte {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	return data
}

// ── Main ──────────────────────────────────────────────────────────

func main() {
	port := "9090"
	if len(os.Args) > 2 && os.Args[1] == "--port" {
		port = os.Args[2]
	}

	workDir, _ := os.Getwd()
	bs := NewBridgeServer(workDir)

	// Check if running as stdio MCP server (no --http flag)
	if len(os.Args) > 1 && os.Args[1] == "--http" {
		log.Fatal(bs.ServeHTTP(":" + port))
	}

	// Default: stdio mode for Claude Code MCP integration
	log.SetPrefix("[reasonix-bridge] ")
	log.Println("Starting in stdio mode (MCP)...")
	if err := bs.ServeStdio(); err != nil {
		log.Fatal(err)
	}
}


