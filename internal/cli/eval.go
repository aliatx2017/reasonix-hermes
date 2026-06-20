package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"reasonix/internal/eval"
)

// ── Output abstraction ───────────────────────────────────────────────────────

// evalOutput abstracts the two output channels shared by CLI and TUI eval code:
// lines (transcript/output) and notices (errors/alerts).
type evalOutput struct {
	line   func(string)
	notice func(string)
}

// cliOutput writes lines to stdout and notices to stderr.
func cliOutput() evalOutput {
	return evalOutput{
		line:   func(s string) { fmt.Println(s) },
		notice: func(s string) { fmt.Fprintln(os.Stderr, s) },
	}
}

// tuiOutput writes lines to the transcript (commitLine) and notices via notice.
func (m *chatTUI) tuiOutput() evalOutput {
	return evalOutput{
		line:   m.commitLine,
		notice: m.notice,
	}
}

// ── CLI entry point ──────────────────────────────────────────────────────────

// evalCommand handles the CLI-level "reasonix eval ..." subcommand.
// Every subcommand delegates to a shared function; the CLI wrappers are
// thin adapters that supply a cliOutput and convert errors to exit codes.
func evalCommand(args []string) int {
	if len(args) == 0 {
		evalPrintUsage(cliOutput())
		return 2
	}

	sub := args[0]
	out := cliOutput()

	switch sub {
	case "compare":
		if len(args) < 3 {
			out.notice("usage: reasonix eval compare <session-a> <session-b>")
			return 2
		}
		return evalCompare(args[1], args[2], out)
	case "define":
		if len(args) < 2 {
			out.notice("usage: reasonix eval define <name>")
			return 2
		}
		if err := evalSharedDefine(evalCLIRoot, args[1], out); err != nil {
			return 1
		}
		return 0
	case "check":
		if len(args) < 2 {
			out.notice("usage: reasonix eval check <name>")
			return 2
		}
		if err := evalSharedCheck(evalCLIRoot, args[1], out); err != nil {
			return 1
		}
		return 0
	case "report":
		if len(args) < 2 {
			out.notice("usage: reasonix eval report <name>")
			return 2
		}
		if err := evalSharedReport(evalCLIRoot, args[1], out); err != nil {
			return 1
		}
		return 0
	case "list":
		if err := evalSharedList(evalCLIRoot, out); err != nil {
			_ = err // Non-fatal: treat as informational.
		}
		return 0
	case "clean":
		if err := evalSharedClean(evalCLIRoot, out); err != nil {
			return 1
		}
		return 0
	default:
		out.notice(fmt.Sprintf("unknown eval subcommand: %s", sub))
		evalPrintUsage(out)
		return 2
	}
}

func evalPrintUsage(out evalOutput) {
	out.notice(`Eval Command — manage eval-driven development workflow

Usage:
  reasonix eval compare <a> <b>  compare two saved session files
  reasonix eval define <name>    create a new eval definition
  reasonix eval check <name>     run and check evals
  reasonix eval report <name>    generate full eval report
  reasonix eval list             show all eval definitions
  reasonix eval clean            remove old logs (keeps last 10 runs)`)
}

// ── Shared logic: root directory ─────────────────────────────────────────────

// evalCLIRoot returns the evals directory used by the CLI (relative to cwd).
func evalCLIRoot() string {
	return filepath.Join(".claude", "evals")
}

// evalTUIDir returns the evals directory for the TUI (under workspace root).
func (m *chatTUI) evalTUIDir() string {
	root := m.ctrl.WorkspaceRoot()
	if root == "" {
		root = "."
	}
	return filepath.Join(root, ".claude", "evals")
}

// ── Shared logic: define ─────────────────────────────────────────────────────

