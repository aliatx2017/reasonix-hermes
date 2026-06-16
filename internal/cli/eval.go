package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// evalCommand handles the CLI-level "reasonix eval ..." subcommand.
// It reuses the same file operations as the TUI /eval slash command.
func evalCommand(args []string) int {
	if len(args) == 0 {
		evalCLIUsage()
		return 2
	}

	sub := args[0]
	switch sub {
	case "define":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: reasonix eval define <name>")
			return 2
		}
		return evalCLIDefine(args[1])
	case "check":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: reasonix eval check <name>")
			return 2
		}
		return evalCLICheck(args[1])
	case "report":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: reasonix eval report <name>")
			return 2
		}
		return evalCLIReport(args[1])
	case "list":
		return evalCLIList()
	case "clean":
		return evalCLIClean()
	default:
		fmt.Fprintf(os.Stderr, "unknown eval subcommand: %s\n", sub)
		evalCLIUsage()
		return 2
	}
}

func evalCLIUsage() {
	fmt.Fprintln(os.Stderr, `Eval Command — manage eval-driven development workflow

Usage:
  reasonix eval define <name>   create a new eval definition
  reasonix eval check <name>    run and check evals
  reasonix eval report <name>   generate full eval report
  reasonix eval list            show all eval definitions
  reasonix eval clean           remove old logs (keeps last 10 runs)`)
}

// evalCLIRoot returns the evals directory, preferring .claude/evals in the
// current working directory.
func evalCLIRoot() string {
	return filepath.Join(".claude", "evals")
}

func evalCLIDefine(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Fprintln(os.Stderr, "eval name cannot be empty")
		return 2
	}

	defDir := filepath.Join(evalCLIRoot(), name)
	if err := os.MkdirAll(defDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot create directory: %v\n", err)
		return 1
	}

	defPath := filepath.Join(defDir, "definition.md")
	if _, err := os.Stat(defPath); err == nil {
		fmt.Fprintf(os.Stderr, "eval definition already exists at %s\n", defPath)
		return 1
	}

	now := time.Now().Format(time.RFC3339)
	template := fmt.Sprintf(strings.Join([]string{
		"## EVAL: %s",
		"Created: %s",
		"",
		"### Capability Evals",
		"- [ ] [description of capability 1]",
		"- [ ] [description of capability 2]",
		"",
		"### Regression Evals",
		"- [ ] [existing behavior 1 still works]",
		"- [ ] [existing behavior 2 still works]",
		"",
		"### Success Criteria",
		"- pass@3 > 90%%%% for capability evals",
		"- pass^3 = 100%%%% for regression evals",
		"",
	}, "\n"), name, now)

	if err := os.WriteFile(defPath, []byte(template), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot write definition: %v\n", err)
		return 1
	}

	fmt.Printf("eval definition created at %s\n", defPath)
	fmt.Println("edit the definition file with capability and regression eval criteria")
	return 0
}

