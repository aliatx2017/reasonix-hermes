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
		t.Errorf("wrong sort order: %v", taskNames(t, tasks))
	}
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s { return true }
	}
	return false
}

func taskNames(t *testing.T, tasks []Task) []string {
	t.Helper()
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
		t.Fatalf("expected 2 tasks, got %d: %v", len(n), taskNames(t, n))
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

// ── runTask ──────────────────────────────────────────────────────────────────

type recordingSender struct {
	calls []string
	err   error
}

func (r *recordingSender) Send(_ context.Context, text string) error {
	r.calls = append(r.calls, text)
	return r.err
}

func TestRunTaskSuccess(t *testing.T) {
	t.Parallel()
	sender := &recordingSender{}
	s := New(Config{
		Tasks: []Task{{Name: "test", Cron: "* * * * *", Prompt: "do it"}},
	}, sender, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}
	s.parentCtx = context.Background()

	s.runTask(&s.config.Tasks[0])

	if len(sender.calls) != 1 || sender.calls[0] != "do it" {
		t.Errorf("expected Send(do it), got %v", sender.calls)
	}
	results := s.Results(1)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Error("expected success")
	}
	if results[0].Summary != "task dispatched" {
		t.Errorf("unexpected summary: %q", results[0].Summary)
	}
}

func TestRunTaskFailure(t *testing.T) {
	t.Parallel()
	sender := &recordingSender{err: context.DeadlineExceeded}
	s := New(Config{
		Tasks: []Task{{Name: "test", Cron: "* * * * *", Prompt: "fail"}},
	}, sender, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}
	s.parentCtx = context.Background()

	s.runTask(&s.config.Tasks[0])

	results := s.Results(1)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected failure")
	}
	if results[0].Error == "" {
		t.Error("expected error message")
	}
}

func TestRunTaskNilParentCtx(t *testing.T) {
	t.Parallel()
	sender := &recordingSender{}
	s := New(Config{
		Tasks: []Task{{Name: "test", Cron: "* * * * *", Prompt: "x"}},
	}, sender, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}
	// parentCtx is nil by default; runTask should fall back to context.Background.
	s.runTask(&s.config.Tasks[0])

	results := s.Results(1)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Error("expected success even with nil parentCtx")
	}
}

func TestRunTaskResultTruncation(t *testing.T) {
	t.Parallel()
	sender := &recordingSender{}
	s := New(Config{
		Tasks: []Task{{Name: "test", Cron: "* * * * *", Prompt: "x"}},
	}, sender, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}
	s.parentCtx = context.Background()

	// Fill results to >100 and verify truncation keeps latest 100.
	for i := 0; i < 150; i++ {
		s.runTask(&s.config.Tasks[0])
	}

	r := s.Results(200)
	if len(r) != 100 {
		t.Errorf("results should be capped at 100, got %d", len(r))
	}
}

// ── fireDue ──────────────────────────────────────────────────────────────────

func TestFireDueFiresWhenPastNextRun(t *testing.T) {
	t.Parallel()
	sender := &recordingSender{}
	s := New(Config{
		Tasks: []Task{{Name: "hourly", Cron: "0 * * * *", Prompt: "tick"}},
	}, sender, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	// Set nextRun to a time in the past.
	s.nextRun["hourly"] = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// fireDue with "now" clearly past that.
	s.fireDue(time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC))

	// Task should have fired.
	results := s.Results(1)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TaskName != "hourly" {
		t.Errorf("expected hourly task, got %s", results[0].TaskName)
	}
}

func TestFireDueSkipsWhenNotDue(t *testing.T) {
	t.Parallel()
	sender := &recordingSender{}
	s := New(Config{
		Tasks: []Task{{Name: "future", Cron: "0 0 1 1 *", Prompt: "yearly"}},
	}, sender, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	// Set nextRun to a time far in the future.
	s.nextRun["future"] = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	// fireDue with "now" clearly before that.
	s.fireDue(time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC))

	// Task should NOT have fired.
	if len(s.Results(1)) != 0 {
		t.Error("task should not have fired")
	}
}