// evalSharedDefine creates a new eval definition directory and template.
func evalSharedDefine(rootDir func() string, name string, out evalOutput) error {
	name = strings.TrimSpace(name)
	if name == "" {
		out.notice("eval name cannot be empty")
		return errEvalInvalid
	}

	defDir := filepath.Join(rootDir(), name)
	if err := os.MkdirAll(defDir, 0o755); err != nil {
		out.notice(fmt.Sprintf("eval: cannot create directory: %v", err))
		return err
	}

	defPath := filepath.Join(defDir, "definition.md")
	if _, err := os.Stat(defPath); err == nil {
		out.notice(fmt.Sprintf("eval definition already exists at %s", defPath))
		return errEvalExists
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
		"- pass@1 > 90%% for capability evals",
		"- pass@3 = 100%% for regression evals",
		"",
	}, "\n"), name, now)

	if err := os.WriteFile(defPath, []byte(template), 0o644); err != nil {
		out.notice(fmt.Sprintf("eval: cannot write definition: %v", err))
		return err
	}

	out.line(fmt.Sprintf("eval definition created at %s", defPath))
	out.line("edit the definition file with capability and regression eval criteria")
	return nil
}

// ── Shared logic: check ──────────────────────────────────────────────────────

// evalSharedCheck runs the evals for a named feature, recording results.
func evalSharedCheck(rootDir func() string, name string, out evalOutput) error {
	name = strings.TrimSpace(name)
	if name == "" {
		out.notice("eval name cannot be empty")
		return errEvalInvalid
	}

	defPath := filepath.Join(rootDir(), name, "definition.md")
	data, err := os.ReadFile(defPath)
	if err != nil {
		out.notice(fmt.Sprintf("eval: cannot read definition at %s — create it with 'eval define %s' first", defPath, name))
		return err
	}

	content := string(data)
	capabilities := evalExtractCheckboxes(content, "### Capability Evals", "### Regression Evals")
	regressions := evalExtractCheckboxes(content, "### Regression Evals", "### Success Criteria")

	if len(capabilities) == 0 && len(regressions) == 0 {
		out.notice("eval: no capability or regression criteria found in definition — fill in the criteria first")
		return errEvalNoCriteria
	}

	logPath := filepath.Join(rootDir(), name, "results.log")
	logDir := filepath.Dir(logPath)
	_ = os.MkdirAll(logDir, 0o755)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		out.notice(fmt.Sprintf("eval: cannot open log: %v", err))
		return err
	}
	defer f.Close()

	now := time.Now().Format(time.RFC3339)
	fmt.Fprintf(f, "\n=== CHECK at %s ===\n", now)

	var capPass, capTotal, regPass, regTotal int

	out.line(fmt.Sprintf("EVAL CHECK: %s", name))
	out.line("========================")
	out.line("")

	out.line("Capability Evals:")
	for _, c := range capabilities {
		capTotal++
		result := evalRunVerify(c)
		if result == "PASS" {
			capPass++
		}
		out.line(fmt.Sprintf("  [%s] %s", result, c))
		fmt.Fprintf(f, "%s: %s\n", c, result)
	}

	out.line("")
	out.line("Regression Evals:")
	for _, r := range regressions {
		regTotal++
		result := evalRunVerify(r)
		if result == "PASS" {
			regPass++
		}
		out.line(fmt.Sprintf("  [%s] %s", result, r))
		fmt.Fprintf(f, "%s: %s\n", r, result)
	}

	status := "IN PROGRESS"
	if capTotal > 0 && capPass == capTotal && regTotal > 0 && regPass == regTotal {
		status = "READY"
	} else if capTotal == 0 && regTotal > 0 && regPass == regTotal {
		status = "READY"
	}

	out.line("")
	out.line(fmt.Sprintf("Capability: %d/%d passing", capPass, capTotal))
	out.line(fmt.Sprintf("Regression: %d/%d passing", regPass, regTotal))
	out.line(fmt.Sprintf("Status: %s", status))
	out.line(fmt.Sprintf("eval check results appended to %s", logPath))
	return nil
}

// ── Shared verify ────────────────────────────────────────────────────────────