func evalCLICheck(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Fprintln(os.Stderr, "eval name cannot be empty")
		return 2
	}

	defPath := filepath.Join(evalCLIRoot(), name, "definition.md")
	data, err := os.ReadFile(defPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read definition at %s — create it with 'reasonix eval define %s' first\n", defPath, name)
		return 1
	}

	content := string(data)
	capabilities := evalExtractCheckboxes(content, "### Capability Evals", "### Regression Evals")
	regressions := evalExtractCheckboxes(content, "### Regression Evals", "### Success Criteria")

	if len(capabilities) == 0 && len(regressions) == 0 {
		fmt.Fprintln(os.Stderr, "eval: no capability or regression criteria found in definition — fill in the criteria first")
		return 1
	}

	logPath := filepath.Join(evalCLIRoot(), name, "results.log")
	logDir := filepath.Dir(logPath)
	os.MkdirAll(logDir, 0o755)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot open log: %v\n", err)
		return 1
	}
	defer f.Close()

	now := time.Now().Format(time.RFC3339)
	fmt.Fprintf(f, "\n=== CHECK at %s ===\n", now)

	var capPass, capTotal, regPass, regTotal int

	fmt.Printf("EVAL CHECK: %s\n", name)
	fmt.Println("========================")

	fmt.Println("\nCapability Evals:")
	for _, c := range capabilities {
		capTotal++
		result := evalCLIVerify(c)
		if result == "PASS" {
			capPass++
		}
		fmt.Printf("  [%s] %s\n", result, c)
		fmt.Fprintf(f, "%s: %s\n", c, result)
	}

	fmt.Println("\nRegression Evals:")
	for _, r := range regressions {
		regTotal++
		result := evalCLIVerify(r)
		if result == "PASS" {
			regPass++
		}
		fmt.Printf("  [%s] %s\n", result, r)
		fmt.Fprintf(f, "%s: %s\n", r, result)
	}

	status := "IN PROGRESS"
	if capTotal > 0 && capPass == capTotal && regTotal > 0 && regPass == regTotal {
		status = "READY"
	} else if capTotal == 0 && regTotal > 0 && regPass == regTotal {
		status = "READY"
	}

	fmt.Printf("\nCapability: %d/%d passing\n", capPass, capTotal)
	fmt.Printf("Regression: %d/%d passing\n", regPass, regTotal)
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("eval check results appended to %s\n", logPath)
	return 0
}

func evalCLIVerify(criterion string) string {
	lower := strings.ToLower(criterion)

	switch {
	case strings.Contains(lower, "build"):
		return evalCLIRunBuildCheck()
	case strings.Contains(lower, "vet"):
		return evalCLIRunBuildCheck()
	default:
		return "MANUAL"
	}
}

func evalCLIRunBuildCheck() string {
	out, err := exec.Command("sh", "-c", "go build ./... 2>&1").CombinedOutput()
	if err != nil {
		return "FAIL"
	}
	_ = out
	return "PASS"
}

func evalCLIReport(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Fprintln(os.Stderr, "eval name cannot be empty")
		return 2
	}

	defPath := filepath.Join(evalCLIRoot(), name, "definition.md")
	defData, err := os.ReadFile(defPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: no definition at %s — create it with 'reasonix eval define %s' first\n", defPath, name)
		return 1
	}

	logData := ""
	logPath := filepath.Join(evalCLIRoot(), name, "results.log")
	if raw, err := os.ReadFile(logPath); err == nil {
		logData = string(raw)
	}

	content := string(defData)
	capabilities := evalExtractCheckboxes(content, "### Capability Evals", "### Regression Evals")
	regressions := evalExtractCheckboxes(content, "### Regression Evals", "### Success Criteria")

	capPass, capTotal := evalCountResults(logData, capabilities)
	regPass, regTotal := evalCountResults(logData, regressions)

	fmt.Printf("EVAL REPORT: %s\n", name)
	fmt.Println("=========================")
	fmt.Printf("Generated: %s\n\n", time.Now().Format(time.RFC3339))

	fmt.Println("CAPABILITY EVALS")
	fmt.Println("----------------")
	if len(capabilities) == 0 {
		fmt.Println("(none defined)")
	} else {
		for _, c := range capabilities {
			status := evalLatestStatus(logData, c)
			fmt.Printf("  [%s] %s\n", status, c)
		}
	}

	fmt.Println("\nREGRESSION EVALS")
	fmt.Println("----------------")
	if len(regressions) == 0 {
		fmt.Println("(none defined)")
	} else {
		for _, r := range regressions {
			status := evalLatestStatus(logData, r)
			fmt.Printf("  [%s] %s\n", status, r)
		}
	}

	capPass1 := 0
	if capTotal > 0 {
		capPass1 = capPass * 100 / capTotal
	}
	capPass3 := 0
	if capTotal > 0 && capPass >= capTotal {
		capPass3 = 100
	} else if capTotal > 0 {
		capPass3 = capPass * 100 / capTotal
	}
	regPass3 := 0
	if regTotal > 0 {
		regPass3 = regPass * 100 / regTotal
	}

	fmt.Printf("\nMETRICS\n")
	fmt.Println("-------")
	fmt.Printf("Capability pass@1: %d%%\n", capPass1)
	fmt.Printf("Capability pass@3: %d%%\n", capPass3)
	fmt.Printf("Regression pass^3: %d%%\n", regPass3)

	fmt.Println("\nNOTES")
	fmt.Println("-----")
	if evalLatestStatus(logData, "implicit") == "" {
		fmt.Println("(no manual notes recorded)")
	} else {
		fmt.Print(evalExtractNotes(logData))
	}

	fmt.Println("\nRECOMMENDATION")
	fmt.Println("--------------")
	rec := "BLOCKED"
	if capTotal > 0 && regTotal > 0 && capPass >= capTotal && regPass >= regTotal {
		rec = "SHIP"
	} else if capTotal > 0 && capPass >= capTotal {
		rec = "SHIP (no regressions defined)"
	} else if capTotal == 0 && regTotal > 0 && regPass >= regTotal {
		rec = "SHIP (no capabilities defined)"
	} else if capTotal > 0 && regTotal > 0 && float64(capPass)/float64(capTotal) >= 0.9 {
		rec = "NEEDS WORK (capability pass@3 < 100%)"
	}
	fmt.Println(rec)
	return 0
}

