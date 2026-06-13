package main

import (
	"path/filepath"
	"strings"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/constitution"
)

// --- Sub-agent Task Tree ---

// SubagentNodeView is one node in the sub-agent task tree.
type SubagentNodeView struct {
	Ref           string             `json:"ref"`
	Name          string             `json:"name"`
	Kind          string             `json:"kind"`
	Model         string             `json:"model"`
	Effort        string             `json:"effort"`
	Status        string             `json:"status"`
	ParentSession string             `json:"parentSession,omitempty"`
	CreatedAt     string             `json:"createdAt"`
	Children      []SubagentNodeView `json:"children"`
}

// SubagentTree returns the sub-agent task tree for the active session.
func (a *App) SubagentTree() []SubagentNodeView {
	return a.SubagentTreeForTab("")
}

func (a *App) SubagentTreeForTab(tabID string) []SubagentNodeView {
	a.mu.RLock()
	tab := a.tabByIDLocked(tabID)
	var sessionDir string
	var sessionPath string
	if tab != nil && tab.Ctrl != nil {
		sessionDir = tab.Ctrl.SessionDir()
		sessionPath = tab.Ctrl.SessionPath()
	}
	a.mu.RUnlock()

	if sessionDir == "" {
		return []SubagentNodeView{}
	}
	sessionID := strings.TrimSuffix(filepath.Base(sessionPath), filepath.Ext(sessionPath))

	artifacts, err := agent.ListSubagentsByParent(sessionDir, sessionID)
	if err != nil || len(artifacts) == 0 {
		return []SubagentNodeView{}
	}

	nodes := make([]SubagentNodeView, 0, len(artifacts))
	for _, art := range artifacts {
		nodes = append(nodes, SubagentNodeView{
			Ref:           art.Ref,
			Name:          art.Meta.Name,
			Kind:          art.Meta.Kind,
			Model:         art.Meta.Model,
			Effort:        art.Meta.Effort,
			Status:        string(art.Meta.Status),
			ParentSession: art.Meta.ParentSession,
			CreatedAt:     art.Meta.CreatedAt.Format(time.RFC3339),
		})
	}
	return nodes
}

// --- Constitution Health Check ---

// ConstitutionRuleView is one rule from .reasonix/constitution.json.
type ConstitutionRuleView struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
	Severity    string `json:"severity"`
}

// ConstitutionHealthView carries the constitution health check result.
type ConstitutionHealthView struct {
	Loaded      bool                   `json:"loaded"`
	Path        string                 `json:"path"`
	Version     int                    `json:"version"`
	Rules       []ConstitutionRuleView `json:"rules"`
	Principles  []string               `json:"principles"`
	Constraints []string               `json:"constraints"`
	Status      string                 `json:"status"`
}

// ConstitutionHealth reads .reasonix/constitution.json from the project root.
func (a *App) ConstitutionHealth() ConstitutionHealthView {
	a.mu.RLock()
	tab := a.tabByIDLocked("")
	var root string
	if tab != nil && tab.Ctrl != nil {
		root = tab.Ctrl.WorkspaceRoot()
	}
	a.mu.RUnlock()

	if root == "" {
		return ConstitutionHealthView{Loaded: false, Status: "no_config"}
	}

	doc, ok := constitution.Load(root)
	if !ok {
		return ConstitutionHealthView{
			Loaded: false,
			Path:   filepath.Join(root, constitution.Dir, constitution.File),
			Status: "no_config",
		}
	}

	rules := make([]ConstitutionRuleView, 0, len(doc.Rules))
	for _, r := range doc.Rules {
		sev := r.Severity
		if sev == "" {
			sev = "warning"
		}
		rules = append(rules, ConstitutionRuleView{
			ID:          r.ID,
			Description: r.Description,
			Scope:       r.Scope,
			Severity:    sev,
		})
	}

	principles := doc.Principles
	if principles == nil {
		principles = []string{}
	}
	constraints := doc.Constraints
	if constraints == nil {
		constraints = []string{}
	}

	return ConstitutionHealthView{
		Loaded:      true,
		Path:        filepath.Join(root, constitution.Dir, constitution.File),
		Version:     doc.Version,
		Rules:       rules,
		Principles:  principles,
		Constraints: constraints,
		Status:      "healthy",
	}
}