// evalRunVerify attempts to verify a single criterion by running the relevant
// shell check. Returns PASS, FAIL, or MANUAL (when no automated check applies).
func evalRunVerify(criterion string) string {
	lower := strings.ToLower(criterion)

	switch {
	case strings.Contains(lower, "build") && !strings.Contains(lower, "vet"):
		return evalRunCommandCheck("go build ./... 2>&1",
			func(out string) string { return fmt.Sprintf("FAIL (build output):\n%s", out) })
	case strings.Contains(lower, "vet"):
		return evalRunCommandCheck("go vet ./... 2>&1",
			func(out string) string { return fmt.Sprintf("FAIL (vet output):\n%s", out) })
	case strings.Contains(lower, "go test") || (strings.Contains(lower, "test") && strings.Contains(lower, "./...")):
		return evalRunCommandCheck("go test ./... 2>&1",
			func(out string) string { return fmt.Sprintf("FAIL (test output):\n%s", out) })
	case strings.Contains(lower, "test") && strings.Contains(lower, "pass"):
		return evalRunCommandCheck("go test ./... 2>&1",
			func(out string) string { return fmt.Sprintf("FAIL (test output):\n%s", out) })
	default:
		return "MANUAL"
	}
}

// evalRunCommandCheck runs cmd via sh -c and returns PASS on success or a
// detailed FAIL message (via failMsg) that includes the error output.
func evalRunCommandCheck(cmd string, failMsg func(string) string) string {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed == "" {
			return "FAIL"
		}
		return failMsg(trimmed)
	}
	return "PASS"
}

// ── Shared logic: report ─────────────────────────────────────────────────────

// evalSharedReport generates a comprehensive eval report for the named feature.
func evalSharedReport(rootDir func() string, name string, out evalOutput) error {
	name = strings.TrimSpace(name)
	if name == "" {
		out.notice("eval name cannot be empty")
		return errEvalInvalid
	}

	defPath := filepath.Join(rootDir(), name, "definition.md")
	defData, err := os.ReadFile(defPath)
	if err != nil {
		out.notice(fmt.Sprintf("eval: no definition at %s — create it with 'eval define %s' first", defPath, name))
		return err
	}

	logData := ""
	logPath := filepath.Join(rootDir(), name, "results.log")
	if raw, err := os.ReadFile(logPath); err == nil {
		logData = string(raw)
	}

	content := string(defData)
	capabilities := evalExtractCheckboxes(content, "### Capability Evals", "### Regression Evals")
	regressions := evalExtractCheckboxes(content, "### Regression Evals", "### Success Criteria")

	capPass, capTotal := evalCountResults(logData, capabilities)
	regPass, regTotal := evalCountResults(logData, regressions)

	var b strings.Builder
	fmt.Fprintf(&b, "EVAL REPORT: %s\n", name)
	b.WriteString("=========================\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", time.Now().Format(time.RFC3339))

	b.WriteString("CAPABILITY EVALS\n")
	b.WriteString("----------------\n")
	if len(capabilities) == 0 {
		b.WriteString("(none defined)\n")
	} else {
		for _, c := range capabilities {
			status := evalLatestStatus(logData, c)
			fmt.Fprintf(&b, "  [%s] %s\n", status, c)
		}
	}

	b.WriteString("\nREGRESSION EVALS\n")
	b.WriteString("----------------\n")
	if len(regressions) == 0 {
		b.WriteString("(none defined)\n")
	} else {
		for _, r := range regressions {
			status := evalLatestStatus(logData, r)
			fmt.Fprintf(&b, "  [%s] %s\n", status, r)
		}
	}

	// Compute pass@1 (percentage of checks passing in latest run).
	passAt1 := 0
	if capTotal > 0 {
		passAt1 = capPass * 100 / capTotal
	}

	// pass@3: count how many of the last 3 runs fully passed (all criteria green).
	// This is not a simple percentage — it measures run-level reliability.
	passAt3 := evalComputePassAtN(logData, 3, capabilities)

	regPassAt3 := 0
	if regTotal > 0 {
		regPassAt3 = regPass * 100 / regTotal
	}

	b.WriteString("\nMETRICS\n")
	b.WriteString("-------\n")
	fmt.Fprintf(&b, "Capability pass@1: %d%%\n", passAt1)
	fmt.Fprintf(&b, "Capability pass@3: %d/%d runs all-green\n", passAt3, min(3, evalCountCheckRuns(logData)))
	fmt.Fprintf(&b, "Regression pass@1: %d%%\n", regPassAt3)

	b.WriteString("\nNOTES\n")
	b.WriteString("-----\n")
	if evalLatestStatus(logData, "implicit") == "" {
		b.WriteString("(no manual notes recorded)\n")
	} else {
		b.WriteString(evalExtractNotes(logData))
	}

	b.WriteString("\nRECOMMENDATION\n")
	b.WriteString("--------------\n")
	rec := evalBuildRecommendation(capTotal, regTotal, capPass, regPass)
	b.WriteString(rec + "\n")

	out.line(b.String())
	return nil
}