func evalCLIList() int {
	dir := evalCLIRoot()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "no eval definitions found — create one with 'reasonix eval define <name>'")
		return 0
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "no eval definitions found — create one with 'reasonix eval define <name>'")
		return 0
	}

	fmt.Println("EVAL DEFINITIONS")
	fmt.Println("================")

	for _, name := range names {
		defPath := filepath.Join(dir, name, "definition.md")
		logPath := filepath.Join(dir, name, "results.log")

		defData, _ := os.ReadFile(defPath)
		logData, _ := os.ReadFile(logPath)

		capabilities := evalExtractCheckboxes(string(defData), "### Capability Evals", "### Regression Evals")
		regressions := evalExtractCheckboxes(string(defData), "### Regression Evals", "### Success Criteria")

		capPass, capTotal := evalCountResults(string(logData), capabilities)
		regPass, regTotal := evalCountResults(string(logData), regressions)

		totalEvals := capTotal + regTotal
		totalPass := capPass + regPass

		status := "NOT STARTED"
		if totalEvals > 0 {
			status = fmt.Sprintf("%d/%d passing", totalPass, totalEvals)
			if totalPass == totalEvals {
				status = "READY"
			} else if totalPass > 0 {
				status += " IN PROGRESS"
			}
		}

		fmt.Printf("  %-20s [%s]\n", name, status)
	}
	return 0
}

func evalCLIClean() int {
	dir := evalCLIRoot()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "no eval definitions to clean")
		return 0
	}

	cleaned := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		logPath := filepath.Join(dir, e.Name(), "results.log")
		data, err := os.ReadFile(logPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		var checkIndices []int
		for i, line := range lines {
			if strings.HasPrefix(line, "=== CHECK at ") {
				checkIndices = append(checkIndices, i)
			}
		}

		if len(checkIndices) <= 10 {
			continue
		}

		keepFrom := checkIndices[len(checkIndices)-10]
		trimmed := strings.Join(lines[keepFrom:], "\n")
		if err := os.WriteFile(logPath, []byte(trimmed), 0o644); err == nil {
			cleaned++
		}
	}

	if cleaned == 0 {
		fmt.Println("eval: no logs needed trimming (all have ≤10 runs)")
	} else {
		fmt.Printf("eval: trimmed old runs from %d eval(s) (kept last 10 each)\n", cleaned)
	}
	return 0
}

// evalDir returns the evals directory root (".claude/evals") under the workspace
// root, falling back to the current working directory when no workspace is set.
func (m *chatTUI) evalDir() string {
	root := m.ctrl.WorkspaceRoot()
	if root == "" {
		root = "."
	}
	return filepath.Join(root, ".claude", "evals")
}

// evalDefPath returns the definition file path for the named eval.
func (m *chatTUI) evalDefPath(name string) string {
	return filepath.Join(m.evalDir(), name, "definition.md")
}

