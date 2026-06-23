// Package scheduler runs user-configured cron tasks through the agent loop.
// Tasks are defined in reasonix.toml under [[schedule.tasks]] with standard
// 5-field cron expressions. When a task fires, the scheduler sends the prompt
// to the controller, captures the result, and stores it for later retrieval.
//
// The scheduler is optional — it is only started when at least one task is
// configured and the controller is available. It runs in a background goroutine
// and stops cleanly when the context is canceled.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Task is one scheduled job.
type Task struct {
	Name    string `toml:"name"`
	Cron    string `toml:"cron"`
	Prompt  string `toml:"prompt"`
	Model   string `toml:"model"`  // optional; empty = use default
	Enabled *bool  `toml:"enabled"` // nil or true = enabled
}

func (t Task) isEnabled() bool { return t.Enabled == nil || *t.Enabled }

// IsEnabled reports whether the task is enabled.
func (t Task) IsEnabled() bool { return t.isEnabled() }

// Config is the [schedule] block.
type Config struct {
	Tasks []Task `toml:"tasks"`
}

// Result is the outcome of one scheduled run.
type Result struct {
	TaskName  string    `json:"taskName"`
	RunAt     time.Time `json:"runAt"`
	Duration  time.Duration `json:"duration"`
	Success   bool      `json:"success"`
	Summary   string    `json:"summary"`   // first 500 chars of response
	Error     string    `json:"error,omitempty"`
}

// Sender is the interface the scheduler uses to submit prompts to the agent.
type Sender interface {
	Send(ctx context.Context, text string) error
}

// SenderFunc wraps a function as a Sender.
type SenderFunc func(ctx context.Context, text string) error

func (f SenderFunc) Send(ctx context.Context, text string) error { return f(ctx, text) }

// Scheduler manages cron-driven agent tasks.
type Scheduler struct {
	config    Config
	sender    Sender
	logger    *slog.Logger
	results   []Result
	mu        sync.Mutex
	nextRun   map[string]time.Time // task name → next scheduled run
	parentCtx context.Context      // set by Start, used as parent for task contexts
}

// New creates a scheduler. It returns nil if no tasks are configured.
func New(cfg Config, sender Sender, logger *slog.Logger) *Scheduler {
	enabled := 0
	for _, t := range cfg.Tasks {
		if t.isEnabled() {
			enabled++
		}
	}
	if enabled == 0 {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{
		config:  cfg,
		sender:  sender,
		logger:  logger,
		results: make([]Result, 0),
		nextRun: make(map[string]time.Time),
	}
}

// Start begins the scheduler loop. It blocks until ctx is canceled.
func (s *Scheduler) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.logger.Info("scheduler: starting", "tasks", len(s.config.Tasks))
	s.parentCtx = ctx
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Compute initial next-run times.
	s.recomputeAll()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler: stopped")
			return
		case now := <-ticker.C:
			s.fireDue(now)
			s.recomputeAll()
		}
	}
}

// fireDue runs every task whose next scheduled time has arrived.
func (s *Scheduler) fireDue(now time.Time) {
	for i := range s.config.Tasks {
		t := &s.config.Tasks[i]
		if !t.isEnabled() {
			continue
		}
		next, ok := s.nextRun[t.Name]
		if !ok {
			continue
		}
		if now.Before(next) {
			continue
		}
		s.logger.Info("scheduler: firing task", "name", t.Name)
		s.runTask(t)
	}
}

// runTask executes one task and records the result.
func (s *Scheduler) runTask(t *Task) {
	start := time.Now()
	parent := s.parentCtx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()

	result := Result{
		TaskName: t.Name,
		RunAt:    start,
	}

	err := s.sender.Send(ctx, t.Prompt)
	result.Duration = time.Since(start)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		s.logger.Error("scheduler: task failed", "name", t.Name, "err", err)
	} else {
		result.Success = true
		result.Summary = "task dispatched"
		s.logger.Info("scheduler: task dispatched", "name", t.Name, "duration", result.Duration)
	}

	s.mu.Lock()
	s.results = append(s.results, result)
	if len(s.results) > 100 {
		s.results = s.results[len(s.results)-100:]
	}
	s.mu.Unlock()
}

