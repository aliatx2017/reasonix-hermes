package main

import (
	"context"
	"os"
	"path/filepath"
	"reasonix/internal/control"
	"reasonix/internal/diff"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// --- Cache Economy Gauge ---

// CacheEconomyView carries session-wide cache performance metrics.
type CacheEconomyView struct {
	HitTokens   int     `json:"hitTokens"`
	MissTokens  int     `json:"missTokens"`
	TotalTokens int     `json:"totalTokens"`
	HitRate     float64 `json:"hitRate"`
}

// CacheEconomy returns cache hit/miss stats for the active tab.
func (a *App) CacheEconomy() CacheEconomyView {
	return a.CacheEconomyForTab("")
}

// --- Cost Summary ---

// CostSummaryView carries session-level cost information.
type CostSummaryView struct {
	SessionCost float64 `json:"sessionCost"`
	Currency    string  `json:"currency"`
}

// CostSummary returns cumulative cost for the active tab.
func (a *App) CostSummary() CostSummaryView {
	return a.CostSummaryForTab("")
}

// CostSummaryForTab returns cumulative cost for a specific tab.
func (a *App) CostSummaryForTab(tabID string) CostSummaryView {
	ctrl := a.ctrlForTab(tabID)
	if ctrl == nil {
		return CostSummaryView{}
	}
	cost := ctrl.SessionCost()
	curr := "¥"
	// Try to get currency from the active provider config.
	if p := ctrl.ActivePricing(); p != nil {
		curr = p.Symbol()
	}
	return CostSummaryView{
		SessionCost: cost,
		Currency:    curr,
	}
}

// CacheEconomyForTab returns cache stats for a specific tab.
func (a *App) CacheEconomyForTab(tabID string) CacheEconomyView {
	ctrl := a.ctrlForTab(tabID)
	if ctrl == nil {
		return CacheEconomyView{}
	}
	hit, miss := ctrl.SessionCache()
	total := hit + miss
	var rate float64
	if total > 0 {
		rate = float64(hit) * 100.0 / float64(total)
	}
	return CacheEconomyView{
		HitTokens:   hit,
		MissTokens:  miss,
		TotalTokens: total,
		HitRate:     rate,
	}
}

// --- Memory Dashboard ---

// MemoryDashboardView carries aggregate memory statistics.
type MemoryDashboardView struct {
	TotalFacts  int `json:"totalFacts"`
	TotalDocs   int `json:"totalDocs"`
	TotalScopes int `json:"totalScopes"`
}

// MemoryDashboard returns aggregate memory stats.
func (a *App) MemoryDashboard() MemoryDashboardView {
	mv := a.Memory()
	return MemoryDashboardView{
		TotalFacts:  len(mv.Facts),
		TotalDocs:   len(mv.Docs),
		TotalScopes: len(mv.Scopes),
	}
}

// --- Discord Bot Live Monitor ---

// BotLiveStatusView carries live IM bot session stats.
type BotLiveStatusView struct {
	Running        bool   `json:"running"`
	Platform       string `json:"platform"`
	ActiveSessions int    `json:"activeSessions"`
	Status         string `json:"status"`
}

// BotLiveStatus returns live bot runtime status.
func (a *App) BotLiveStatus() BotLiveStatusView {
	s := a.BotRuntimeStatus()
	return BotLiveStatusView{
		Running:        s.Running,
		Platform:       "discord",
		ActiveSessions: s.Connections,
		Status:         s.Status,
	}
}

// --- Goal Progress Widget ---

// GoalProgressView carries active goal tracking data.
type GoalProgressView struct {
	Active bool   `json:"active"`
	Goal   string `json:"goal"`
	Status string `json:"status"`
	Turns  int    `json:"turns"`
	Blocks int    `json:"blocks"`
}

// GoalProgress returns goal progress for the active tab.
func (a *App) GoalProgress() GoalProgressView {
	return a.GoalProgressForTab("")
}

// GoalProgressForTab returns goal progress for a specific tab.
func (a *App) GoalProgressForTab(tabID string) GoalProgressView {
	ctrl := a.ctrlForTab(tabID)
	if ctrl == nil {
		return GoalProgressView{}
	}
	goal := ctrl.Goal()
	if goal == "" {
		return GoalProgressView{Active: false}
	}
	return GoalProgressView{
		Active: true,
		Goal:   goal,
		Status: ctrl.GoalStatus(),
		Turns:  ctrl.GoalTurns(),
		Blocks: ctrl.GoalBlocks(),
	}
}

// --- helpers ---

func (a *App) ctrlForTab(tabID string) *control.Controller {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if tabID == "" {
		return a.activeCtrlLocked()
	}
	tab := a.tabByIDLocked(tabID)
	if tab != nil {
		return tab.Ctrl
	}
	return nil
}

// --- Live Dashboard Event Loop ---

// HermesDashboardEvent is the push payload sent to the frontend every few seconds.
type HermesDashboardEvent struct {
	Cache        CacheEconomyView        `json:"cache"`
	Cost         CostSummaryView         `json:"cost"`
	Memory       MemoryDashboardView     `json:"memory"`
	Bot          BotLiveStatusView       `json:"bot"`
	Goal         GoalProgressView        `json:"goal"`
	Subagents    []SubagentNodeView      `json:"subagents"`
	Constitution ConstitutionHealthView  `json:"constitution"`
	TurnUsage    []TurnUsagePoint        `json:"turnUsage"`
	Compactions  []CompactionEvent       `json:"compactions"`
	MemoryFacts  []MemoryFactView        `json:"memoryFacts"`
}

// MemoryFactView is one fact from the auto-memory store.
type MemoryFactView struct {
	Title       string `json:"title"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// MemoryFacts returns facts and docs from the auto-memory store for graph display.
func (a *App) MemoryFacts() []MemoryFactView {
	mv := a.Memory()
	var facts []MemoryFactView
	for _, f := range mv.Facts {
		facts = append(facts, MemoryFactView{
			Title:       f.Title,
			Type:        f.Type,
			Description: f.Description,
		})
	}
	for _, d := range mv.Docs {
		facts = append(facts, MemoryFactView{
			Title:       filepath.Base(d.Path),
			Type:        "doc:" + d.Scope,
			Description: d.Path,
		})
	}
	return facts
}

// CompactionEvent is one compaction pass for the timeline.
type CompactionEvent struct {
	Trigger  string `json:"trigger"`
	Messages int    `json:"messages"`
	Summary  string `json:"summary"`
}

// CheckpointFileSnap is one file's pre-edit state at a checkpoint.
type CheckpointFileSnap struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// CheckpointFileDiff is the diff between a checkpoint file snapshot and the current file.
type CheckpointFileDiff struct {
	Path    string `json:"path"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
	Diff    string `json:"diff"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Same    bool   `json:"same"`
}

// CheckpointFileDiff returns a unified diff between the checkpoint snapshot for a
// file and its current content on disk. If the file hasn't changed, Same is true
// and Diff is empty.
func (a *App) CheckpointFileDiff(turn int, relPath string) *CheckpointFileDiff {
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return nil
	}
	snaps := ctrl.CheckpointFileSnaps(turn)
	var oldContent *string
	for _, s := range snaps {
		if s.Path == relPath {
			oldContent = s.Content
			break
		}
	}
	if oldContent == nil {
		return nil // file not found in checkpoint
	}

	// Read current file content.
	root := ctrl.WorkspaceRoot()
	if root == "" {
		return nil
	}
	fullPath := filepath.Join(root, relPath)
	current, err := os.ReadFile(fullPath)
	if err != nil {
		// File may have been deleted.
		d := diff.Build(relPath, *oldContent, "", diff.Delete)
		return &CheckpointFileDiff{
			Path:    relPath,
			OldText: *oldContent,
			NewText: "",
			Diff:    d.Diff,
			Added:   d.Added,
			Removed: d.Removed,
			Same:    false,
		}
	}
	currentStr := string(current)
	if *oldContent == currentStr {
		return &CheckpointFileDiff{
			Path:    relPath,
			OldText: *oldContent,
			NewText: currentStr,
			Diff:    "",
			Same:    true,
		}
	}

	d := diff.Build(relPath, *oldContent, currentStr, diff.Modify)
	return &CheckpointFileDiff{
		Path:    relPath,
		OldText: *oldContent,
		NewText: currentStr,
		Diff:    d.Diff,
		Added:   d.Added,
		Removed: d.Removed,
		Same:    false,
	}
}

// CheckpointFileList returns file snapshots for a specific checkpoint turn.
func (a *App) CheckpointFileList(turn int) []CheckpointFileSnap {
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return []CheckpointFileSnap{}
	}
	snaps := ctrl.CheckpointFileSnaps(turn)
	out := make([]CheckpointFileSnap, len(snaps))
	for i, s := range snaps {
		content := ""
		if s.Content != nil {
			content = *s.Content
		}
		out[i] = CheckpointFileSnap{Path: s.Path, Content: content}
	}
	return out
}
func (a *App) CompactionHistory() []CompactionEvent {
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return []CompactionEvent{}
	}
	history := ctrl.CompactionHistory()
	events := make([]CompactionEvent, len(history))
	for i, c := range history {
		events[i] = CompactionEvent{
			Trigger:  c.Trigger,
			Messages: c.Messages,
			Summary:  c.Summary,
		}
	}
	return events
}

// TurnUsagePoint is one entry in the per-turn token breakdown chart.
type TurnUsagePoint struct {
	Turn             int `json:"turn"`
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	CacheHitTokens   int `json:"cacheHitTokens"`
	CacheMissTokens  int `json:"cacheMissTokens"`
}

// TurnUsageHistory returns per-turn token usage for sparkline charts.
func (a *App) TurnUsageHistory() []TurnUsagePoint {
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return []TurnUsagePoint{}
	}
	history := ctrl.TurnUsageHistory()
	points := make([]TurnUsagePoint, len(history))
	for i, u := range history {
		points[i] = TurnUsagePoint{
			Turn:             i + 1,
			PromptTokens:     u.PromptTokens,
			CompletionTokens: u.CompletionTokens,
			CacheHitTokens:   u.CacheHitTokens,
			CacheMissTokens:  u.CacheMissTokens,
		}
	}
	return points
}

// startHermesEventLoop pushes dashboard data to the frontend on a timer
// so components can subscribe via EventsOn instead of polling individually.
func (a *App) startHermesEventLoop(ctx context.Context) {
	const interval = 3 * time.Second
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ev := HermesDashboardEvent{
					Cache:         a.CacheEconomy(),
					Cost:          a.CostSummary(),
					Memory:        a.MemoryDashboard(),
					Bot:           a.BotLiveStatus(),
					Goal:          a.GoalProgress(),
					Subagents:     a.SubagentTree(),
					Constitution:  a.ConstitutionHealth(),
					TurnUsage:     a.TurnUsageHistory(),
					Compactions:   a.CompactionHistory(),
					MemoryFacts:   a.MemoryFacts(),
				}
				runtime.EventsEmit(ctx, "hermes:dashboard", ev)
			}
		}
	}()
}
