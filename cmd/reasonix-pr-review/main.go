// reasonix-pr-review — GitHub Action CLI: fetches a PR diff, runs a Reasonix
// review, and prints a markdown review suitable for posting as a PR comment.
//
// Usage:
//
//	reasonix-pr-review [flags]
//
// Flags:
//
//	-repo       GitHub repository (owner/name) — defaults to GITHUB_REPOSITORY env
//	-pr         PR number — defaults to PR_NUMBER env or derived from GITHUB_REF
//	-token      GitHub token — defaults to GITHUB_TOKEN env
//	-model      Model for review (default "deepseek-pro")
//	-reasonix   Path to reasonix binary (default "reasonix")
//
// Environment variables (set by GitHub Actions):
//
//	GITHUB_REPOSITORY  owner/repo
//	GITHUB_TOKEN       actions token
//	PR_NUMBER          set from github.event.pull_request.number
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"reasonix/internal/netclient"
)

func main() {
	repo := flag.String("repo", "", "GitHub repository (owner/name)")
	prNum := flag.Int("pr", 0, "PR number")
	token := flag.String("token", "", "GitHub token")
	model := flag.String("model", "deepseek-pro", "Model for review")
	reasonixBin := flag.String("reasonix", "reasonix", "Path to reasonix binary")
	flag.Parse()

	// Resolve from env
	if *repo == "" {
		*repo = os.Getenv("GITHUB_REPOSITORY")
	}
	if *token == "" {
		*token = os.Getenv("GITHUB_TOKEN")
	}
	if *prNum == 0 {
		if s := os.Getenv("PR_NUMBER"); s != "" {
			n, _ := strconv.Atoi(s)
			*prNum = n
		}
	}

	if *repo == "" || *token == "" || *prNum == 0 {
		fmt.Fprintln(os.Stderr, "error: -repo, -pr, and -token are required")
		os.Exit(1)
	}

	ctx := context.Background()

	// 1. Fetch PR metadata (title, body)
	title, body, err := fetchPR(ctx, *repo, *prNum, *token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching PR: %v\n", err)
		os.Exit(1)
	}

	// 2. Fetch PR diff
	diff, err := fetchDiff(ctx, *repo, *prNum, *token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching diff: %v\n", err)
		os.Exit(1)
	}

	if len(diff) == 0 {
		fmt.Println("### Reasonix PR Review\n\nNo diff to review (empty PR).")
		return
	}

	// 3. Build review prompt
	prompt := buildReviewPrompt(title, body, diff)

	// 4. Run reasonix review
	review, err := runReasonix(ctx, *reasonixBin, *model, prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running reasonix: %v\n", err)
		os.Exit(1)
	}

	// 5. Print review
	fmt.Println(review)
}

func fetchPR(ctx context.Context, repo string, pr int, token string) (title, body string, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", repo, pr)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := netclient.DefaultClient().Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var prData struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&prData); err != nil {
		return "", "", fmt.Errorf("parse PR: %w", err)
	}
	return prData.Title, prData.Body, nil
}

func fetchDiff(ctx context.Context, repo string, pr int, token string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", repo, pr)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	resp, err := netclient.DefaultClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Cap at 2MB — covers most PRs
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, resp.Body, 2<<20); err != nil && err != io.EOF {
		return "", fmt.Errorf("read diff: %w", err)
	}
	return buf.String(), nil
}

func buildReviewPrompt(title, body, diff string) string {
	var b strings.Builder
	b.WriteString("Review this pull request. Research shows 46% of agent-generated PRs are rejected for four reasons:\n")
	b.WriteString("incorrect implementation, CI failures, missing constraints, and inability to implement.\n")
	b.WriteString("Apply these findings — catch what human reviewers reject.\n\n")

	b.WriteString("### Required Review Dimensions\n\n")
	b.WriteString("1. **Correctness** — does the implementation actually fix the stated problem?\n")
	b.WriteString("   Check: off-by-one, nil dereference, race condition, inverted condition, wrong target.\n")
	b.WriteString("2. **CI & validation** — will tests/lint/build pass? Flag anything that would break CI:\n")
	b.WriteString("   missing imports, syntax errors, type mismatches, broken references.\n")
	b.WriteString("3. **Missing constraints** — what should NOT be changed? Flag changes that:\n")
	b.WriteString("   touch unrelated code, break existing APIs, alter config defaults, remove guards.\n")
	b.WriteString("4. **Security** — injection, auth bypass, secret exposure, path traversal, unsafe deserialization.\n")
	b.WriteString("5. **Verification & trustworthiness** — agent output can appear correct while being deceptive.\n")
	b.WriteString("   Research shows deceptive agents share a linguistic signature: selective helpfulness,\n")
	b.WriteString("   hedging without evidence, and evasive brevity on hard questions. Flag output that:\n")
	b.WriteString("   dodges the hard part of the change, claims 'should work' without evidence,\n")
	b.WriteString("   explains what was done but not WHY, or uses vague reassurances instead of specifics.\n")
	b.WriteString("6. **Completeness** — is the fix self-contained or does it need follow-up? Flag missing:\n")
	b.WriteString("   error handling, tests, docs updates, config examples, migration notes.\n\n")

	b.WriteString("### Output Format\n\n")
	b.WriteString("- One section per finding, with file:line references and severity (🔴 high / 🟡 medium / 🟢 low).\n")
	b.WriteString("- Final summary: one-paragraph verdict with merge recommendation.\n")
	b.WriteString("- Be concise. Skip praise — focus on what would cause rejection.\n\n")

	fmt.Fprintf(&b, "## PR: %s\n\n", title)
	if body != "" {
		fmt.Fprintf(&b, "### Description\n%s\n\n", body)
	}
	fmt.Fprintf(&b, "### Diff\n```diff\n%s\n```\n", truncate(diff, 50000))

	return b.String()
}

func runReasonix(ctx context.Context, bin, model, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, "run",
		"--model", model,
		"--max-steps", "5",
		"--approval", "auto",
		prompt,
	)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("reasonix: %w", err)
	}
	return string(out), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n... (truncated)"
}