// Results returns the last N results, newest first.
func (s *Scheduler) Results(limit int) []Result {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.results)
	start := n - limit
	if start < 0 {
		start = 0
	}
	out := make([]Result, n-start)
	copy(out, s.results[start:])
	// Reverse to newest-first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// NextRun returns the next scheduled run time for a task, and whether the task exists.
func (s *Scheduler) NextRun(name string) (time.Time, bool) {
	if s == nil {
		return time.Time{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.nextRun[name]
	return t, ok
}

// AllNextRuns returns all task → next-run mappings.
func (s *Scheduler) AllNextRuns() map[string]time.Time {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]time.Time, len(s.nextRun))
	for k, v := range s.nextRun {
		out[k] = v
	}
	return out
}

// recomputeAll updates next-run for all enabled tasks.
func (s *Scheduler) recomputeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := range s.config.Tasks {
		t := &s.config.Tasks[i]
		if !t.isEnabled() {
			continue
		}
		next, err := NextAfter(t.Cron, now)
		if err != nil {
			s.logger.Warn("scheduler: bad cron expression", "name", t.Name, "cron", t.Cron, "err", err)
			continue
		}
		s.nextRun[t.Name] = next
	}
}

// ── minimal cron parser ─────────────────────────────────────────────────────

// cronField holds the parsed values for one cron field (minute, hour, dom, month, dow).
type cronField struct {
	values map[int]bool // explicit set
	star   bool          // true if * (any value)
}

// cronExpr is a parsed 5-field cron expression.
type cronExpr struct {
	minute cronField
	hour   cronField
	dom    cronField
	month  cronField
	dow    cronField
}

// parseCron parses a standard 5-field cron expression: "min hour dom month dow".
// Supports: *, explicit values (0,15,30), ranges (1-5), steps (*/5, 1-10/2),
// and comma-separated combinations. Returns an error on syntax problems.
func parseCron(expr string) (*cronExpr, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron: expected 5 fields, got %d", len(fields))
	}
	e := &cronExpr{}
	ranges := []struct {
		name  string
		min   int
		max   int
		field *cronField
	}{
		{"minute", 0, 59, &e.minute},
		{"hour", 0, 23, &e.hour},
		{"dom", 1, 31, &e.dom},
		{"month", 1, 12, &e.month},
		{"dow", 0, 7, &e.dow}, // 0 and 7 both = Sunday
	}
	for i, f := range fields {
		cf, err := parseCronField(f, ranges[i].min, ranges[i].max)
		if err != nil {
			return nil, fmt.Errorf("cron: field %d (%s: %q): %w", i+1, ranges[i].name, f, err)
		}
		*ranges[i].field = cf
	}
	return e, nil
}

func parseCronField(s string, min, max int) (cronField, error) {
	if s == "*" {
		return cronField{star: true}, nil
	}
	cf := cronField{values: make(map[int]bool)}
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if stepIdx := strings.Index(part, "/"); stepIdx >= 0 {
			// Range with step: "1-10/2" or "*/5"
			rangePart := part[:stepIdx]
			step, err := strconv.Atoi(part[stepIdx+1:])
			if err != nil || step < 1 {
				return cronField{}, fmt.Errorf("invalid step in %q", part)
			}
			lo, hi := min, max
			if rangePart != "*" {
				if dash := strings.Index(rangePart, "-"); dash >= 0 {
					lo, err = strconv.Atoi(rangePart[:dash])
					if err != nil {
						return cronField{}, fmt.Errorf("invalid range start in %q", part)
					}
					hi, err = strconv.Atoi(rangePart[dash+1:])
					if err != nil {
						return cronField{}, fmt.Errorf("invalid range end in %q", part)
					}
				} else {
					return cronField{}, fmt.Errorf("expected range in step expression %q", part)
				}
			}
			for v := lo; v <= hi; v += step {
				cf.values[v] = true
			}
		} else if dash := strings.Index(part, "-"); dash >= 0 {
			lo, err := strconv.Atoi(part[:dash])
			if err != nil {
				return cronField{}, fmt.Errorf("invalid range in %q", part)
			}
			hi, err := strconv.Atoi(part[dash+1:])
			if err != nil {
				return cronField{}, fmt.Errorf("invalid range in %q", part)
			}
			for v := lo; v <= hi; v++ {
				cf.values[v] = true
			}
		} else {
			v, err := strconv.Atoi(part)
			if err != nil {
				return cronField{}, fmt.Errorf("invalid value %q", part)
			}
			cf.values[v] = true
		}
	}
	return cf, nil
}

