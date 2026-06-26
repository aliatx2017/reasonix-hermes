// MCP Bridge Server — connects external AI agents (Claude Code, Codex, etc.)
// to Reasonix/DeepSeek via the Model Context Protocol.
//
// Usage:
//
//	go run ./cmd/reasonix-mcpbridge [--http] [--port 9090]
//
// This exposes tools like reasonix_run, reasonix_doctor, and plan_task
// that other agents can call to delegate work to Reasonix.
package main

import (
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

	"reasonix/internal/netclient"
	"reasonix/pkg/mcputil"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

// Bridge holds state for tool execution.
type Bridge struct {
	workDir string
	apiBase string
}

func NewBridge(workDir string) *Bridge {
	b := &Bridge{
		workDir: workDir,
		apiBase: os.Getenv("DEEPSEEK_BASE_URL"),
	}
	if b.apiBase == "" {
		b.apiBase = "https://api.deepseek.com"
	}
	return b
}

func (b *Bridge) tools() []mcputil.Tool {
	return []mcputil.Tool{
		{
			Name:        "reasonix_run",
			Description: "Execute a one-shot task using Reasonix with DeepSeek. Provide a coding/refactoring task description.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task":    map[string]any{"type": "string", "description": "The coding task to execute (e.g. 'refactor auth module')"},
					"model":   map[string]any{"type": "string", "description": "Optional model override (deepseek-flash, deepseek-pro, mimo-pro)"},
					"workdir": map[string]any{"type": "string", "description": "Working directory for the task"},
				},
				"required": []string{"task"},
			},
		},
		{
			Name:        "reasonix_doctor",
			Description: "Diagnose Reasonix configuration and connectivity. Returns setup status, model availability, and API key check.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "plan_task",
			Description: "Plan a complex multi-step task and return a structured execution plan without executing.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"objective": map[string]any{"type": "string", "description": "The high-level objective to plan"},
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
					"task": map[string]any{"type": "string", "description": "The complex task to orchestrate"},
				},
				"required": []string{"task"},
			},
		},
		{
			Name:        "get_skill",
			Description: "Read a specific Reasonix skill body by name. Returns the full skill Markdown content. Use get_skills to list available skill names first.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "Skill identifier (e.g. 'code-review', 'adversarial-review')"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "get_skills",
			Description: "List all available Reasonix skills with descriptions.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
	}
}

