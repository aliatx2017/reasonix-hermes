package main

import (
	"fmt"

	"reasonix/internal/eval"
)

// SessionComparisonView is the Wails-friendly view for session comparison results.
type SessionComparisonView struct {
	SessionA   string            `json:"sessionA"`
	SessionB   string            `json:"sessionB"`
	TokensInA  int               `json:"tokensInA"`
	TokensInB  int               `json:"tokensInB"`
	TokensOutA int               `json:"tokensOutA"`
	TokensOutB int               `json:"tokensOutB"`
	TurnsA     int               `json:"turnsA"`
	TurnsB     int               `json:"turnsB"`
	CostA      float64           `json:"costA"`
	CostB      float64           `json:"costB"`
	Currency   string            `json:"currency"`
	Similarity float64           `json:"similarity"`
	TextReport string            `json:"textReport"`
	ToolDiffs  []ToolDiffEntry   `json:"toolDiffs"`
	TurnDiffs  []TurnDiffEntry   `json:"turnDiffs"`
}

// ToolDiffEntry mirrors eval.ToolDiff for Wails JSON serialization.
// Wails discovers types by scanning the main package's AST; it cannot
// resolve structs from reasonix/internal/eval, so we define a local copy
// with the same shape and populate it from the eval package's result.
type ToolDiffEntry struct {
	Name   string `json:"name"`
	CountA int    `json:"countA"`
	CountB int    `json:"countB"`
	Delta  int    `json:"delta"`
}

// TurnDiffEntry mirrors eval.TurnDiff. See ToolDiffEntry for rationale.
type TurnDiffEntry struct {
	Index    int      `json:"index"`
	Match    bool     `json:"match"`
	ToolsA   []string `json:"toolsA"`
	ToolsB   []string `json:"toolsB"`
	MissingA []string `json:"missingA"`
	MissingB []string `json:"missingB"`
}

// CompareSessions loads and compares two saved session files.
func (a *App) CompareSessions(pathA, pathB string) (SessionComparisonView, error) {
	snapA, err := eval.LoadSessionSnapshot(pathA)
	if err != nil {
		return SessionComparisonView{}, fmt.Errorf("session A: %w", err)
	}
	snapB, err := eval.LoadSessionSnapshot(pathB)
	if err != nil {
		return SessionComparisonView{}, fmt.Errorf("session B: %w", err)
	}

	r := eval.Compare(snapA, snapB)

	view := SessionComparisonView{
		SessionA:   r.SessionA,
		SessionB:   r.SessionB,
		TokensInA:  snapA.Meta.TokensIn,
		TokensInB:  snapB.Meta.TokensIn,
		TokensOutA: snapA.Meta.TokensOut,
		TokensOutB: snapB.Meta.TokensOut,
		TurnsA:     snapA.Meta.TurnCount,
		TurnsB:     snapB.Meta.TurnCount,
		CostA:      snapA.Meta.Cost,
		CostB:      snapB.Meta.Cost,
		Currency:   snapB.Meta.Currency,
		Similarity: r.Similarity,
		TextReport: r.FormatText(),
	}

	for _, td := range r.ToolDiffs {
		view.ToolDiffs = append(view.ToolDiffs, ToolDiffEntry{
			Name:   td.Name,
			CountA: td.CountA,
			CountB: td.CountB,
			Delta:  td.Delta,
		})
	}
	for _, td := range r.TurnDiffs {
		view.TurnDiffs = append(view.TurnDiffs, TurnDiffEntry{
			Index:    td.Index,
			Match:    td.Match,
			ToolsA:   td.ToolsA,
			ToolsB:   td.ToolsB,
			MissingA: td.MissingA,
			MissingB: td.MissingB,
		})
	}

	return view, nil
}
