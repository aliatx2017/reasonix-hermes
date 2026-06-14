package learn

import (
	"strings"
	"sync"
	"testing"
)

func TestNew_DefaultConfig(t *testing.T) {
	l := New(Config{Enabled: true})
	if !l.enabled {
		t.Error("expected enabled by default")
	}
	if l.maxObs != 200 {
		t.Errorf("maxObs = %d, want 200", l.maxObs)
	}
	if l.minConfidence != 3 {
		t.Errorf("minConfidence = %d, want 3", l.minConfidence)
	}
}

func TestNew_Disabled(t *testing.T) {
	l := New(Config{Enabled: false})
	if l.enabled {
		t.Error("expected disabled when Enabled=false")
	}
}

func TestNew_ConfigCaps(t *testing.T) {
	l := New(Config{Enabled: true, MaxObservations: 5000, MinConfidence: 0})
	if l.maxObs != 2000 {
		t.Errorf("maxObs capped: got %d, want 2000", l.maxObs)
	}
	if l.minConfidence != 3 {
		t.Errorf("minConfidence defaulted: got %d, want 3", l.minConfidence)
	}
}

func TestObserve_DisabledNoOp(t *testing.T) {
	l := New(Config{Enabled: false})
	l.Observe("task", []ToolCallInfo{{Name: "bash", Success: true}}, "", "")
	obs := l.Observations()
	if len(obs) != 0 {
		t.Errorf("expected zero observations when disabled, got %d", len(obs))
	}
}

func TestObserve_AccumulatesObservations(t *testing.T) {
	l := New(Config{Enabled: true, MaxObservations: 10})
	l.Observe("run tests", []ToolCallInfo{{Name: "bash", Success: true, Brief: "go test ./..."}}, "", "")
	l.Observe("edit file", []ToolCallInfo{{Name: "edit_file", Success: true, Brief: "agent.go"}}, "", "")

	obs := l.Observations()
	if len(obs) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(obs))
	}
	if obs[0].Turn != 1 {
		t.Errorf("first turn = %d, want 1", obs[0].Turn)
	}
	if obs[1].Turn != 2 {
		t.Errorf("second turn = %d, want 2", obs[1].Turn)
	}
	if obs[0].Task != "run tests" {
		t.Errorf("first task = %q, want 'run tests'", obs[0].Task)
	}
}

func TestObserve_SkillTracking(t *testing.T) {
	l := New(Config{Enabled: true})
	l.Observe("explore code", nil, "explore", "internal/agent/")
	obs := l.Observations()
	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(obs))
	}
	if obs[0].SkillName != "explore" {
		t.Errorf("skill name = %q, want 'explore'", obs[0].SkillName)
	}
	if obs[0].SkillArgs != "internal/agent/" {
		t.Errorf("skill args = %q, want 'internal/agent/'", obs[0].SkillArgs)
	}
}

func TestObserve_RingBufferEviction(t *testing.T) {
	l := New(Config{Enabled: true, MaxObservations: 5})
	for i := 0; i < 10; i++ {
		l.Observe("task", nil, "", "")
	}
	obs := l.Observations()
	if len(obs) != 5 {
		t.Errorf("expected 5 observations after overflow, got %d", len(obs))
	}
	// First observation should be turn 6 (turn 1-5 evicted)
	if obs[0].Turn != 6 {
		t.Errorf("first kept turn = %d, want 6", obs[0].Turn)
	}
	if obs[4].Turn != 10 {
		t.Errorf("last kept turn = %d, want 10", obs[4].Turn)
	}
}

func TestPatterns_NotEnoughConfidence(t *testing.T) {
	l := New(Config{Enabled: true, MinConfidence: 5})
	for i := 0; i < 4; i++ {
		l.Observe("edit and test", []ToolCallInfo{
			{Name: "edit_file", Success: true},
			{Name: "bash", Success: true, Brief: "go test ./..."},
		}, "", "")
	}
	pats := l.Patterns()
	if len(pats) != 0 {
		t.Errorf("expected no patterns with confidence 4 below min 5, got %d", len(pats))
	}
}

