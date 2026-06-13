package main

import (
	"reasonix/internal/control"
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
