// Package learn provides self-improving skill loops for Reasonix. It observes
// agent behaviour across turns, detects repeated tool/skill patterns, and
// surfaces them so a post-session reflection turn can generate new or updated
// SKILL.md files.
//
// The learner is a passive observer — it never calls the agent itself. A
// reflection turn is triggered externally (by boot.go at session end) using
// the accumulated observations as input.
//
// Architecture: boot.go creates a Learner, passes it to agent.Options.Learner.
// The agent calls Observe(turn) after each turn. On session close, boot.go
// reads Observations() and feeds them to the controller for synthesis.
package learn

import (
	"fmt"
	"strings"
	"sync"
)

// ToolCallInfo captures one tool invocation observed during a turn.
type ToolCallInfo struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Brief   string `json:"brief"` // first 200 chars of args, trimmed
}

// TurnObservation captures the key facts from one agent turn.
type TurnObservation struct {
	Turn       int            `json:"turn"`
	Task       string         `json:"task"` // first 200 chars of task
	ToolCalls  []ToolCallInfo `json:"toolCalls"`
	SkillName  string         `json:"skillName,omitempty"`  // if a skill was invoked
	SkillArgs  string         `json:"skillArgs,omitempty"`  // skill arguments
	Compacted  bool           `json:"compacted,omitempty"`  // this turn triggered compaction
}

// Pattern represents a detected repeated behaviour that could become a skill.
type Pattern struct {
	Name       string `json:"name"`
	Trigger    string `json:"trigger"`    // when to apply (e.g. "after editing Go files")
	Action     string `json:"action"`     // what to do (e.g. "run go build ./...")
	Confidence int    `json:"confidence"` // number of observations
}

// Config controls the learner.
type Config struct {
	Enabled        bool `toml:"enabled"`
	MaxPatterns    int  `toml:"max_patterns"`    // max patterns to detect (default 20)
	MinConfidence  int  `toml:"min_confidence"`   // observations needed to form a pattern (default 3)
	MaxObservations int `toml:"max_observations"` // ring buffer cap (default 200)
}

// Learner collects turn observations and detects repeated behaviour patterns.
// It is safe for concurrent use.
type Learner struct {
	mu             sync.Mutex
	observations   []TurnObservation
	patterns       []Pattern
	enabled        bool
	maxObs         int
	minConfidence  int
	nextTurn       int
}

// New creates a Learner. Pass enabled=false to make Observe a no-op.
func New(cfg Config) *Learner {
	maxObs := cfg.MaxObservations
	if maxObs <= 0 {
		maxObs = 200
	}
	if maxObs > 2000 {
		maxObs = 2000
	}
	minConf := cfg.MinConfidence
	if minConf <= 0 {
		minConf = 3
	}
	maxPatterns := cfg.MaxPatterns
	if maxPatterns <= 0 {
		maxPatterns = 20
	}
	return &Learner{
		enabled:       cfg.Enabled,
		maxObs:        maxObs,
		minConfidence: minConf,
		observations:  make([]TurnObservation, 0, maxObs),
		patterns:      make([]Pattern, 0, maxPatterns),
		nextTurn:      1,
	}
}

// Observe records one turn's worth of behaviour. No-op when disabled.
func (l *Learner) Observe(task string, toolCalls []ToolCallInfo, skillName, skillArgs string) {
	if !l.enabled {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	turn := l.nextTurn
	l.nextTurn++

	obs := TurnObservation{
		Turn:      turn,
		Task:      truncate(task, 200),
		ToolCalls: toolCalls,
		SkillName: skillName,
		SkillArgs: truncate(skillArgs, 200),
	}

	// Ring-buffer eviction
	if len(l.observations) >= l.maxObs {
		l.observations = l.observations[1:]
	}
	l.observations = append(l.observations, obs)
}

// Observations returns a copy of all accumulated turn observations.
func (l *Learner) Observations() []TurnObservation {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]TurnObservation, len(l.observations))
	copy(out, l.observations)
	return out
}

// Patterns returns the detected repeated-behaviour patterns.
func (l *Learner) Patterns() []Pattern {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.detectPatterns()
	out := make([]Pattern, len(l.patterns))
	copy(out, l.patterns)
	return out
}

