// Package control is the transport-agnostic session driver.
package control

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"reasonix/internal/config"
)

// SendCtx implements scheduler.Sender. Unlike Send it is synchronous and
// reports failures: a task that arrives while another turn is in flight gets
// ErrTurnRunning rather than being silently dropped, and ctx bounds the turn —
// the caller's deadline cancels it and is returned.
func (c *Controller) SendCtx(ctx context.Context, text string) error {
	done, err := c.startGuarded(ctx, func(turnCtx context.Context) error {
		return c.runGoalLoopWithRaw(turnCtx, text, text)
	})
	if err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// The turn's context is derived from ctx, so it is already unwinding;
		// don't hold the caller past its own deadline waiting for it.
		return ctx.Err()
	}
}

func (c *Controller) ApplyProfile(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	name = strings.TrimSpace(name)

	if name != "" {
		p, ok := cfg.Profiles[name]
		if !ok {
			return fmt.Errorf("unknown profile %q; available: %s", name, profileNames(cfg.Profiles))
		}
		if p.Model != "" {
			cfg.ActiveProfile = name
			cfg.DefaultModel = p.Model
		}
		if p.Effort != "" {
			cfg.Agent.PlannerModel = ""
		}
		if p.ToolApproveMode != "" {
			c.SetToolApprovalMode(p.ToolApproveMode)
		}
		if p.AutoPlan != "" {
			c.SetAutoPlan(p.AutoPlan)
		}
		if p.OutputStyle != "" {
			cfg.Agent.OutputStyle = p.OutputStyle
		}
		cfg.ActiveProfile = name
	} else {
		cfg.ActiveProfile = ""
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if name != "" {
		c.notice("profile: switched to " + name + " — model: " + cfg.DefaultModel +
			", approve: " + c.ToolApprovalMode() +
			", auto-plan: " + c.PlanModeStr())
	} else {
		c.notice("profile: deactivated (using default settings)")
	}
	return nil
}

func (c *Controller) PlanModeStr() string {
	if c.PlanMode() {
		return "on"
	}
	return "off"
}

func (c *Controller) GoalBlocks() int {
	return c.goals.Blocks()
}

func (c *Controller) GoalTurns() int {
	return c.goals.Turns()
}

func (c *Controller) profileListText() string {
	cfg, err := config.Load()
	if err != nil {
		return "profile: " + err.Error()
	}
	if len(cfg.Profiles) == 0 {
		return "no harness profiles configured. Add [profiles.<name>] blocks to reasonix.toml or ~/.config/reasonix/config.toml."
	}
	var b strings.Builder
	active := cfg.ActiveProfile
	fmt.Fprintf(&b, "Harness profiles%s:\n", "")
	for name, p := range cfg.Profiles {
		mark := " "
		if name == active {
			mark = "*"
		}
		desc := p.Description
		if desc == "" {
			desc = p.Model
		}
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(&b, " %s %s — %s\n", mark, name, desc)
	}
	b.WriteString("switch with /profile <name>")
	return b.String()
}

func profileNames(profiles map[string]config.ProfileConfig) string {
	names := make([]string, 0, len(profiles))
	for n := range profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