func TestFireDueSkipsDisabledTasks(t *testing.T) {
	t.Parallel()
	disabled := false
	sender := &recordingSender{}
	cfg := Config{
		Tasks: []Task{
			{Name: "enabled", Cron: "0 * * * *", Prompt: "go"},
			{Name: "disabled", Cron: "0 * * * *", Prompt: "nope", Enabled: &disabled},
		},
	}
	s := New(cfg, sender, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	// Both would fire based on time.
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	s.nextRun["enabled"] = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.nextRun["disabled"] = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	s.fireDue(now)

	results := s.Results(10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (only enabled task), got %d", len(results))
	}
	if results[0].TaskName != "enabled" {
		t.Errorf("expected enabled task, got %s", results[0].TaskName)
	}
}

func TestFireDueSkipsMissingNextRun(t *testing.T) {
	t.Parallel()
	sender := &recordingSender{}
	s := New(Config{
		Tasks: []Task{{Name: "orphan", Cron: "0 * * * *", Prompt: "x"}},
	}, sender, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	// No nextRun entry — should skip without panic.
	s.fireDue(time.Now())
	if len(s.Results(1)) != 0 {
		t.Error("task without nextRun should not fire")
	}
}

// ── NextRun / AllNextRuns ────────────────────────────────────────────────────

func TestNextRun(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{{Name: "test", Cron: "0 2 * * *", Prompt: "x"}},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	s.nextRun["test"] = time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC)

	got, ok := s.NextRun("test")
	if !ok {
		t.Fatal("expected to find task")
	}
	if !got.Equal(time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC)) {
		t.Errorf("got %v, want Jun 16 02:00", got)
	}

	_, ok = s.NextRun("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent task")
	}
}

func TestNextRunNil(t *testing.T) {
	t.Parallel()
	var s *Scheduler
	_, ok := s.NextRun("any")
	if ok {
		t.Error("nil scheduler should return false")
	}
}

func TestAllNextRuns(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{
			{Name: "a", Cron: "0 2 * * *", Prompt: "x"},
			{Name: "b", Cron: "0 9 * * 1-5", Prompt: "y"},
		},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	s.nextRun["a"] = time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC)
	s.nextRun["b"] = time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)

	all := s.AllNextRuns()
	if len(all) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(all))
	}
	if !all["a"].Equal(time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC)) {
		t.Errorf("a mismatch: %v", all["a"])
	}
	if !all["b"].Equal(time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("b mismatch: %v", all["b"])
	}
}

func TestAllNextRunsNil(t *testing.T) {
	t.Parallel()
	var s *Scheduler
	if s.AllNextRuns() != nil {
		t.Error("nil scheduler should return nil")
	}
}

// ── Tasks ────────────────────────────────────────────────────────────────────

func TestTasksNil(t *testing.T) {
	t.Parallel()
	var s *Scheduler
	if s.Tasks() != nil {
		t.Error("nil scheduler should return nil")
	}
}

// ── Results nil guard ────────────────────────────────────────────────────────

func TestResultsNil(t *testing.T) {
	t.Parallel()
	var s *Scheduler
	if s.Results(10) != nil {
		t.Error("nil scheduler should return nil")
	}
}

// ── recomputeAll / recomputeOne ──────────────────────────────────────────────

func TestRecomputeAll(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{
			{Name: "hourly", Cron: "0 * * * *", Prompt: "x"},
			{Name: "daily", Cron: "0 2 * * *", Prompt: "y"},
		},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	s.recomputeAll()

	if len(s.nextRun) != 2 {
		t.Fatalf("expected 2 nextRun entries, got %d", len(s.nextRun))
	}
	_, ok := s.nextRun["hourly"]
	if !ok {
		t.Error("hourly task missing from nextRun")
	}
	_, ok = s.nextRun["daily"]
	if !ok {
		t.Error("daily task missing from nextRun")
	}
}

