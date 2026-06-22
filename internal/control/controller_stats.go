// Package control is the transport-agnostic session driver.
package control

import (
	"encoding/json"
	"net/http"
	"time"

	"reasonix/internal/compress"
	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/scheduler"
)

// HeadroomProxyStats holds a snapshot of headroom proxy metrics.
type HeadroomProxyStats struct {
	Running       bool    `json:"running"`
	Requests      int     `json:"requests"`
	TokensBefore  int     `json:"tokens_before"`
	TokensSaved   int     `json:"tokens_saved"`
	SavingsPct    float64 `json:"savings_pct"`
	CostSavedUSD  float64 `json:"cost_saved_usd"`
	Version       string  `json:"version"`
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

// HeadroomProxyStats fetches a snapshot of headroom proxy metrics from the
// local proxy. Returns zero stats (Running=false) if the proxy is unreachable.
func (c *Controller) HeadroomProxyStats() HeadroomProxyStats {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://localhost:8787/stats")
	if err != nil {
		return HeadroomProxyStats{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return HeadroomProxyStats{}
	}
	var raw struct {
		Summary struct {
			APIRequests int `json:"api_requests"`
			Compression struct {
				TotalTokensBefore float64 `json:"total_tokens_before_with_cli_filtering"`
				TotalTokensSaved  float64 `json:"total_tokens_saved_with_cli_filtering"`
			} `json:"compression"`
			Cost struct {
				TotalSavedUSD float64 `json:"total_saved_usd"`
				SavingsPct    float64 `json:"savings_pct"`
			} `json:"cost"`
		} `json:"summary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return HeadroomProxyStats{}
	}
	tb := int(raw.Summary.Compression.TotalTokensBefore)
	ts := int(raw.Summary.Compression.TotalTokensSaved)
	pct := raw.Summary.Cost.SavingsPct
	if pct == 0 && tb > 0 {
		pct = float64(ts) / float64(tb) * 100
	}
	return HeadroomProxyStats{
		Running:      true,
		Requests:     raw.Summary.APIRequests,
		TokensBefore: tb,
		TokensSaved:  ts,
		SavingsPct:   pct,
		CostSavedUSD: raw.Summary.Cost.TotalSavedUSD,
	}
}



