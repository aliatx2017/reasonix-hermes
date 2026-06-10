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
}

// NewBridgeServer creates a new bridge server.
func NewBridgeServer(workDir string) *BridgeServer {
	bs := &BridgeServer{
		workDir: workDir,
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
func (bs *BridgeServer) ServeHTTP(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", bs.handleHTTPMCP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, version)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
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
	// TODO: integrate with reasonix plan mode once --plan flag is available
	return "", fmt.Errorf("plan_task is not yet implemented — use reasonix_run for one-shot task execution")
}

func (bs *BridgeServer) orchestrateTask(task string) (string, error) {
	// TODO: implement multi-step orchestration with sub-agent decomposition
	return "", fmt.Errorf("orchestrate_task is not yet implemented — use reasonix_run for one-shot task execution")
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