func TestRecomputeAllSkipsDisabled(t *testing.T) {
	t.Parallel()
	disabled := false
	s := New(Config{
		Tasks: []Task{
			{Name: "enabled", Cron: "0 * * * *", Prompt: "x"},
			{Name: "disabled", Cron: "0 * * * *", Prompt: "y", Enabled: &disabled},
		},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	s.recomputeAll()

	if len(s.nextRun) != 1 {
		t.Fatalf("expected 1 nextRun entry (disabled skipped), got %d", len(s.nextRun))
	}
	if _, ok := s.nextRun["disabled"]; ok {
		t.Error("disabled task should not have a nextRun")
	}
}

func TestRecomputeOneDisablesTask(t *testing.T) {
	t.Parallel()
	disabled := false
	// Create with an enabled task first, then replace with disabled via AddTask
	// to exercise recomputeOne on a disabled task.
	s := New(Config{
		Tasks: []Task{{Name: "test", Cron: "0 * * * *", Prompt: "x"}},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	// Set a nextRun to verify it gets deleted when the task becomes disabled.
	s.nextRun["test"] = time.Now()

	// Replace with disabled version (calls recomputeOne internally).
	s.AddTask(Task{Name: "test", Cron: "0 * * * *", Prompt: "x", Enabled: &disabled})

	if _, ok := s.nextRun["test"]; ok {
		t.Error("nextRun for disabled task should be deleted")
	}
}

func TestRecomputeOneBadCron(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{{Name: "bad", Cron: "invalid", Prompt: "x"}},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}
	s.parentCtx = context.Background()

	// Manually set a nextRun to verify it gets deleted on bad cron.
	s.nextRun["bad"] = time.Now()
	s.recomputeOne(&s.config.Tasks[0])

	if _, ok := s.nextRun["bad"]; ok {
		t.Error("nextRun for task with bad cron should be deleted")
	}
}

// ── Start lifecycle ──────────────────────────────────────────────────────────

func TestStartNilScheduler(t *testing.T) {
	t.Parallel()
	// Must not panic.
	var s *Scheduler
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel
	s.Start(ctx)
}

func TestStartStopsOnCancel(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{{Name: "test", Cron: "0 0 1 1 *", Prompt: "yearly"}},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()

	// Give Start time to enter the loop.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK — Start returned.
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not stop after cancel")
	}
}

// ── parseCronField edge cases ────────────────────────────────────────────────

func TestParseCronFieldStepWithoutRange(t *testing.T) {
	t.Parallel()
	// "5/2" — step expression without a range should error.
	_, err := parseCronField("5/2", 0, 59)
	if err == nil {
		t.Error("expected error for step without range")
	}
}

func TestParseCronFieldInvalidStep(t *testing.T) {
	t.Parallel()
	_, err := parseCronField("*/0", 0, 59)
	if err == nil {
		t.Error("expected error for step 0")
	}
	_, err = parseCronField("*/negative", 0, 59)
	if err == nil {
		t.Error("expected error for non-numeric step")
	}
}

func TestParseCronFieldOutOfRangeValues(t *testing.T) {
	t.Parallel()
	// Values outside the allowed range are accepted by the parser but will
	// never match in NextAfter (since time fields stay within range).
	// This is by design — range enforcement happens implicitly.
	cf, err := parseCronField("99", 0, 59)
	if err != nil {
		t.Fatalf("parser accepts out-of-range values: %v", err)
	}
	if !cf.values[99] {
		t.Error("expected value 99 to be stored")
	}
	// Verify it never matches in-range values.
	for v := 0; v <= 59; v++ {
		if cf.matches(v) {
			t.Errorf("value 99 should not match %d", v)
		}
	}
}

// ── NextAfter exhaustion ─────────────────────────────────────────────────────

func TestNextAfterExhaustsDeadline(t *testing.T) {
	t.Parallel()
	// Feb 31 never occurs; no match within 2 years should error.
	_, err := NextAfter("0 0 31 2 *", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("expected error for impossible cron (Feb 31)")
	}
}

func TestNextAfterExactNow(t *testing.T) {
	t.Parallel()
	// When "now" is exactly the fire time (e.g., 14:00) and cron is "0 14 * * *",
	// next fire should be 14:00 tomorrow, not 14:00 today.
	now := time.Date(2026, 6, 15, 14, 0, 0, 0, time.UTC)
	next, err := NextAfter("0 14 * * *", now)
	if err != nil {
		t.Fatal(err)
	}
	// Should be tomorrow 14:00, not today.
	want := time.Date(2026, 6, 16, 14, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextAfterDOW7AsSunday(t *testing.T) {
	t.Parallel()
	// "0 0 * * 7" — 7 also means Sunday per POSIX cron.
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC) // Saturday
	next, err := NextAfter("0 0 * * 7", now)
	if err != nil {
		t.Fatal(err)
	}
	// Next Sunday is June 14.
	want := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v (Sunday as 7)", next, want)
	}
}

func TestNextAfterDOWSunday(t *testing.T) {
	t.Parallel()
	// "0 0 * * 0" — every Sunday at midnight.
	// June 14 2026 is a Sunday. Test from Saturday June 13.
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC) // Saturday
	next, err := NextAfter("0 0 * * 0", now)
	if err != nil {
		t.Fatal(err)
	}
	// Next Sunday is June 14.
	want := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v (Sunday)", next, want)
	}
}



