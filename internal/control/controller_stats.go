// Package control is the transport-agnostic session driver.
package control

import (
	"context"
	"strings"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/billing"
	"reasonix/internal/compress"
	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/scheduler"
)

func (c *Controller) SessionDir() string { return c.sessionDir }

// SessionPath reports the file the current conversation auto-saves to ("" when
// persistence is disabled), so a history view can mark the active session.
func (c *Controller) SessionPath() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionPath
}

func (c *Controller) parentSessionID() string {
	return agent.BranchID(c.SessionPath())
}

// History returns the executor's current message log (for repopulating a
// resumed frontend's view).
func (c *Controller) History() []provider.Message {
	if c.executor == nil {
		return nil
	}
	return c.executor.Session().Snapshot() // copy — a turn may be appending concurrently
}

// ContextSnapshot returns (promptTokens, contextWindow) from the most recent
// turn. Both zero means no data yet — a gauge hides itself.
func (c *Controller) ContextSnapshot() (int, int) {
	if c.executor == nil {
		return 0, 0
	}
	u := c.executor.LastUsage()
	if u == nil {
		return 0, c.executor.ContextWindow()
	}
	return u.PromptTokens, c.executor.ContextWindow()
}

// CompactRatio returns the auto-compaction threshold as a fraction of the window
// (0 when the executor is unset). The status line shows headroom against it.
func (c *Controller) CompactRatio() float64 {
	if c.executor == nil {
		return 0
	}
	return c.executor.CompactRatio()
}

// LastUsage returns the most recent turn's token telemetry (nil before the first
// turn), so frontends can derive the prompt cache-hit rate for the status line.
func (c *Controller) LastUsage() *provider.Usage {
	if c.executor == nil {
		return nil
	}
	return c.executor.LastUsage()
}

// SessionCache returns cumulative cache hit/miss prompt tokens for the session,
// so a frontend can render the aggregate (session-wide) cache-hit rate — steadier
// than the single-turn rate and unaffected by compaction.
func (c *Controller) SessionCache() (hit, miss int) {
	if c.executor == nil {
		return 0, 0
	}
	return c.executor.SessionCache()
}

// SessionCost returns the estimated total spend this session.
func (c *Controller) SessionCost() float64 {
	if c.executor == nil {
		return 0
	}
	return c.executor.SessionCost()
}

// SessionTokensIn returns the cumulative prompt tokens across every API call.
func (c *Controller) SessionTokensIn() int {
	if c.executor == nil {
		return 0
	}
	return c.executor.SessionTokensIn()
}

// SessionTokensOut returns the cumulative completion tokens across every API call.
func (c *Controller) SessionTokensOut() int {
	if c.executor == nil {
		return 0
	}
	return c.executor.SessionTokensOut()
}

// SessionTurns returns the number of completed model turns this session.
func (c *Controller) SessionTurns() int {
	if c.executor == nil {
		return 0
	}
	return c.executor.SessionTurns()
}

// AuxTokens returns tokens routed to auxiliary providers (compression/vision).
func (c *Controller) AuxTokens() int {
	if c.executor == nil {
		return 0
	}
	return c.executor.AuxTokens()
}

// CompressStats returns compression statistics (cache hits, lines collapsed, bytes saved).
func (c *Controller) CompressStats() compress.Stats {
	if c.executor == nil {
		return compress.Stats{}
	}
	return c.executor.CompressStats()
}

// ActivePricing returns the pricing data for the active model, or nil.
func (c *Controller) ActivePricing() *provider.Pricing {
	if c.executor == nil {
		return nil
	}
	return c.executor.Pricing()
}

// ScheduleStatus returns the next scheduled task run times.
func (c *Controller) ScheduleStatus() map[string]time.Time {
	if c.schedule == nil {
		return nil
	}
	return c.schedule.AllNextRuns()
}

// ScheduleResults returns the last N scheduled task results.
func (c *Controller) ScheduleResults(limit int) []scheduler.Result {
	if c.schedule == nil {
		return nil
	}
	return c.schedule.Results(limit)
}

// Schedule returns the scheduler, or nil when not configured.
func (c *Controller) Schedule() *scheduler.Scheduler {
	return c.schedule
}

// AddScheduledTask adds or updates a scheduled task at runtime.
func (c *Controller) AddScheduledTask(t scheduler.Task) bool {
	if c.schedule == nil {
		return false
	}
	return c.schedule.AddTask(t)
}

// RemoveScheduledTask removes a scheduled task by name at runtime.
func (c *Controller) RemoveScheduledTask(name string) bool {
	if c.schedule == nil {
		return false
	}
	return c.schedule.RemoveTask(name)
}

// SessionMessages returns a snapshot of the session message history for export.
func (c *Controller) SessionMessages() []provider.Message {
	if c.executor == nil {
		return []provider.Message{}
	}
	sess := c.executor.Session()
	if sess == nil {
		return []provider.Message{}
	}
	return sess.Snapshot()
}

// TurnUsageHistory returns a snapshot of the last N per-turn Usage samples for
// rendering token breakdown charts in the frontend.
func (c *Controller) TurnUsageHistory() []provider.Usage {
	if c.executor == nil {
		return nil
	}
	return c.executor.TurnUsageHistory()
}

// CompactionHistory returns compaction events for the timeline panel.
func (c *Controller) CompactionHistory() []event.Compaction {
	if c.executor == nil {
		return nil
	}
	return c.executor.CompactionHistory()
}

// ToolResultData holds the full arguments and output for one tool call, loaded
// on demand when a frontend expands a collapsed tool card.
type ToolResultData struct {
	Args   string `json:"args"`
	Output string `json:"output"`
}

// ToolResult looks up a tool call by its ID in the session history and returns
// the full arguments + output that were elided from the frontend's items[].
// Returns nil when the tool ID isn't found (e.g. a sub-agent's tool call that
// lives in a different session).
func (c *Controller) ToolResult(toolID string) *ToolResultData {
	if c.executor == nil {
		return nil
	}
	msgs := c.executor.Session().Snapshot()
	// Search backwards: tool result first (most recent), then find the args
	// from the preceding assistant turn.
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != provider.RoleTool || msgs[i].ToolCallID != toolID {
			continue
		}
		out := &ToolResultData{
			Args:   "",
			Output: msgs[i].Content,
		}
		// Walk back to find the assistant turn that issued this call.
		for j := i; j >= 0; j-- {
			if msgs[j].Role != provider.RoleAssistant {
				continue
			}
			for _, tc := range msgs[j].ToolCalls {
				if tc.ID == toolID {
					out.Args = tc.Arguments
					return out
				}
			}
		}
		return out
	}
	return nil
}

// Balance queries the active provider's wallet balance, or (nil, nil) when the
// provider declares no balance_url — so a caller treats "not configured" and
// "fetched" the same and just omits the readout when nil.
func (c *Controller) Balance(ctx context.Context) (*billing.Balance, error) {
	if strings.TrimSpace(c.balanceURL) == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	return billing.FetchWithClient(ctx, c.balanceClient, c.balanceURL, c.balanceKey)
}