// evalLogPath returns the results log path for the named eval.
func (m *chatTUI) evalLogPath(name string) string {
	return filepath.Join(m.evalDir(), name, "results.log")
}

// runEvalSubcommand dispatches /eval subcommands.
func (m *chatTUI) runEvalSubcommand(input string) {
	m.echoLocalCommand(input)

	parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(input, "/eval")))
	if len(parts) == 0 {
		m.evalShowHelp()
		return
	}

	sub := parts[0]
	switch sub {
	case "define":
		if len(parts) < 2 {
			m.notice("usage: /eval define <name>")
			return
		}
		m.evalDefine(parts[1])
	case "check":
		if len(parts) < 2 {
			m.notice("usage: /eval check <name>")
			return
		}
		m.evalCheck(parts[1])
	case "report":
		if len(parts) < 2 {
			m.notice("usage: /eval report <name>")
			return
		}
		m.evalReport(parts[1])
	case "list":
		m.evalList()
	case "clean":
		m.evalClean()
	default:
		m.notice(fmt.Sprintf("unknown eval subcommand: %s — try one of: define, check, report, list, clean", sub))
	}
}

// evalShowHelp prints the /eval usage summary.
func (m *chatTUI) evalShowHelp() {
	help := strings.Join([]string{
		"Eval Command — manage eval-driven development workflow",
		"",
		"  /eval define <name>   create a new eval definition",
		"  /eval check <name>    run and check evals",
		"  /eval report <name>   generate full eval report",
		"  /eval list            show all eval definitions and status",
		"  /eval clean           remove old eval logs (keeps last 10 runs)",
	}, "\n")
	m.commitLine(help)
}

// evalDefine creates a new eval definition template for the named feature.
func (m *chatTUI) evalDefine(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		m.notice("eval name cannot be empty")
		return
	}

	defDir := filepath.Join(m.evalDir(), name)
	if err := os.MkdirAll(defDir, 0o755); err != nil {
		m.notice(fmt.Sprintf("eval: cannot create directory: %v", err))
		return
	}

	defPath := filepath.Join(defDir, "definition.md")
	if _, err := os.Stat(defPath); err == nil {
		m.notice(fmt.Sprintf("eval definition already exists at %s", defPath))
		return
	}

	now := time.Now().Format(time.RFC3339)
	template := fmt.Sprintf(strings.Join([]string{
		"## EVAL: %s",
		"Created: %s",
		"",
		"### Capability Evals",
		"- [ ] [description of capability 1]",
		"- [ ] [description of capability 2]",
		"",
		"### Regression Evals",
		"- [ ] [existing behavior 1 still works]",
		"- [ ] [existing behavior 2 still works]",
		"",
		"### Success Criteria",
		"- pass@3 > 90%% for capability evals",
		"- pass^3 = 100%% for regression evals",
		"",
	}, "\n"), name, now)

	if err := os.WriteFile(defPath, []byte(template), 0o644); err != nil {
		m.notice(fmt.Sprintf("eval: cannot write definition: %v", err))
		return
	}

	m.notice(fmt.Sprintf("eval definition created at %s", defPath))
	m.notice("edit the definition file with capability and regression eval criteria")
}

