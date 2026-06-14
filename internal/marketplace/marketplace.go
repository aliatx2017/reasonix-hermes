// Package marketplace provides a community skill registry — search, browse,
// and install skills from a curated index. Uses the agentskills.io-compatible
// SKILL.md format with YAML frontmatter.
//
// The registry is a JSON file listing community skills with metadata
// (name, description, author, tags, rating, install URL). The default
// registry ships with the reasonix-hermes repo at skills-hub/registry.json.
package marketplace

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
)

// Entry is one community skill in the registry.
type Entry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Rating      float64  `json:"rating"`
}

//go:embed registry.json
var defaultRegistry []byte

// Registry is a searchable skill index.
type Registry struct {
	entries []Entry
}

// DefaultRegistry loads the embedded default skill registry.
func DefaultRegistry() *Registry {
	var entries []Entry
	if err := json.Unmarshal(defaultRegistry, &entries); err != nil {
		return &Registry{}
	}
	return &Registry{entries: entries}
}

// NewRegistry loads a custom registry from JSON bytes.
func NewRegistry(data []byte) (*Registry, error) {
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return &Registry{entries: entries}, nil
}

// List returns all entries, sorted by rating (highest first).
func (r *Registry) List() []Entry {
	sorted := make([]Entry, len(r.entries))
	copy(sorted, r.entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Rating > sorted[j].Rating })
	return sorted
}

// Search finds entries whose name, description, or tags match the query.
// Case-insensitive substring match. Returns results sorted by rating.
func (r *Registry) Search(query string) []Entry {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return r.List()
	}
	var results []Entry
	for _, e := range r.entries {
		if match := matchEntry(e, query); match {
			results = append(results, e)
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Rating > results[j].Rating })
	return results
}

// ByName returns the entry with the given name, or nil.
func (r *Registry) ByName(name string) *Entry {
	name = strings.TrimSpace(name)
	for i := range r.entries {
		if strings.EqualFold(r.entries[i].Name, name) {
			return &r.entries[i]
		}
	}
	return nil
}

// Tags returns all unique tags across the registry, sorted alphabetically.
func (r *Registry) Tags() []string {
	seen := map[string]bool{}
	for _, e := range r.entries {
		for _, t := range e.Tags {
			seen[t] = true
		}
	}
	var tags []string
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// ByTag returns entries matching a specific tag.
func (r *Registry) ByTag(tag string) []Entry {
	tag = strings.ToLower(strings.TrimSpace(tag))
	var results []Entry
	for _, e := range r.entries {
		for _, t := range e.Tags {
			if strings.EqualFold(t, tag) {
				results = append(results, e)
				break
			}
		}
	}
	return results
}

// Len returns the number of entries in the registry.
func (r *Registry) Len() int {
	return len(r.entries)
}

func matchEntry(e Entry, query string) bool {
	if strings.Contains(strings.ToLower(e.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(e.Description), query) {
		return true
	}
	for _, t := range e.Tags {
		if strings.Contains(strings.ToLower(t), query) {
			return true
		}
	}
	return false
}