// evalBuildRecommendation returns a human-readable ship/block recommendation.
func evalBuildRecommendation(capTotal, regTotal, capPass, regPass int) string {
	switch {
	case capTotal > 0 && regTotal > 0 && capPass >= capTotal && regPass >= regTotal:
		return "SHIP"
	case capTotal > 0 && capPass >= capTotal && regTotal == 0:
		return "SHIP (no regressions defined)"
	case capTotal == 0 && regTotal > 0 && regPass >= regTotal:
		return "SHIP (no capabilities defined)"
	default:
		return "BLOCKED"
	}
}

// ── Log helpers ──────────────────────────────────────────────────────────────

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

// evalLatestStatus returns the most recent PASS/FAIL/… status for a criterion.
func evalLatestStatus(logData, criterion string) string {
	lines := strings.Split(logData, "\n")
	status := ""
	prefix := criterion + ":"
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
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

// evalCountCheckRuns returns the number of "=== CHECK at" markers in the log.
func evalCountCheckRuns(logData string) int {
	n := 0
	for _, line := range strings.Split(logData, "\n") {
		if strings.HasPrefix(line, "=== CHECK at ") {
			n++
		}
	}
	return n
}

// evalComputePassAtN returns how many of the last N full check runs had all
// criteria passing. If there are fewer than N runs, uses the available count.
func evalComputePassAtN(logData string, n int, criteria []string) int {
	blocks := strings.Split(logData, "=== CHECK at ")
	if len(blocks) <= 1 {
		return 0
	}

	// Blocks[1:] are the check runs. Take the last N.
	start := len(blocks) - n
	if start < 1 {
		start = 1
	}

	allPassCount := 0
	for i := start; i < len(blocks); i++ {
		block := blocks[i]
		allPass := true
		for _, c := range criteria {
			if evalStatusInBlock(block, c) != "PASS" {
				allPass = false
				break
			}
		}
		if allPass && len(criteria) > 0 {
			allPassCount++
		}
	}
	return allPassCount
}

// evalStatusInBlock returns the status of a criterion within a single check block.
func evalStatusInBlock(block, criterion string) string {
	prefix := criterion + ":"
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "NOT RUN"
}

// evalExtractNotes extracts free-form notes from the latest check run.
func evalExtractNotes(logData string) string {
	parts := strings.Split(logData, "=== CHECK at ")
	if len(parts) <= 1 {
		return ""
	}
	last := parts[len(parts)-1]
	lines := strings.SplitN(last, "\n", 2)
	if len(lines) < 2 {
		return ""
	}
	return strings.TrimSpace(lines[1])
}

// ── Shared logic: list ───────────────────────────────────────────────────────

// evalSharedList shows all eval definitions with their current status.
func evalSharedList(rootDir func() string, out evalOutput) error {
	dir := rootDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		out.notice("no eval definitions found — create one with 'eval define <name>'")
		return nil
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	if len(names) == 0 {
		out.notice("no eval definitions found — create one with 'eval define <name>'")
		return nil
	}

	var b strings.Builder
	b.WriteString("EVAL DEFINITIONS\n")
	b.WriteString("================\n")

	for _, name := range names {
		defPath := filepath.Join(dir, name, "definition.md")
		logPath := filepath.Join(dir, name, "results.log")

		defData, err := os.ReadFile(defPath)
		if err != nil {
			fmt.Fprintf(&b, "  %-20s [ERROR: cannot read definition]\n", name)
			continue
		}
		logData := ""
		if raw, err := os.ReadFile(logPath); err == nil {
			logData = string(raw)
		}

		capabilities := evalExtractCheckboxes(string(defData), "### Capability Evals", "### Regression Evals")
		regressions := evalExtractCheckboxes(string(defData), "### Regression Evals", "### Success Criteria")

		capPass, capTotal := evalCountResults(logData, capabilities)
		regPass, regTotal := evalCountResults(logData, regressions)

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

		fmt.Fprintf(&b, "  %-20s [%s]\n", name, status)
	}

	out.line(b.String())
	return nil
}

// ── Shared logic: clean ──────────────────────────────────────────────────────

// evalSharedClean removes old result logs, keeping the last 10 runs per eval.
func evalSharedClean(rootDir func() string, out evalOutput) error {
	dir := rootDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		out.notice("no eval definitions to clean")
		return nil
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
		out.line("eval: no logs needed trimming (all have ≤10 runs)")
	} else {
		out.line(fmt.Sprintf("eval: trimmed old runs from %d eval(s) (kept last 10 each)", cleaned))
	}
	return nil
}

