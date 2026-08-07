package marketplace

import (
	"testing"
)

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()
	r := DefaultRegistry()
	if r.Len() == 0 {
		t.Fatal("default registry is empty")
	}
	t.Logf("%d skills in default registry", r.Len())
}

func TestSearch(t *testing.T) {
	t.Parallel()
	r := DefaultRegistry()

	tests := []struct {
		query   string
		minHits int
	}{
		{"go", 2},
		{"python", 3},
		{"testing", 4},
		{"nonexistent", 0},
		{"", r.Len()}, // empty query returns all
	}
	for _, tt := range tests {
		results := r.Search(tt.query)
		if len(results) < tt.minHits {
			t.Errorf("Search(%q): got %d results, want at least %d", tt.query, len(results), tt.minHits)
		}
		for _, e := range results {
			if e.Name == "" || e.Description == "" {
				t.Errorf("Search(%q): entry has empty name or description: %+v", tt.query, e)
			}
		}
	}
}

func TestByName(t *testing.T) {
	t.Parallel()
	r := DefaultRegistry()
	e := r.ByName("golang-patterns")
	if e == nil {
		t.Fatal("golang-patterns not found")
	}
	if e.Author == "" {
		t.Error("author should not be empty")
	}
	if len(e.Tags) == 0 {
		t.Error("tags should not be empty")
	}

	if r.ByName("nonexistent-skill-xyz") != nil {
		t.Error("nonexistent skill should return nil")
	}
}

func TestTags(t *testing.T) {
	t.Parallel()
	r := DefaultRegistry()
	tags := r.Tags()
	if len(tags) == 0 {
		t.Fatal("no tags found")
	}
	t.Logf("tags: %v", tags)
}

func TestByTag(t *testing.T) {
	t.Parallel()
	r := DefaultRegistry()
	results := r.ByTag("go")
	if len(results) < 1 {
		t.Errorf("ByTag(go): got %d results, want at least 1", len(results))
	}
	for _, e := range results {
		found := false
		for _, t := range e.Tags {
			if t == "go" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ByTag(go): entry %q doesn't have tag 'go'", e.Name)
		}
	}
}

func TestListSorted(t *testing.T) {
	t.Parallel()
	r := DefaultRegistry()
	entries := r.List()
	if len(entries) == 0 {
		t.Fatal("list is empty")
	}
	// Verify descending rating order.
	for i := 1; i < len(entries); i++ {
		if entries[i].Rating > entries[i-1].Rating {
			t.Errorf("List: item %d (%s, %.1f) > item %d (%s, %.1f) — not sorted",
				i, entries[i].Name, entries[i].Rating, i-1, entries[i-1].Name, entries[i-1].Rating)
		}
	}
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()
	data := []byte(`[{"name":"test-skill","description":"A test skill","url":"https://example.com","author":"test","tags":["test"],"rating":5.0}]`)
	r, err := NewRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.Len() != 1 {
		t.Errorf("expected 1 entry, got %d", r.Len())
	}
	e := r.ByName("test-skill")
	if e == nil {
		t.Fatal("test-skill not found")
	}
	if e.Rating != 5.0 {
		t.Errorf("rating = %f, want 5.0", e.Rating)
	}
}

func TestNewRegistryInvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := NewRegistry([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