func (b *Bridge) handle(name string, args map[string]any) (string, error) {
	switch name {
	case "reasonix_run":
		task, _ := args["task"].(string)
		if task == "" {
			return "", fmt.Errorf("task is required")
		}
		model, _ := args["model"].(string)
		workdir, _ := args["workdir"].(string)
		if workdir == "" {
			workdir = b.workDir
		}
		return b.runReasonix(workdir, model, task)

	case "reasonix_doctor":
		return b.doctorCheck()

	case "plan_task":
		objective, _ := args["objective"].(string)
		if objective == "" {
			return "", fmt.Errorf("objective is required")
		}
		return b.planTask(objective)

	case "orchestrate_task":
		task, _ := args["task"].(string)
		if task == "" {
			return "", fmt.Errorf("task is required")
		}
		return b.orchestrateTask(task)

	case "get_skill":
		name, _ := args["name"].(string)
		if name == "" {
			return "", fmt.Errorf("name is required")
		}
		return b.getSkill(name)

	case "get_skills":
		return b.listSkills()

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func (b *Bridge) runReasonix(workdir, model, task string) (string, error) {
	// Validate workdir: must be an existing directory.
	if workdir == "" {
		workdir = "."
	}
	if info, err := os.Stat(workdir); err != nil {
		return "", fmt.Errorf("workdir %q does not exist: %w", workdir, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("workdir %q is not a directory", workdir)
	}
	// Resolve to absolute path to avoid confusion.
	if abs, err := filepath.Abs(workdir); err == nil {
		workdir = abs
	}

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

func (b *Bridge) doctorCheck() (string, error) {
	var report strings.Builder
	report.WriteString("# Reasonix Doctor Report\n\n")

	if _, err := exec.LookPath("reasonix"); err != nil {
		report.WriteString("❌ **reasonix binary**: Not found in PATH\n")
		report.WriteString("   Install: `npm i -g reasonix` or `brew install esengine/reasonix/reasonix`\n")
	} else {
		report.WriteString("✅ **reasonix binary**: Found\n")
	}

	if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
		report.WriteString("✅ **DEEPSEEK_API_KEY**: Set\n")
	} else {
		report.WriteString("⚠️  **DEEPSEEK_API_KEY**: Not set\n")
	}

	if key := os.Getenv("MCP_API_KEY"); key != "" {
		report.WriteString("✅ **MCP_API_KEY**: Set (HTTP auth enabled)\n")
	} else {
		report.WriteString("⚠️  **MCP_API_KEY**: Not set (HTTP auth disabled)\n")
	}

	if _, err := os.Stat("reasonix.toml"); err == nil {
		report.WriteString("✅ **reasonix.toml**: Found in working directory\n")
	} else {
		report.WriteString("⚠️  **reasonix.toml**: Not found (run 'reasonix setup')\n")
	}

	if _, err := exec.LookPath("go"); err == nil {
		report.WriteString("✅ **Go**: Available\n")
	} else {
		report.WriteString("⚠️  **Go**: Not available\n")
	}

	fmt.Fprintf(&report, "\n---\nBridge version: %s\nTimestamp: %s\n", version, time.Now().Format(time.RFC3339))
	return report.String(), nil
}

func (b *Bridge) planTask(objective string) (string, error) {
	systemPrompt := "You are a task planning assistant. Given an objective, produce a structured execution plan with numbered steps. Each step should have: step number, description, files to modify (if any), and dependencies on other steps. Be concise and actionable."
	plan, err := b.callDeepSeek(context.Background(), systemPrompt, objective)
	if err != nil {
		return "", fmt.Errorf("plan generation failed: %w", err)
	}
	return "# Execution Plan\n\n" + plan, nil
}

func (b *Bridge) orchestrateTask(task string) (string, error) {
	if _, err := exec.LookPath("reasonix"); err != nil {
		return "", fmt.Errorf("reasonix binary not found in PATH — required for orchestration")
	}

	// Total orchestration timeout: 15 minutes. Individual steps already
	// have their own 5-minute timeout via runReasonix.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	systemPrompt := "You are a task decomposition assistant. Break the given task into independent sub-tasks that can be executed in parallel. For each sub-task provide: name, description, scope (files to modify). Output as a numbered list. Only decompose if the task naturally splits into independent pieces; for simple tasks, return a single step."
	decomposition, err := b.callDeepSeek(ctx, systemPrompt, task)
	if err != nil {
		return "", fmt.Errorf("task decomposition failed: %w", err)
	}

	steps := parseSteps(decomposition)
	if len(steps) == 0 {
		steps = []string{task}
	}

	// Cap at 5 concurrent to prevent resource exhaustion
	const maxConcurrent = 3
	type stepResult struct {
		index int
		desc  string
		out   string
		err   error
	}

	results := make([]stepResult, len(steps))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)

	for i, step := range steps {
		wg.Add(1)
		go func(idx int, stepDesc string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			out, err := b.runReasonix(b.workDir, "", stepDesc)
			results[idx] = stepResult{index: idx, desc: stepDesc, out: out, err: err}
		}(i, step)
	}
	wg.Wait()

	var sb strings.Builder
	sb.WriteString("# Orchestration Results\n\n## Decomposition\n")
	sb.WriteString(decomposition)
	sb.WriteString("\n\n## Execution Results\n")
	for _, r := range results {
		fmt.Fprintf(&sb, "### Step %d: %s\n", r.index+1, r.desc)
		if r.err != nil {
			fmt.Fprintf(&sb, "❌ Error: %v\n", r.err)
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

func (b *Bridge) callDeepSeek(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
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
		return "", fmt.Errorf("marshal request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	url := strings.TrimRight(b.apiBase, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := netclient.DefaultClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("DeepSeek API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024)) // 4 MB limit
	if err != nil {
		return "", fmt.Errorf("read API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepSeek API status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("parse API response: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("DeepSeek API returned empty response")
	}

	return apiResp.Choices[0].Message.Content, nil
}

func parseSteps(text string) []string {
	var steps []string
	lines := strings.Split(text, "\n")

	var currentStep strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isStepHeader(trimmed) {
			if currentStep.Len() > 0 {
				steps = append(steps, strings.TrimSpace(currentStep.String()))
				currentStep.Reset()
			}
			stepText := stripStepPrefix(trimmed)
			currentStep.WriteString(stepText)
		} else if currentStep.Len() > 0 && trimmed != "" {
			currentStep.WriteString(" ")
			currentStep.WriteString(trimmed)
		}
	}
	if currentStep.Len() > 0 {
		steps = append(steps, strings.TrimSpace(currentStep.String()))
	}
	return steps
}

func isStepHeader(line string) bool {
	if len(line) == 0 {
		return false
	}
	if line[0] >= '0' && line[0] <= '9' {
		i := 1
		for i < len(line) && line[i] >= '0' && line[i] <= '9' {
			i++
		}
		if i < len(line) && (line[i] == '.' || line[i] == ')' || line[i] == ':') {
			return true
		}
	}
	if strings.HasPrefix(strings.ToUpper(line), "STEP ") {
		rest := line[len("STEP "):]
		if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
			return true
		}
	}
	return false
}

func stripStepPrefix(line string) string {
	upper := strings.ToUpper(line)
	if strings.HasPrefix(upper, "STEP ") {
		rest := line[len("STEP "):]
		i := 0
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i < len(rest) {
			return strings.TrimSpace(rest[i+1:])
		}
		return strings.TrimSpace(rest[i:])
	}
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i < len(line) {
		return strings.TrimSpace(line[i+1:])
	}
	return line
}

// skillDirs returns candidate skill directories to search, in priority order.
// Respects REASONIX_PORTABLE: when set, uses <binary_dir>/.reasonix/skills instead
// of the OS user config dir.
func (b *Bridge) skillDirs() []string {
	var dirs []string
	if os.Getenv("REASONIX_PORTABLE") != "" {
		if exe, err := os.Executable(); err == nil {
			dirs = append(dirs, filepath.Join(filepath.Dir(exe), ".reasonix", "skills"))
		}
	} else if configDir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(configDir, "reasonix", "skills"))
	}
	dirs = append(dirs,
		filepath.Join(b.workDir, ".reasonix", "skills"),
		filepath.Join(b.workDir, "skills-hub", "skills"),
	)
	return dirs
}

// findSkillFile looks for <name>.md across skill directories.  Also checks
// the <name>/SKILL.md layout used by upstream install_source.
func (b *Bridge) findSkillFile(name string) (string, error) {
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return "", fmt.Errorf("invalid skill name %q: path separators not allowed", name)
	}
	for _, dir := range b.skillDirs() {
		path := filepath.Join(dir, name+".md")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		path = filepath.Join(dir, name, "SKILL.md")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("skill %q not found in any skills directory", name)
}

// getSkill reads a skill body from disk by name.
func (b *Bridge) getSkill(name string) (string, error) {
	path, err := b.findSkillFile(name)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read skill %q: %w", name, err)
	}
	var out strings.Builder
	fmt.Fprintf(&out, "# Skill: %s\n\n", name)
	out.Write(data)
	return out.String(), nil
}

