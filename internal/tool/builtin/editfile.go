package builtin

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"reasonix/internal/tool"
)

func init() { tool.RegisterBuiltin(editFile{}) }

// editFile replaces an exact string in a file. roots confines the target to the
// workspace when non-empty (see writeFile); workDir, when non-empty, is the
// directory a relative path resolves against (see resolveIn).
type editFile struct {
	roots   []string
	workDir string
}

func (editFile) Name() string { return "edit_file" }

func (editFile) Description() string {
	return "Replace an exact string in a file with another. old_string must occur exactly once; add surrounding context to disambiguate. Use for targeted edits instead of rewriting the whole file."
}

func (editFile) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path"},"old_string":{"type":"string","description":"Exact text to replace (must be unique in the file)"},"new_string":{"type":"string","description":"Replacement text (may be empty to delete)"},"content_hash":{"type":"string","description":"Optional SHA-256 hash of the file content from read_file output. If provided, the edit is rejected if the file content has changed since the hash was computed — preventing stale-context edits."}},"required":["path","old_string","new_string"]}`)
}

func (editFile) ReadOnly() bool { return false }

func (e editFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path        string `json:"path"`
		OldString   string `json:"old_string"`
		NewString   string `json:"new_string"`
		ContentHash string `json:"content_hash"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if p.OldString == "" {
		return "", fmt.Errorf("old_string is required")
	}
	p.Path = resolveIn(e.workDir, p.Path)
	if err := confine(e.roots, p.Path); err != nil {
		return "", err
	}

	content, enc, err := readFileEncoded(p.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", p.Path, err)
	}

<<<<<<< HEAD
	// Hash-anchored edit: if the caller provided a content_hash (from a prior
	// read_file call), verify the file hasn't changed since. This catches the
	// race where another process — or another agent — modified the file between
	// the read and the edit, which would make the old_string match unreliable.
	if p.ContentHash != "" {
		actual := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
		if actual != p.ContentHash {
			return "", fmt.Errorf("file %s changed since content_hash was computed (read=%s, current=%s) — re-read the file and try again", p.Path, p.ContentHash[:12], actual[:12])
		}
	}

	old, newStr := matchLineEndings(content, p.OldString, p.NewString)
	switch strings.Count(content, old) {
	case 0:
		return "", fmt.Errorf("old_string not found in %s", p.Path)
	case 1:
=======
	applied := applyOldStringEdit(content, p.OldString, p.NewString, false)
	switch {
	case applied.applied == 1:
>>>>>>> upstream/main-v2
		// ok
	case applied.matches == 0:
		return "", oldStringNotFoundError(p.Path, p.OldString, content)
	default:
		return "", oldStringNotUniqueError(p.Path, applied.matches, false)
	}

	if err := writeFileEncoded(p.Path, applied.updated, enc); err != nil {
		return "", fmt.Errorf("write %s: %w", p.Path, err)
	}
	if applied.fuzzy {
		return fmt.Sprintf("edited %s (fuzzy match)", p.Path), nil
	}
	return fmt.Sprintf("edited %s", p.Path), nil
}
