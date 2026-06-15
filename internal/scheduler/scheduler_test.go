package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestParseCronStar(t *testing.T) {
	t.Parallel()
	e, err := parseCron("* * * * *")
	if err != nil {
		t.Fatal(err)
	}
	if !e.minute.star || !e.hour.star || !e.dom.star || !e.month.star || !e.dow.star {
		t.Error("all fields should be star")
	}
	// Every minute of every day matches.
	now := time.Date(2026, 6, 13, 14, 30, 0, 0, time.UTC)
	next, err := NextAfter("* * * * *", now)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Equal(time.Date(2026, 6, 13, 14, 31, 0, 0, time.UTC)) {
		t.Errorf("next = %v, want 14:31", next)
	}
}

func TestParseCronSpecificValues(t *testing.T) {
	t.Parallel()
	// Every day at 02:00
	e, err := parseCron("0 2 * * *")
	if err != nil {
		t.Fatal(err)
	}
	if e.minute.star {
		t.Error("minute should not be star")
	}
	if !e.minute.values[0] {
		t.Error("minute should match 0")
	}
	if !e.hour.values[2] {
		t.Error("hour should match 2")
	}
	if !e.dom.star {
		t.Error("dom should be star")
	}

	// Test next fire time.
	now := time.Date(2026, 6, 13, 14, 30, 0, 0, time.UTC)
	next, err := NextAfter("0 2 * * *", now)
	if err != nil {
		t.Fatal(err)
	}
	// Should be next day at 02:00.
	want := time.Date(2026, 6, 14, 2, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestParseCronRange(t *testing.T) {
	t.Parallel()
	e, err := parseCron("0 9-17 * * 1-5")
	if err != nil {
		t.Fatal(err)
	}
	for h := 9; h <= 17; h++ {
		if !e.hour.values[h] {
			t.Errorf("hour %d should match", h)
		}
	}
	if e.hour.values[8] {
		t.Error("hour 8 should NOT match")
	}
	for d := 1; d <= 5; d++ {
		if !e.dow.values[d] {
			t.Errorf("dow %d should match", d)
		}
	}
}

func TestParseCronStep(t *testing.T) {
	t.Parallel()
	e, err := parseCron("*/15 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	// Every 15 minutes: 0, 15, 30, 45
	for _, m := range []int{0, 15, 30, 45} {
		if !e.minute.values[m] {
			t.Errorf("minute %d should match", m)
		}
	}
	if e.minute.values[1] {
		t.Error("minute 1 should NOT match */15")
	}

	// Test next fire at e.g. 14:07 → 14:15.
	now := time.Date(2026, 6, 13, 14, 7, 0, 0, time.UTC)
	next, err := NextAfter("*/15 * * * *", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 13, 14, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestParseCronCommaSeparated(t *testing.T) {
	t.Parallel()
	e, err := parseCron("0,30 0,12 * * *")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []int{0, 30} {
		if !e.minute.values[m] {
			t.Errorf("minute %d should match", m)
		}
	}
	for _, h := range []int{0, 12} {
		if !e.hour.values[h] {
			t.Errorf("hour %d should match", h)
		}
	}
}

func TestParseCronRangeWithStep(t *testing.T) {
	t.Parallel()
	e, err := parseCron("0 0-23/3 * * *")
	if err != nil {
		t.Fatal(err)
	}
	// Every 3 hours: 0, 3, 6, 9, 12, 15, 18, 21
	for _, h := range []int{0, 3, 6, 9, 12, 15, 18, 21} {
		if !e.minute.values[0] {
			t.Error("minute 0 should match")
		}
		if !e.hour.values[h] {
			t.Errorf("hour %d should match", h)
		}
	}
	if e.hour.values[1] || e.hour.values[2] {
		t.Error("hours 1,2 should NOT match 0-23/3")
	}
}

func TestParseCronErrors(t *testing.T) {
	t.Parallel()
	tests := []string{
		"* * *",          // too few fields
		"* * * * * *",    // too many
		"abc * * * *",     // non-numeric
		"60 * * * *",      // out of range
		"* 24 * * *",
		"* * 32 * *",
		"* * * 13 *",
		"* * * * 8",
	}
	for _, expr := range tests {
		_, err := NextAfter(expr, time.Now())
		if err == nil {
			t.Errorf("expected error for %q", expr)
		}
	}
}

func TestNextAfterMidnight(t *testing.T) {
	t.Parallel()
	// At 23:59, next "0 0 * * *" should be 00:00.
	now := time.Date(2026, 6, 13, 23, 59, 0, 0, time.UTC)
	next, err := NextAfter("0 0 * * *", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextAfterMonthBoundary(t *testing.T) {
	t.Parallel()
	// On Jan 31, next "0 0 1 * *" should be Feb 1.
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	next, err := NextAfter("0 0 1 * *", now)
	if err != nil {
		t.Fatal(err)
	}
	if next.Month() != 2 || next.Day() != 1 {
		t.Errorf("next = %v, want Feb 1", next)
	}
}

func TestSchedulerNewEmpty(t *testing.T) {
	t.Parallel()
	s := New(Config{}, nil, nil)
	if s != nil {
		t.Error("expected nil scheduler for empty config")
	}
}

func TestSchedulerNewDisabled(t *testing.T) {
	t.Parallel()
	disabled := false
	cfg := Config{
		Tasks: []Task{
			{Name: "test", Cron: "* * * * *", Prompt: "hello", Enabled: &disabled},
		},
	}
	s := New(cfg, nil, nil)
	if s != nil {
		t.Error("expected nil scheduler when all tasks are disabled")
	}
}

func TestSchedulerNewEnabled(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Tasks: []Task{
			{Name: "daily", Cron: "0 2 * * *", Prompt: "run tests"},
		},
	}
	s := New(cfg, &stubSender{}, nil)
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
	if len(s.config.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(s.config.Tasks))
	}
}

func TestSchedulerResults(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{{Name: "t", Cron: "* * * * *", Prompt: "x"}},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	if r := s.Results(10); len(r) != 0 {
		t.Error("expected no results initially")
	}

	// Newest first: the "ok" result was added last (by runTask), so after reversal
	// it should come first.
	s.results = []Result{
		{TaskName: "t", RunAt: time.Now().Add(-time.Hour), Success: false, Error: "timeout"},
		{TaskName: "t", RunAt: time.Now(), Success: true, Summary: "ok"},
	}

	r := s.Results(10)
	if len(r) != 2 {
		t.Fatalf("expected 2 results, got %d", len(r))
	}
	// Newest first (reversed from append order).
	if r[0].Summary != "ok" {
		t.Errorf("newest result should be 'ok', got %q", r[0].Summary)
	}
	if r[1].Error != "timeout" {
		t.Errorf("oldest result should be 'timeout', got %q", r[1].Error)
	}

	// Limit.
	r = s.Results(1)
	if len(r) != 1 {
		t.Errorf("limit 1 should return 1, got %d", len(r))
	}
}

func TestSchedulerTaskEnabled(t *testing.T) {
	t.Parallel()
	t1 := Task{Name: "a"} // nil = enabled
	if !t1.isEnabled() {
		t.Error("nil Enabled should be treated as enabled")
	}
	e := true
	t2 := Task{Name: "b", Enabled: &e}
	if !t2.isEnabled() {
		t.Error("Enabled=true should be enabled")
	}
	d := false
	t3 := Task{Name: "c", Enabled: &d}
	if t3.isEnabled() {
		t.Error("Enabled=false should be disabled")
	}
}

type stubSender struct{}

func (s *stubSender) Send(_ context.Context, text string) error {
	return nil
}

func TestSortTasksByName(t *testing.T) {
	t.Parallel()
	tasks := []Task{
		{Name: "z"},
		{Name: "alpha"},
		{Name: "beta"},
	}
	SortTasksByName(tasks)
	if tasks[0].Name != "alpha" || tasks[1].Name != "beta" || tasks[2].Name != "z" {
		t.Errorf("wrong sort order: %v", taskNames(tasks))
	}
}

func taskNames(tasks []Task) []string {
	names := make([]string, len(tasks))
	for i, t := range tasks {
		names[i] = t.Name
	}
	return names
}

func TestAddTask(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{{Name: "t1", Cron: "* * * * *", Prompt: "hello"}},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	// Add a new task.
	if !s.AddTask(Task{Name: "t2", Cron: "0 2 * * *", Prompt: "daily"}) {
		t.Fatal("AddTask returned false")
	}
	if n := s.Tasks(); len(n) != 2 {
		t.Fatalf("expected 2 tasks, got %d: %v", len(n), taskNames(n))
	}

	// Update existing task.
	s.AddTask(Task{Name: "t1", Cron: "0 9 * * 1-5", Prompt: "updated"})
	tasks := s.Tasks()
	if len(tasks) != 2 {
		t.Fatalf("expected still 2 tasks, got %d", len(tasks))
	}
	for _, tk := range tasks {
		if tk.Name == "t1" && tk.Prompt != "updated" {
			t.Errorf("t1 not updated: %+v", tk)
		}
	}
}

func TestRemoveTask(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{
			{Name: "t1", Cron: "* * * * *", Prompt: "a"},
			{Name: "t2", Cron: "0 2 * * *", Prompt: "b"},
		},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	if !s.RemoveTask("t1") {
		t.Fatal("RemoveTask t1 returned false")
	}
	ns := s.Tasks()
	if len(ns) != 1 {
		t.Fatalf("expected 1 task, got %d", len(ns))
	}
	if ns[0].Name != "t2" {
		t.Errorf("expected t2 to remain, got %s", ns[0].Name)
	}

	// Remove non-existent task.
	if s.RemoveTask("nope") {
		t.Error("RemoveTask on non-existent should return false")
	}

	// Remove last task.
	s.RemoveTask("t2")
	if n := s.Tasks(); len(n) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(n))
	}
}

func TestAddRemoveTaskNilScheduler(t *testing.T) {
	t.Parallel()
	var s *Scheduler
	if s.AddTask(Task{Name: "t"}) {
		t.Error("AddTask on nil scheduler should return false")
	}
	if s.RemoveTask("t") {
		t.Error("RemoveTask on nil scheduler should return false")
	}
}
