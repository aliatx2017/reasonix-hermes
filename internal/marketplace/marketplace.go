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
	"fmt"
	"sort"
	"strings"
	"sync"
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
	mu      sync.RWMutex
	entries []Entry
}

var (
	defaultReg     *Registry
	defaultRegOnce sync.Once
)

// DefaultRegistry loads the embedded default skill registry.
// The result is cached — all callers share the same instance,
// so MergeFromLobeHub additions are visible to subsequent callers.
func DefaultRegistry() *Registry {
	defaultRegOnce.Do(func() {
		var entries []Entry
		if err := json.Unmarshal(defaultRegistry, &entries); err != nil {
			defaultReg = &Registry{}
			return
		}
		defaultReg = &Registry{entries: entries}
	})
	return defaultReg
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
	r.mu.RLock()
	defer r.mu.RUnlock()
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
	r.mu.RLock()
	defer r.mu.RUnlock()
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
	r.mu.RLock()
	defer r.mu.RUnlock()
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
	r.mu.RLock()
	defer r.mu.RUnlock()
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
	r.mu.RLock()
	defer r.mu.RUnlock()
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// MergeFromLobeHub merges LobeHub marketplace skills into the registry.
// Duplicates (matched by name, case-insensitive) are skipped; new entries
// are appended. Returns the count of newly added skills.
func (r *Registry) MergeFromLobeHub(agents []LobeHubAgentItem) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Build a set of existing names for O(1) dedup.
	seen := make(map[string]bool, len(r.entries))
	for _, e := range r.entries {
		seen[strings.ToLower(e.Name)] = true
	}

	added := 0
	for _, s := range agents {
		entry := s.ToEntry()
		if entry.Name == "" {
			continue
		}
		// Skip entries that are predominantly Chinese (CJK).
		if isCJKHeavy(entry.Name) || isCJKHeavy(entry.Description) {
			continue
		}
		if seen[strings.ToLower(entry.Name)] {
			continue
		}
		seen[strings.ToLower(entry.Name)] = true
		r.entries = append(r.entries, entry)
		added++
	}
	return added
}

// SyncFromLobeHub fetches agents from the LobeHub marketplace and merges
// them into the registry. It uses the provided client (which should already
// be registered). query, sort, and category may be empty to fetch all.
// Returns the total number of agents fetched and the count of newly added entries.
func (r *Registry) SyncFromLobeHub(client *LobeHubClient, query, sort, category string) (fetched int, added int, err error) {
	agents, err := client.FetchAllAgents(20, query, sort, "desc", category)
	if err != nil {
		return 0, 0, fmt.Errorf("lobehub sync: %w", err)
	}
	added = r.MergeFromLobeHub(agents)
	return len(agents), added, nil
}

// isCJKHeavy returns true if >30% of the runes in s are CJK characters.
func isCJKHeavy(s string) bool {
	if len(s) == 0 {
		return false
	}
	cjk := 0
	for _, r := range s {
		if (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) || (r >= 0xF900 && r <= 0xFAFF) {
			cjk++
		}
	}
	return float64(cjk)/float64(len([]rune(s))) > 0.3
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