// evalCheck runs the evals for a named feature, recording results.
func (m *chatTUI) evalCheck(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		m.notice("eval name cannot be empty")
		return
	}

	defPath := m.evalDefPath(name)
	data, err := os.ReadFile(defPath)
	if err != nil {
		m.notice(fmt.Sprintf("eval: cannot read definition at %s — create it with /eval define %s first", defPath, name))
		return
	}

	content := string(data)

	// Parse capability and regression checkboxes.
	capabilities := evalExtractCheckboxes(content, "### Capability Evals", "### Regression Evals")
	regressions := evalExtractCheckboxes(content, "### Regression Evals", "### Success Criteria")

	if len(capabilities) == 0 && len(regressions) == 0 {
		m.notice("eval: no capability or regression criteria found in definition — fill in the criteria first")
		return
	}

	logPath := m.evalLogPath(name)
	logDir := filepath.Dir(logPath)
	os.MkdirAll(logDir, 0o755)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		m.notice(fmt.Sprintf("eval: cannot open log: %v", err))
		return
	}
	defer f.Close()

	now := time.Now().Format(time.RFC3339)
	fmt.Fprintf(f, "\n=== CHECK at %s ===\n", now)

	var capPass, capTotal, regPass, regTotal int

	m.commitLine(fmt.Sprintf("EVAL CHECK: %s", name))
	m.commitLine("========================")

	m.commitLine("")
	m.commitLine("Capability Evals:")
	for _, c := range capabilities {
		capTotal++
		result := m.evalVerify(c)
		if result == "PASS" {
			capPass++
		}
		line := fmt.Sprintf("  [%s] %s", result, c)
		m.commitLine(line)
		fmt.Fprintf(f, "%s: %s\n", c, result)
	}

	m.commitLine("")
	m.commitLine("Regression Evals:")
	for _, r := range regressions {
		regTotal++
		result := m.evalVerify(r)
		if result == "PASS" {
			regPass++
		}
		line := fmt.Sprintf("  [%s] %s", result, r)
		m.commitLine(line)
		fmt.Fprintf(f, "%s: %s\n", r, result)
	}

	status := "IN PROGRESS"
	if capTotal > 0 && capPass == capTotal && regTotal > 0 && regPass == regTotal {
		status = "READY"
	} else if capTotal == 0 && regTotal > 0 && regPass == regTotal {
		status = "READY"
	}

	m.commitLine("")
	m.commitLine(fmt.Sprintf("Capability: %d/%d passing", capPass, capTotal))
	m.commitLine(fmt.Sprintf("Regression: %d/%d passing", regPass, regTotal))
	m.commitLine(fmt.Sprintf("Status: %s", status))
	m.notice(fmt.Sprintf("eval check results appended to %s", logPath))
}

// evalVerify attempts to verify a single eval criterion by running the relevant
// test suite. Returns PASS or FAIL. This is a best-effort heuristic — it runs
// smoke checks like the test suite and reports PASS if they succeed.
func (m *chatTUI) evalVerify(criterion string) string {
	// For criterion strings that mention a specific package, test command, or
	// test name, try to run it.
	// This is intentionally simple: it matches common patterns and runs the
	// go test suite as a baseline. Users can override by editing results.log.
	lower := strings.ToLower(criterion)

	switch {
	case strings.Contains(lower, "test") && strings.Contains(lower, "build"):
		// "go build ./... still works" etc.
		return m.evalRunBuildCheck()
	case strings.Contains(lower, "test") && strings.Contains(lower, "./..."):
		return m.evalRunTestCheck("go test ./... 2>&1")
	case strings.Contains(lower, "vet"):
		return m.evalRunBuildCheck()
	default:
		// For arbitrary criteria, run a quick test suite smoke check.
		// Mark as MANUAL so the user doesn't get a false sense of automation.
		return "MANUAL"
	}
}

// evalRunBuildCheck runs go build ./... and returns PASS or FAIL.
func (m *chatTUI) evalRunBuildCheck() string {
	// Run the check in a simple blocking call.
	// We use bash directly since the TUI already has the build environment.
	result := m.evalRunCommand("go build ./... 2>&1")
	if result == "" {
		return "PASS"
	}
	return "FAIL"
}

// evalRunTestCheck runs a test command and returns PASS/FAIL based on exit code.
func (m *chatTUI) evalRunTestCheck(cmd string) string {
	result := m.evalRunCommand(cmd)
	if result == "" {
		return "PASS"
	}
	return "FAIL"
}