func (b *Bridge) listSkills() (string, error) {
	var out strings.Builder
	out.WriteString("# Available Skills\n\n")
	found := false
	for _, skillsDir := range b.skillDirs() {
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				// Check for <name>/SKILL.md layout
				if _, err := os.Stat(filepath.Join(skillsDir, name, "SKILL.md")); err == nil {
					fmt.Fprintf(&out, "- %s\n", name)
					found = true
				}
				continue
			}
			if strings.HasSuffix(name, ".md") {
				fmt.Fprintf(&out, "- %s\n", strings.TrimSuffix(name, ".md"))
				found = true
			}
		}
	}
	if !found {
		out.WriteString("No skills found.\n")
	}
	return out.String(), nil
}

// Server constructs the mcputil.Server wired to this bridge.
func (b *Bridge) Server() *mcputil.Server {
	return &mcputil.Server{
		Name:    "reasonix-bridge",
		Version: version,
		Tools:   b.tools(),
		Handle:  b.handle,
	}
}

// ── Main ──────────────────────────────────────────────────────────

func main() {
	port := "9090"
	httpMode := false
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--port":
			if i+1 < len(os.Args) {
				port = os.Args[i+1]
				i++
			}
		case "--http":
			httpMode = true
		}
	}

	workDir, _ := os.Getwd()
	b := NewBridge(workDir)

	srv := &mcputil.Server{
		Name:    "reasonix-bridge",
		Version: version,
		Tools:   b.tools(),
		Handle:  b.handle,
	}

	if httpMode {
		log.Fatal(srv.ServeHTTP(":"+port, "MCP_API_KEY"))
	}

	// Graceful shutdown on SIGINT/SIGTERM — close stdin so ServeStdio's
	// ReadBytes returns EOF and the process exits cleanly through all defers.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		_ = os.Stdin.Close()
	}()

	log.SetPrefix("[reasonix-bridge] ")
	log.Println("Starting in stdio mode (MCP)...")
	if err := srv.ServeStdio(); err != nil {
		log.Fatal(err)
	}
}