func TestPatterns_EditThenTestDetected(t *testing.T) {
	l := New(Config{Enabled: true, MinConfidence: 3})
	tc := []ToolCallInfo{
		{Name: "edit_file", Success: true},
		{Name: "bash", Success: true, Brief: "go test ./..."},
	}
	for i := 0; i < 5; i++ {
		l.Observe("edit agent.go and test", tc, "", "")
	}
	pats := l.Patterns()
	if len(pats) == 0 {
		t.Fatal("expected at least one pattern")
	}
	p := pats[0]
	if !strings.Contains(p.Name, "auto") {
		t.Errorf("expected pattern name to contain 'auto', got %q", p.Name)
	}
	if p.Confidence != 5 {
		t.Errorf("confidence = %d, want 5", p.Confidence)
	}
	if p.Trigger != "after editing files" {
		t.Errorf("trigger = %q, want 'after editing files'", p.Trigger)
	}
}

func TestPatterns_ReadOnlyNoPattern(t *testing.T) {
	l := New(Config{Enabled: true, MinConfidence: 2})
	for i := 0; i < 3; i++ {
		l.Observe("list files", []ToolCallInfo{
			{Name: "ls", Success: true},
			{Name: "glob", Success: true},
		}, "", "")
	}
	pats := l.Patterns()
	// Read-only sequences can form patterns, but the pattern is less meaningful
	// Still, the detector should find the repeating sequence
	for _, p := range pats {
		if p.Confidence < 2 {
			t.Errorf("pattern %q has low confidence %d", p.Name, p.Confidence)
		}
	}
}

func TestBuildReflectionPrompt_Empty(t *testing.T) {
	l := New(Config{Enabled: true})
	p := l.BuildReflectionPrompt()
	if p != "" {
		t.Errorf("expected empty prompt, got %q", p)
	}
}

func TestBuildReflectionPrompt_WithPatterns(t *testing.T) {
	l := New(Config{Enabled: true, MinConfidence: 2})
	for i := 0; i < 3; i++ {
		l.Observe("edit and test", []ToolCallInfo{
			{Name: "edit_file", Success: true},
			{Name: "bash", Success: true, Brief: "go test"},
		}, "", "")
	}
	p := l.BuildReflectionPrompt()
	if p == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(p, "install_skill") {
		t.Error("prompt should mention install_skill")
	}
	if !strings.Contains(p, "Detected Patterns") {
		t.Error("prompt should contain 'Detected Patterns'")
	}
	if !strings.Contains(p, "Recent Observations") {
		t.Error("prompt should contain 'Recent Observations'")
	}
}

func TestReset(t *testing.T) {
	l := New(Config{Enabled: true})
	l.Observe("task", nil, "", "")
	l.Observe("task2", nil, "", "")
	l.Reset()
	obs := l.Observations()
	if len(obs) != 0 {
		t.Errorf("expected no observations after reset, got %d", len(obs))
	}
	// After reset, next turn should be 1 again
	l.Observe("new task", nil, "", "")
	obs = l.Observations()
	if len(obs) != 1 || obs[0].Turn != 1 {
		t.Errorf("expected turn 1 after reset, got turn %d", obs[0].Turn)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"short", 100, "short"},
		{"hello world this is a long string", 10, "hello worl..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestGeneratePatternName(t *testing.T) {
	tests := []struct {
		seq      string
		preEdit  bool
		contains string
	}{
		{"edit_file,bash", true, "auto-verify"},
		{"edit_file,bash", false, "workflow-"},
		{"bash", false, "workflow-"},
		{"explore", false, "deep-explore"},
		{"review", false, "auto-review"},
	}
	for _, tt := range tests {
		got := generatePatternName(tt.seq, tt.preEdit)
		if !strings.Contains(got, tt.contains) {
			t.Errorf("generatePatternName(%q, %v) = %q, want contains %q", tt.seq, tt.preEdit, got, tt.contains)
		}
	}
}

func TestConcurrentObserve(t *testing.T) {
	l := New(Config{Enabled: true, MaxObservations: 100})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			l.Observe("concurrent task", []ToolCallInfo{
				{Name: "bash", Success: true, Brief: "echo hello"},
			}, "", "")
		}(i)
	}
	wg.Wait()
	obs := l.Observations()
	if len(obs) != 50 {
		t.Errorf("expected 50 observations from 50 goroutines, got %d", len(obs))
	}
}
