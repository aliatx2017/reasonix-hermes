// Package control is the transport-agnostic session driver.
package control

import (
	"time"

	"reasonix/internal/compress"
	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/scheduler"
)







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

// ScheduleNextRuns returns the next scheduled task run times.
func (c *Controller) ScheduleNextRuns() map[string]time.Time {
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

// ScheduleTasks returns the configured scheduled tasks.
func (c *Controller) ScheduleTasks() []scheduler.Task {
	if c.schedule == nil {
		return nil
	}
	return c.schedule.Tasks()
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



