package cli

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"reasonix/internal/i18n"
	"reasonix/internal/learn"
)

// showLearn displays detected patterns and trajectories from the learner.
// Handles subcommands: "/learn patterns" (default), "/learn trajectories",
// "/learn reflect" — builds a reflection prompt and feeds it to the agent.
func (m *chatTUI) showLearn(input string) tea.Cmd {
	learner := m.ctrl.Learner()
	if learner == nil {
		m.notice(i18n.M.CmdLearn + " — learner not enabled. Set [learn].enabled = true in reasonix.toml")
		return nil
	}

	args := strings.Fields(input)
	mode := "patterns"
	if len(args) > 1 {
		mode = args[1]
	}

	switch mode {
	case "trajectories", "traj":
		m.showTrajectories(learner)
		return nil
	case "reflect":
		prompt := learner.BuildReflectionPrompt()
		if prompt == "" {
			m.notice("no observations to reflect on — the learner needs data from agent turns first")
			return nil
		}
		return m.startTurn(prompt, "/learn reflect", input)
	default:
		m.showPatterns(learner)
		return nil
	}
}

func (m *chatTUI) showPatterns(l *learn.Learner) {
	patterns := l.Patterns()
	if len(patterns) == 0 {
		m.notice("no patterns detected yet — the learner observes turns as you work")
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Learner: %d pattern(s) detected\n", len(patterns)))
	for i, p := range patterns {
		conf := "▁"
		if p.Confidence >= 3 {
			conf = "▃"
		}
		if p.Confidence >= 5 {
			conf = "▅"
		}
		if p.Confidence >= 8 {
			conf = "█"
		}
		b.WriteString(fmt.Sprintf("  %s %s → %s (×%d)\n", conf, p.Trigger, p.Action, p.Confidence))
		_ = i
	}
	b.WriteString(fmt.Sprintf("\n/learn trajectories — view multi-turn sequences\n"))
	b.WriteString(fmt.Sprintf("/learn reflect — have the agent reflect and generate skills\n"))
	b.WriteString(fmt.Sprintf("Set [learn].min_confidence to tune sensitivity"))

	m.commitLine(b.String())
}

func (m *chatTUI) showTrajectories(l *learn.Learner) {
	trajs := l.Trajectories()
	if len(trajs) == 0 {
		m.notice("no trajectories recorded yet")
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Learner: %d trajectory(s)\n", len(trajs)))
	for _, t := range trajs {
		b.WriteString(fmt.Sprintf("  %s (%d turns, %d×)\n", t.Label, t.Turns, t.Count))
	}
	b.WriteString(fmt.Sprintf("\n/learn patterns — view detected patterns\n"))
	b.WriteString(fmt.Sprintf("/learn reflect — have the agent reflect and generate skills"))

	m.commitLine(b.String())
}