func (f cronField) matches(v int) bool {
	if f.star {
		return true
	}
	return f.values[v]
}

// NextAfter returns the next time after `after` when the cron expression fires.
// It scans minute-by-minute up to 2 years ahead, then gives up.
func NextAfter(expr string, after time.Time) (time.Time, error) {
	e, err := parseCron(expr)
	if err != nil {
		return time.Time{}, err
	}
	// Start at the next minute boundary.
	t := after.Truncate(time.Minute).Add(time.Minute)
	deadline := after.Add(2 * 365 * 24 * time.Hour)
	for t.Before(deadline) {
		if e.matches(t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("cron: no match within 2 years for %q", expr)
}

func (e *cronExpr) matches(t time.Time) bool {
	dow := int(t.Weekday())
	// DOW: 0 and 7 both map to Sunday. Go's Weekday() returns 0 for Sunday
	// and never returns 7, so check both when dow == 0.
	dowMatch := e.dow.matches(dow)
	if dow == 0 {
		dowMatch = dowMatch || e.dow.matches(7)
	}
	return e.minute.matches(t.Minute()) &&
		e.hour.matches(t.Hour()) &&
		e.dom.matches(t.Day()) &&
		e.month.matches(int(t.Month())) &&
		dowMatch
}

// Tasks returns a copy of the configured task list for display.
func (s *Scheduler) Tasks() []Task {
	if s == nil {
		return nil
	}
	out := make([]Task, len(s.config.Tasks))
	copy(out, s.config.Tasks)
	return out
}

// AddTask adds a task to the scheduler at runtime and recomputes next-run times.
// It replaces an existing task with the same name. Returns false if the scheduler
// is nil.
func (s *Scheduler) AddTask(t Task) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.config.Tasks {
		if existing.Name == t.Name {
			s.config.Tasks[i] = t
			s.logger.Info("scheduler: task updated", "name", t.Name)
			s.recomputeOne(&s.config.Tasks[i])
			return true
		}
	}
	s.config.Tasks = append(s.config.Tasks, t)
	s.logger.Info("scheduler: task added", "name", t.Name)
	s.recomputeOne(&s.config.Tasks[len(s.config.Tasks)-1])
	return true
}

// RemoveTask removes a task by name and recomputes next-run times.
// Returns false if the scheduler is nil or the name is not found.
func (s *Scheduler) RemoveTask(name string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, t := range s.config.Tasks {
		if t.Name == name {
			s.config.Tasks = append(s.config.Tasks[:i], s.config.Tasks[i+1:]...)
			delete(s.nextRun, name)
			s.logger.Info("scheduler: task removed", "name", name)
			return true
		}
	}
	return false
}

// recomputeOne recalculates the next-run time for a single task.
func (s *Scheduler) recomputeOne(t *Task) {
	if !t.isEnabled() {
		delete(s.nextRun, t.Name)
		return
	}
	next, err := NextAfter(t.Cron, time.Now())
	if err != nil {
		s.logger.Warn("scheduler: bad cron for task", "name", t.Name, "cron", t.Cron, "err", err)
		delete(s.nextRun, t.Name)
		return
	}
	s.nextRun[t.Name] = next
}

// SortTasksByName sorts a task slice by name for stable display ordering.
func SortTasksByName(tasks []Task) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name < tasks[j].Name
	})
}
