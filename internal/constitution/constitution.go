// Package constitution loads structured project invariants from
// .reasonix/constitution.json and formats them as a "Project Constitution"
// block for the system prompt. It is layered on top of the REASONIX.md memory
// files — the constitution declares what is OK and what is not in a structured
// way the model can reason about more precisely than free-form prose.
package constitution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// File is the constitution file scanned at the project root.
const File = "constitution.json"

// Dir is the directory containing the constitution file.
const Dir = ".reasonix"

// Doc is the parsed constitution document.
type Doc struct {
	Version     int            `json:"version"`
	Conventions map[string]any  `json:"conventions,omitempty"` // key-value pairs like "language":"Go"
	Rules       []Rule          `json:"rules,omitempty"`
	Principles  []string        `json:"principles,omitempty"`   // high-level design principles
	Constraints []string        `json:"constraints,omitempty"`  // hard constraints (must/must-not)
}

// Rule is one code-level invariant.
type Rule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Severity    string `json:"severity,omitempty"` // error | warning | info
	Scope       string `json:"scope,omitempty"`    // e.g. "*.go", "internal/**"
}

// Load reads and parses .reasonix/constitution.json under cwd. Returns a
// zero Doc with ok=false when the file is absent or unparseable — never an
// error; a malformed constitution should not prevent the agent from starting.
func Load(cwd string) (Doc, bool) {
	path := filepath.Join(cwd, Dir, File)
	data, err := os.ReadFile(path)
	if err != nil {
		return Doc{}, false
	}
	var d Doc
	if err := json.Unmarshal(data, &d); err != nil {
		return Doc{}, false
	}
	if d.Version == 0 {
		d.Version = 1
	}
	return d, true
}

// Format produces the system-prompt block for a constitution. Returns ""
// when the doc carries nothing to inject.
func Format(d Doc) string {
	var b strings.Builder
	empty := true

	if len(d.Conventions) > 0 {
		b.WriteString("## Project Conventions\n\n")
		keys := sortedKeys(d.Conventions)
		for _, k := range keys {
			v := d.Conventions[k]
			b.WriteString("- **")
			b.WriteString(capitalize(k))
			b.WriteString("**: ")
			b.WriteString(formatValue(v))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		empty = false
	}

	if len(d.Principles) > 0 {
		b.WriteString("## Design Principles\n\n")
		for _, p := range d.Principles {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(p))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		empty = false
	}

	if len(d.Constraints) > 0 {
		b.WriteString("## Hard Constraints\n\n")
		for _, c := range d.Constraints {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(c))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		empty = false
	}

	if len(d.Rules) > 0 {
		b.WriteString("## Code Rules\n\n")
		for _, r := range d.Rules {
			sev := r.Severity
			if sev == "" {
				sev = "info"
			}
			b.WriteString("- **[")
			b.WriteString(strings.ToUpper(sev))
			b.WriteString("]** ")
			if r.ID != "" {
				b.WriteString("`")
				b.WriteString(r.ID)
				b.WriteString("`: ")
			}
			b.WriteString(strings.TrimSpace(r.Description))
			if r.Scope != "" {
				b.WriteString(" (scope: ")
				b.WriteString(r.Scope)
				b.WriteString(")")
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		empty = false
	}

	if empty {
		return ""
	}

	s := "# Project Constitution\n\n"
	s += b.String()
	return strings.TrimRight(s, "\n")
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
	}
	return string(r)
}

func formatValue(v any) string {
	switch vv := v.(type) {
	case string:
		return vv
	case []any:
		parts := make([]string, len(vv))
		for i, item := range vv {
			parts[i] = formatValue(item)
		}
		return strings.Join(parts, ", ")
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