// BuildReflectionPrompt constructs a prompt the agent can use to reflect on
// observed patterns and generate skill files.
func (l *Learner) BuildReflectionPrompt() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.detectPatterns()

	if len(l.observations) == 0 && len(l.patterns) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Reflect on these observed patterns from this session. ")
	b.WriteString("For each pattern that would make a useful reusable skill, ")
	b.WriteString("call `install_skill` to create it. A skill should capture a ")
	b.WriteString("common workflow that was repeated enough to automate.\n\n")

	if len(l.patterns) > 0 {
		b.WriteString("## Detected Patterns\n\n")
		for _, p := range l.patterns {
			b.WriteString(fmt.Sprintf("- **%s** (confidence=%d)\n", p.Name, p.Confidence))
			b.WriteString(fmt.Sprintf("  Trigger: %s\n", p.Trigger))
			b.WriteString(fmt.Sprintf("  Action: %s\n\n", p.Action))
		}
	}

	b.WriteString("## Recent Observations\n\n")
	start := 0
	if len(l.observations) > 10 {
		start = len(l.observations) - 10
	}
	for _, obs := range l.observations[start:] {
		b.WriteString(fmt.Sprintf("Turn %d: %s\n", obs.Turn, obs.Task))
		if len(obs.ToolCalls) > 0 {
			names := make([]string, len(obs.ToolCalls))
			for i, tc := range obs.ToolCalls {
				stat := " "
				if !tc.Success {
					stat = "✗"
				}
				names[i] = stat + tc.Name
			}
			b.WriteString(fmt.Sprintf("  Tools: %s\n", strings.Join(names, ", ")))
		}
	}

	return b.String()
}

// SuggestSkill generates a SKILL.md markdown draft from a detected pattern.
// The user should review and approve before it is written to disk.
func (l *Learner) SuggestSkill(p Pattern) string {
	var b strings.Builder
	name := strings.ReplaceAll(strings.ToLower(p.Name), " ", "-")
	if !strings.HasPrefix(name, "skill-") {
		name = "skill-" + name
	}
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", name))
	b.WriteString(fmt.Sprintf("description: Auto-detected pattern: %s (confidence=%d)\n", p.Name, p.Confidence))
	b.WriteString(fmt.Sprintf("trigger: %s\n", p.Trigger))
	b.WriteString("runAs: inline\n")
	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("# %s\n\n", p.Name))
	b.WriteString(fmt.Sprintf("**Detected by Reasonix learner** (observed %d times).\n\n", p.Confidence))
	b.WriteString(fmt.Sprintf("## Trigger\n\n%s\n\n", p.Trigger))
	b.WriteString(fmt.Sprintf("## Action\n\n%s\n\n", p.Action))
	b.WriteString("## Review\n\n")
	b.WriteString("This skill was automatically suggested. Please review the trigger and action ")
	b.WriteString("before saving — the learner may have detected a coincidence, not a genuine workflow.\n")
	return b.String()
}

// MultiTurnTrajectory groups consecutive turns that share a tool sequence into a
// higher-level pattern. It returns trajectory summaries useful for reflection.
type MultiTurnTrajectory struct {
	Label  string `json:"label"`  // e.g. "edit+test (3 turns)"
	Turns  []int  `json:"turns"`  // turn numbers in this trajectory
	Count  int    `json:"count"`  // number of turns
}

// Trajectories returns multi-turn sequences detected from observations.
// A trajectory is a run of consecutive turns that share the same tool-call
// pattern (same ordered tool names).
func (l *Learner) Trajectories() []MultiTurnTrajectory {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.observations) < 2 {
		return nil
	}

	// Build signatures for consecutive turns.
	type sig struct {
		key  string
		turn int
	}
	var sigs []sig
	for _, obs := range l.observations {
		if len(obs.ToolCalls) == 0 {
			continue
		}
		names := make([]string, len(obs.ToolCalls))
		for i, tc := range obs.ToolCalls {
			names[i] = tc.Name
		}
		sigs = append(sigs, sig{key: strings.Join(names, "→"), turn: obs.Turn})
	}

	if len(sigs) < 2 {
		return nil
	}

	// Group consecutive turns with the same signature.
	var trajectories []MultiTurnTrajectory
	current := MultiTurnTrajectory{Label: sigs[0].key, Turns: []int{sigs[0].turn}, Count: 1}
	for i := 1; i < len(sigs); i++ {
		if sigs[i].key == current.Label {
			current.Turns = append(current.Turns, sigs[i].turn)
			current.Count++
		} else {
			if current.Count >= 3 {
				trajectories = append(trajectories, current)
			}
			current = MultiTurnTrajectory{Label: sigs[i].key, Turns: []int{sigs[i].turn}, Count: 1}
		}
	}
	if current.Count >= 3 {
		trajectories = append(trajectories, current)
	}

	return trajectories
}