// ── eval compare (restored) ──────────────────────────────────────────────────

// evalCompare compares two saved session files using the internal/eval package.
func evalCompare(sessionA, sessionB string, out evalOutput) int {
	a, err := eval.LoadSessionSnapshot(sessionA)
	if err != nil {
		out.notice(fmt.Sprintf("error loading %s: %v", sessionA, err))
		return 1
	}
	b, err := eval.LoadSessionSnapshot(sessionB)
	if err != nil {
		out.notice(fmt.Sprintf("error loading %s: %v", sessionB, err))
		return 1
	}

	result := eval.Compare(a, b)
	out.line(result.FormatText())
	return 0
}

// ── Markdown checkbox extraction ─────────────────────────────────────────────

// evalExtractCheckboxes extracts markdown checklist items between two section
// headers. Each item is the text after "- [ ] " or "- [x] ".
func evalExtractCheckboxes(content, startSection, endSection string) []string {
	var items []string

	startIdx := strings.Index(content, startSection)
	if startIdx < 0 {
		return nil
	}

	endIdx := len(content)
	if endSection != "" {
		if idx := strings.Index(content[startIdx+len(startSection):], endSection); idx >= 0 {
			endIdx = startIdx + len(startSection) + idx
		}
	}

	section := content[startIdx:endIdx]
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [ ] ") {
			items = append(items, strings.TrimPrefix(trimmed, "- [ ] "))
		} else if strings.HasPrefix(trimmed, "- [x] ") {
			items = append(items, strings.TrimPrefix(trimmed, "- [x] "))
		}
	}

	return items
}

// ── TUI integration ──────────────────────────────────────────────────────────

// runEvalSubcommand dispatches /eval subcommands in the TUI.
func (m *chatTUI) runEvalSubcommand(input string) {
	m.echoLocalCommand(input)

	parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(input, "/eval")))
	if len(parts) == 0 {
		m.evalShowHelp()
		return
	}

	sub := parts[0]
	out := m.tuiOutput()

	switch sub {
	case "define":
		if len(parts) < 2 {
			out.notice("usage: /eval define <name>")
			return
		}
		_ = evalSharedDefine(m.evalTUIDir, parts[1], out)
	case "check":
		if len(parts) < 2 {
			out.notice("usage: /eval check <name>")
			return
		}
		_ = evalSharedCheck(m.evalTUIDir, parts[1], out)
	case "report":
		if len(parts) < 2 {
			out.notice("usage: /eval report <name>")
			return
		}
		_ = evalSharedReport(m.evalTUIDir, parts[1], out)
	case "list":
		_ = evalSharedList(m.evalTUIDir, out)
	case "clean":
		_ = evalSharedClean(m.evalTUIDir, out)
	default:
		out.notice(fmt.Sprintf("unknown eval subcommand: %s — try one of: define, check, report, list, clean", sub))
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

// ── Sentinel errors ──────────────────────────────────────────────────────────

var (
	errEvalInvalid    = fmt.Errorf("invalid eval name")
	errEvalExists     = fmt.Errorf("eval definition already exists")
	errEvalNoCriteria = fmt.Errorf("no criteria in eval definition")
)