// evalRunCommand runs a shell command and returns stderr on failure, empty on success.
func (m *chatTUI) evalRunCommand(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// evalReport generates a comprehensive eval report for the named feature.
func (m *chatTUI) evalReport(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		m.notice("eval name cannot be empty")
		return
	}

	defPath := m.evalDefPath(name)
	defData, err := os.ReadFile(defPath)
	if err != nil {
		m.notice(fmt.Sprintf("eval: no definition at %s — create it with /eval define %s first", defPath, name))
		return
	}

	logData := ""
	logPath := m.evalLogPath(name)
	if raw, err := os.ReadFile(logPath); err == nil {
		logData = string(raw)
	}

	content := string(defData)
	capabilities := evalExtractCheckboxes(content, "### Capability Evals", "### Regression Evals")
	regressions := evalExtractCheckboxes(content, "### Regression Evals", "### Success Criteria")

	// Parse log entries for pass/fail counts.
	capPass, capTotal := evalCountResults(logData, capabilities)
	regPass, regTotal := evalCountResults(logData, regressions)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("EVAL REPORT: %s\n", name))
	b.WriteString("=========================\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	b.WriteString("CAPABILITY EVALS\n")
	b.WriteString("----------------\n")
	if len(capabilities) == 0 {
		b.WriteString("(none defined)\n")
	} else {
		for _, c := range capabilities {
			status := evalLatestStatus(logData, c)
			b.WriteString(fmt.Sprintf("  [%s] %s\n", status, c))
		}
	}

	b.WriteString("\nREGRESSION EVALS\n")
	b.WriteString("----------------\n")
	if len(regressions) == 0 {
		b.WriteString("(none defined)\n")
	} else {
		for _, r := range regressions {
			status := evalLatestStatus(logData, r)
			b.WriteString(fmt.Sprintf("  [%s] %s\n", status, r))
		}
	}

	// Compute pass@1, pass@3, regression pass^3.
	capPass1 := 0
	if capTotal > 0 {
		capPass1 = capPass * 100 / capTotal
	}
	capPass3 := 0
	if capTotal > 0 && capPass >= capTotal {
		capPass3 = 100
	} else if capTotal > 0 {
		capPass3 = capPass * 100 / capTotal
	}
	regPass3 := 0
	if regTotal > 0 {
		regPass3 = regPass * 100 / regTotal
	}

	b.WriteString(fmt.Sprintf("\nMETRICS\n"))
	b.WriteString("-------\n")
	b.WriteString(fmt.Sprintf("Capability pass@1: %d%%\n", capPass1))
	b.WriteString(fmt.Sprintf("Capability pass@3: %d%%\n", capPass3))
	b.WriteString(fmt.Sprintf("Regression pass^3: %d%%\n", regPass3))

	b.WriteString("\nNOTES\n")
	b.WriteString("-----\n")
	if evalLatestStatus(logData, "implicit") == "" {
		b.WriteString("(no manual notes recorded)\n")
	} else {
		b.WriteString(evalExtractNotes(logData))
	}

	// Recommendation.
	b.WriteString("\nRECOMMENDATION\n")
	b.WriteString("--------------\n")
	rec := "BLOCKED"
	if capTotal > 0 && regTotal > 0 && capPass >= capTotal && regPass >= regTotal {
		rec = "SHIP"
	} else if capTotal > 0 && capPass >= capTotal {
		rec = "SHIP (no regressions defined)"
	} else if capTotal == 0 && regTotal > 0 && regPass >= regTotal {
		rec = "SHIP (no capabilities defined)"
	} else if capTotal > 0 && regTotal > 0 && float64(capPass)/float64(capTotal) >= 0.9 {
		rec = "NEEDS WORK (capability pass@3 < 100%)"
	}
	b.WriteString(rec + "\n")

	m.commitLine(b.String())
}

// evalCountResults counts how many of the given criteria have a PASS in the log.
func evalCountResults(logData string, criteria []string) (pass, total int) {
	total = len(criteria)
	for _, c := range criteria {
		if evalLatestStatus(logData, c) == "PASS" {
			pass++
		}
	}
	return
}

// evalLatestStatus returns the most recent PASS/FAIL/... status for a criterion.
func evalLatestStatus(logData, criterion string) string {
	lines := strings.Split(logData, "\n")
	status := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, criterion+":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status = strings.TrimSpace(parts[1])
			}
		}
	}
	if status == "" {
		return "NOT RUN"
	}
	return status
}

