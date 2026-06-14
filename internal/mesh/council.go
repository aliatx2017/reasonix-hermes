package mesh

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Council orchestrates a multi-agent decision: broadcast a task to N peers,
// collect their proposals, and synthesise a consensus answer. This is the
// "council mode" where each peer independently reasons about the same problem.
type Council struct {
	mesh      *Mesh
	proposals []DelegationResult
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