func TestNextAfterMonthChange(t *testing.T) {
	t.Parallel()
	// "0 0 30 * *" — every 30th of the month.
	// From Jan 31, next should be Feb 28 (or Feb 29 in leap years) then keep scanning.
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	next, err := NextAfter("0 0 30 * *", now)
	if err != nil {
		t.Fatal(err)
	}
	// Feb 2026 has 28 days, March has 30. Should be March 30.
	want := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextAfterYearBoundary(t *testing.T) {
	t.Parallel()
	// "0 0 1 1 *" — Jan 1 every year.
	now := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)
	next, err := NextAfter("0 0 1 1 *", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextAfterCommaDOW(t *testing.T) {
	t.Parallel()
	// Weekdays only (Mon-Fri). June 15 2026 is a Monday.
	// From Monday 00:01, next should be Tuesday.
	now := time.Date(2026, 6, 15, 0, 1, 0, 0, time.UTC) // Monday
	next, err := NextAfter("0 0 * * 1,2,3,4,5", now)
	if err != nil {
		t.Fatal(err)
	}
	// Tuesday June 16.
	want := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

// ── Multiple tasks / unsorted ────────────────────────────────────────────────

func TestTasksReturnsCopy(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{{Name: "original", Cron: "0 * * * *", Prompt: "x"}},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	tasks := s.Tasks()
	tasks[0].Name = "mutated"

	if s.config.Tasks[0].Name != "original" {
		t.Error("Tasks() should return a copy, not reference")
	}
}

// ── New with mix of enabled/disabled ─────────────────────────────────────────

func TestNewWithMixed(t *testing.T) {
	t.Parallel()
	disabled := false
	enabled := true
	cfg := Config{
		Tasks: []Task{
			{Name: "a", Cron: "0 * * * *", Prompt: "x", Enabled: &disabled},
			{Name: "b", Cron: "0 * * * *", Prompt: "y", Enabled: &enabled},
			{Name: "c", Cron: "0 * * * *", Prompt: "z"}, // nil = enabled
		},
	}
	s := New(cfg, &stubSender{}, nil)
	if s == nil {
		t.Fatal("expected non-nil scheduler with at least one enabled task")
	}
	tasks := s.Tasks()
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestIsEnabledPublic(t *testing.T) {
	t.Parallel()
	t1 := Task{Name: "a"} // nil = enabled
	if !t1.IsEnabled() {
		t.Error("nil Enabled should be enabled via public IsEnabled")
	}
	e := true
	t2 := Task{Name: "b", Enabled: &e}
	if !t2.IsEnabled() {
		t.Error("Enabled=true should be enabled via public IsEnabled")
	}
	d := false
	t3 := Task{Name: "c", Enabled: &d}
	if t3.IsEnabled() {
		t.Error("Enabled=false should be disabled via public IsEnabled")
	}
}

func TestRecomputeAllBadCron(t *testing.T) {
	t.Parallel()
	s := New(Config{
		Tasks: []Task{
			{Name: "good", Cron: "0 * * * *", Prompt: "x"},
			{Name: "bad", Cron: "nope", Prompt: "y"},
		},
	}, &stubSender{}, nil)
	if s == nil {
		t.Fatal("nil scheduler")
	}

	s.recomputeAll()

	// Good task should have a nextRun.
	if _, ok := s.nextRun["good"]; !ok {
		t.Error("good task should have a nextRun")
	}
	// Bad task should be skipped.
	if _, ok := s.nextRun["bad"]; ok {
		t.Error("bad cron task should not have a nextRun")
	}
}

func TestParseCronFieldRangeEndError(t *testing.T) {
	t.Parallel()
	_, err := parseCronField("1-abc", 0, 59)
	if err == nil {
		t.Error("expected error for invalid range end")
	}
}

func TestParseCronFieldStepRangeStartError(t *testing.T) {
	t.Parallel()
	_, err := parseCronField("abc-10/2", 0, 59)
	if err == nil {
		t.Error("expected error for invalid step range start")
	}
}

func TestParseCronFieldInvalidValue(t *testing.T) {
	t.Parallel()
	_, err := parseCronField("abc", 0, 59)
	if err == nil {
		t.Error("expected error for non-numeric value")
	}
}

// ── cronField.matches direct ─────────────────────────────────────────────────

func TestCronFieldMatchesStar(t *testing.T) {
	t.Parallel()
	f := cronField{star: true}
	for v := 0; v <= 59; v++ {
		if !f.matches(v) {
			t.Errorf("star should match %d", v)
		}
	}
}

func TestCronFieldMatchesExact(t *testing.T) {
	t.Parallel()
	f := cronField{values: map[int]bool{5: true, 10: true}}
	if !f.matches(5) {
		t.Error("should match 5")
	}
	if !f.matches(10) {
		t.Error("should match 10")
	}
	if f.matches(7) {
		t.Error("should NOT match 7")
	}
}