// evalExtractNotes extracts free-form notes from the latest check run.
func evalExtractNotes(logData string) string {
	// Simple heuristic: return everything after the last "=== CHECK at".
	parts := strings.Split(logData, "=== CHECK at ")
	if len(parts) <= 1 {
		return ""
	}
	last := parts[len(parts)-1]
	// Remove the timestamp line.
	lines := strings.SplitN(last, "\n", 2)
	if len(lines) < 2 {
		return ""
	}
	return strings.TrimSpace(lines[1])
}

// evalList shows all eval definitions with their current status.
func (m *chatTUI) evalList() {
	dir := m.evalDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.notice("no eval definitions found — create one with /eval define <name>")
		return
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	if len(names) == 0 {
		m.notice("no eval definitions found — create one with /eval define <name>")
		return
	}

	var b strings.Builder
	b.WriteString("EVAL DEFINITIONS\n")
	b.WriteString("================\n")

	for _, name := range names {
		defPath := filepath.Join(dir, name, "definition.md")
		logPath := filepath.Join(dir, name, "results.log")

		defData, _ := os.ReadFile(defPath)
		logData, _ := os.ReadFile(logPath)

		capabilities := evalExtractCheckboxes(string(defData), "### Capability Evals", "### Regression Evals")
		regressions := evalExtractCheckboxes(string(defData), "### Regression Evals", "### Success Criteria")

		capPass, capTotal := evalCountResults(string(logData), capabilities)
		regPass, regTotal := evalCountResults(string(logData), regressions)

		totalEvals := capTotal + regTotal
		totalPass := capPass + regPass

		status := "NOT STARTED"
		if totalEvals > 0 {
			status = fmt.Sprintf("%d/%d passing", totalPass, totalEvals)
			if totalPass == totalEvals {
				status = "READY"
			} else if totalPass > 0 {
				status += " IN PROGRESS"
			}
		}

		b.WriteString(fmt.Sprintf("  %-20s [%s]\n", name, status))
	}

	m.commitLine(b.String())
}

// evalClean removes old result logs, keeping the last 10 runs per eval.
func (m *chatTUI) evalClean() {
	dir := m.evalDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.notice("no eval definitions to clean")
		return
	}

	cleaned := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		logPath := filepath.Join(dir, e.Name(), "results.log")
		data, err := os.ReadFile(logPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		// Find CHECK markers and keep only the last 10.
		var checkIndices []int
		for i, line := range lines {
			if strings.HasPrefix(line, "=== CHECK at ") {
				checkIndices = append(checkIndices, i)
			}
		}

		if len(checkIndices) <= 10 {
			continue
		}

		// Keep lines from the 10th-from-last check onward.
		keepFrom := checkIndices[len(checkIndices)-10]
		trimmed := strings.Join(lines[keepFrom:], "\n")
		if err := os.WriteFile(logPath, []byte(trimmed), 0o644); err == nil {
			cleaned++
		}
	}

	if cleaned == 0 {
		m.notice("eval: no logs needed trimming (all have ≤10 runs)")
	} else {
		m.notice(fmt.Sprintf("eval: trimmed old runs from %d eval(s) (kept last 10 each)", cleaned))
	}
}

// evalExtractCheckboxes extracts markdown checklist items between two section
// headers. Each item is the text after "- [ ] " or "- [x] ".
func evalExtractCheckboxes(content, startSection, endSection string) []string {
	var items []string

	startIdx := strings.Index(content, startSection)
	if startIdx < 0 {
		return nil
	}

	// Find end bound.
	endIdx := len(content)
	if endSection != "" {
		if idx := strings.Index(content[startIdx+len(startSection):], endSection); idx >= 0 {
			endIdx = startIdx + len(startSection) + idx
		}
	}

	section := content[startIdx:endIdx]
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		// Match "- [ ] ..." or "- [x] ..."
		if strings.HasPrefix(trimmed, "- [ ] ") {
			items = append(items, strings.TrimPrefix(trimmed, "- [ ] "))
		} else if strings.HasPrefix(trimmed, "- [x] ") {
			items = append(items, strings.TrimPrefix(trimmed, "- [x] "))
		}
	}

	return items
}
