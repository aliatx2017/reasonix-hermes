package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// JudgeFunc runs a single agent turn for the council judge. It has the same
// signature as orchestrate.TurnFunc — any frontend can supply it.
type JudgeFunc func(ctx context.Context, prompt string) (string, error)

// CouncilJudgment is the structured analysis a judge model produces after
// comparing peer proposals. Modeled on OpenRouter Fusion Router's judge output:
// consensus (what all/most agreed on), contradictions (disagreements),
// coverage_gaps (what only some covered), unique_insights (unique per-model
// contributions), and blind_spots (what none addressed).
type CouncilJudgment struct {
	Consensus      string   `json:"consensus"`
	Contradictions []string `json:"contradictions"`
	CoverageGaps   []string `json:"coverage_gaps"`
	UniqueInsights []string `json:"unique_insights"`
	BlindSpots     []string `json:"blind_spots"`

	// Raw is the unparsed judge output, preserved when JSON parsing fails
	// so callers can still use the text.
	Raw string `json:"raw,omitempty"`
}

// Council orchestrates a multi-agent decision: broadcast a task to N peers,
// collect their proposals, and synthesise a consensus answer. This is the
// "council mode" where each peer independently reasons about the same problem.
type Council struct {
	mesh      *Mesh
	proposals []DelegationResult
	judgment  *CouncilJudgment // cached result from Judge()
}

// NewCouncil creates a council backed by the given mesh.
func NewCouncil(m *Mesh) *Council {
	return &Council{mesh: m}
}

// Convene broadcasts a task to all peers and collects responses. The council
// can then be queried for the raw proposals, a majority-vote summary, or a
// merged synthesis.
func (c *Council) Convene(ctx context.Context, task string) error {
	results, err := c.mesh.Broadcast(ctx, task)
	if err != nil {
		return fmt.Errorf("council broadcast: %w", err)
	}
	c.proposals = results
	return nil
}

// Proposals returns the raw delegation results from each peer.
func (c *Council) Proposals() []DelegationResult {
	return c.proposals
}

// Consensus returns a synthesis of the peer proposals: a per-peer summary,
// succeeded/failed count, and a combined answer. This is suitable for feeding
// back to the agent as context for the next turn.
func (c *Council) Consensus() string {
	if len(c.proposals) == 0 {
		return "No peers responded."
	}

	var succeeded, failed int
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Council Results (%d peers)\n\n", len(c.proposals)))

	// Sort by name for stable output
	sorted := make([]DelegationResult, len(c.proposals))
	copy(sorted, c.proposals)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Peer < sorted[j].Peer })

	for _, r := range sorted {
		if r.Success {
			succeeded++
			b.WriteString(fmt.Sprintf("### %s ✅ (%v)\n%s\n\n", r.Peer, r.Duration.Round(time.Millisecond), r.Response))
		} else {
			failed++
			b.WriteString(fmt.Sprintf("### %s ❌ (%v)\nError: %s\n\n", r.Peer, r.Duration.Round(time.Millisecond), r.Error))
		}
	}

	b.WriteString(fmt.Sprintf("---\n**Summary**: %d succeeded, %d failed out of %d peers.\n", succeeded, failed, len(sorted)))

	return b.String()
}

// Merge combines all successful proposals into a single synthesis prompt
// suitable for the local agent to reflect on.
func (c *Council) Merge() string {
	if len(c.proposals) == 0 {
		return "No proposals to merge."
	}

	var succeeded []DelegationResult
	for _, r := range c.proposals {
		if r.Success && r.Response != "" {
			succeeded = append(succeeded, r)
		}
	}

	if len(succeeded) == 0 {
		return "All peers failed to respond."
	}

	var b strings.Builder
	b.WriteString("Synthesise the following independent analyses into a single answer. ")
	b.WriteString("Note agreements, disagreements, and any unique insights from individual peers.\n\n")

	for i, r := range succeeded {
		b.WriteString(fmt.Sprintf("## Peer %d: %s\n%s\n\n", i+1, r.Peer, r.Response))
	}

	if len(succeeded) > 1 {
		b.WriteString("## Synthesis Instruction\n")
		b.WriteString("Provide a unified answer that reconciles the above perspectives. ")
		b.WriteString("Where peers disagree, note the disagreement and give the most defensible conclusion.\n")
	}

	return b.String()
}

// judgePrompt builds the structured-analysis prompt for the judge model,
// modeled on OpenRouter Fusion Router's judge output schema.
func (c *Council) judgePrompt() string {
	succeeded := c.succeeded()
	if len(succeeded) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("You are a judge comparing independent analyses of the same task from multiple models.\n")
	b.WriteString("Produce a structured JSON object with these fields:\n\n")
	b.WriteString("- consensus: What all or most models agreed on.\n")
	b.WriteString("- contradictions: Where models disagreed (array of strings).\n")
	b.WriteString("- coverage_gaps: Topics only some models covered (array of strings).\n")
	b.WriteString("- unique_insights: Unique contributions from individual models (array of strings).\n")
	b.WriteString("- blind_spots: Important aspects none of the models addressed (array of strings).\n\n")
	b.WriteString("Return ONLY valid JSON — no markdown fences, no preamble.\n\n")
	b.WriteString("Here are the peer responses:\n\n")

	for i, r := range succeeded {
		b.WriteString(fmt.Sprintf("--- Model %d: %s ---\n%s\n\n", i+1, r.Peer, r.Response))
	}
	return b.String()
}

// succeeded returns the successful proposals with non-empty responses.
func (c *Council) succeeded() []DelegationResult {
	var out []DelegationResult
	for _, r := range c.proposals {
		if r.Success && r.Response != "" {
			out = append(out, r)
		}
	}
	return out
}

// Judge sends the peer proposals to a judge function for structured analysis.
// The judge prompt requests JSON with consensus, contradictions, coverage_gaps,
// unique_insights, and blind_spots. On success the result is cached and
// available via Judgment(). If JSON parsing fails, the raw text is preserved
// in CouncilJudgment.Raw so callers can still use it.
func (c *Council) Judge(ctx context.Context, judge JudgeFunc) error {
	if len(c.succeeded()) == 0 {
		return fmt.Errorf("council judge: no successful proposals to evaluate")
	}
	if judge == nil {
		return fmt.Errorf("council judge: judge function is nil")
	}

	prompt := c.judgePrompt()
	if prompt == "" {
		return fmt.Errorf("council judge: empty prompt")
	}

	raw, err := judge(ctx, prompt)
	if err != nil {
		return fmt.Errorf("council judge: %w", err)
	}

	judgment := &CouncilJudgment{Raw: raw}
	if err := json.Unmarshal([]byte(extractJSON(raw)), judgment); err != nil {
		// Parsing failed — keep Raw so callers can fall back.
		c.judgment = judgment
		return nil
	}
	judgment.Raw = "" // clean: successfully parsed
	c.judgment = judgment
	return nil
}

// Judgment returns the cached judge analysis, or nil if Judge has not been
// called or failed to produce a result.
func (c *Council) Judgment() *CouncilJudgment {
	return c.judgment
}

// extractJSON tries to find a JSON object in text that may be wrapped in
// markdown fences or prefixed with explanatory text.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip ```json / ``` fences.
	if strings.HasPrefix(s, "```") {
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[idx+1:]
		}
		if last := strings.LastIndex(s, "```"); last >= 0 {
			s = s[:last]
		}
	}
	// Find the first { and last }.
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