// Reset clears all observations and patterns.
func (l *Learner) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.observations = l.observations[:0]
	l.patterns = l.patterns[:0]
	l.nextTurn = 1
}

// detectPatterns scans the observation ring buffer for repeated sequences.
// Must be called while holding l.mu.
func (l *Learner) detectPatterns() {
	if len(l.observations) < l.minConfidence {
		return
	}

	// Strategy: group observations by the ordered sequence of tool names
	// used (after common edits). Look for identical tool sequences that
	// repeat across turns.
	type seqInfo struct {
		names   []string
		briefs  []string // parallel to names; first N chars of args
		preEdit bool
	}

	seqCounts := make(map[string]int)
	seqExample := make(map[string]TurnObservation)
	seqTags := make(map[string]seqInfo)

	for _, obs := range l.observations {
		if len(obs.ToolCalls) == 0 {
			continue
		}
		names := make([]string, len(obs.ToolCalls))
		briefs := make([]string, len(obs.ToolCalls))
		for i, tc := range obs.ToolCalls {
			names[i] = tc.Name
			briefs[i] = tc.Brief
		}
		key := strings.Join(names, ",")
		// Detect write-then-test patterns: if turn had write/edit/multi_edit
		// followed by test/build, mark it
		preEdit := false
		for _, tc := range obs.ToolCalls {
			if tc.Name == "write_file" || tc.Name == "edit_file" || tc.Name == "multi_edit" || tc.Name == "delete_range" {
				preEdit = true
				break
			}
		}
		seqCounts[key]++
		if seqCounts[key] == 1 {
			seqExample[key] = obs
			seqTags[key] = seqInfo{names: names, briefs: briefs, preEdit: preEdit}
		}
	}

	// Promoted patterns with sufficient confidence
	l.patterns = l.patterns[:0]
	patCap := 20
	if len(l.observations) > 0 {
		patCap = max(1, l.maxObs/10)
	}

	for key, count := range seqCounts {
		if count < l.minConfidence {
			continue
		}
		tag := seqTags[key]
		ex := seqExample[key]
		// Build a richer display name using briefs
		toolDisplay := make([]string, len(tag.names))
		for i, name := range tag.names {
			if i < len(tag.briefs) && tag.briefs[i] != "" {
				toolDisplay[i] = name + " " + tag.briefs[i]
			} else {
				toolDisplay[i] = name
			}
		}
		name := generatePatternName(strings.Join(toolDisplay, ","), tag.preEdit)
		trigger := "on every turn"
		if tag.preEdit {
			trigger = "after editing files"
		}
		action := fmt.Sprintf("run: %s", strings.Join(tag.names, " then "))
		if ex.Task != "" {
			action = fmt.Sprintf("when task is like %q, run: %s", ex.Task, strings.Join(tag.names, " then "))
		}
		l.patterns = append(l.patterns, Pattern{
			Name:       name,
			Trigger:    trigger,
			Action:     action,
			Confidence: count,
		})
		if len(l.patterns) >= patCap {
			break
		}
	}
}

func generatePatternName(richSeq string, preEdit bool) string {
	lower := strings.ToLower(richSeq)
	if preEdit {
		if strings.Contains(lower, "go test") {
			return "auto-test-go"
		}
		if strings.Contains(lower, "go build") || strings.Contains(lower, "go vet") {
			return "auto-verify-go"
		}
		if strings.Contains(lower, "npm") || strings.Contains(lower, "tsc") || strings.Contains(lower, "npx") {
			return "auto-check-js"
		}
		return "auto-verify"
	}
	if strings.Contains(lower, "explore") {
		return "deep-explore"
	}
	if strings.Contains(lower, "review") {
		return "auto-review"
	}
	// Extract just the tool names (strip briefs)
	clean := richSeq
	if idx := strings.Index(clean, " "); idx > 0 {
		// has briefs, rebuild with just names
		parts := strings.Split(clean, ",")
		names := make([]string, len(parts))
		for i, p := range parts {
			if idx2 := strings.Index(p, " "); idx2 > 0 {
				names[i] = p[:idx2]
			} else {
				names[i] = p
			}
		}
		clean = strings.Join(names, "-")
	}
	return "workflow-" + truncate(strings.ReplaceAll(clean, ",", "-"), 30)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
