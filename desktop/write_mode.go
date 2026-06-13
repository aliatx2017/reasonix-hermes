package main

import (
	"os"
	"path/filepath"
	"strings"
)

// --- Write Mode: Markdown Workspace ---

// MarkdownFileEntry is one file in the workspace for the Write Mode browser.
type MarkdownFileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	RelPath string `json:"relPath"`
	Size    int64  `json:"size"`
}

// MarkdownContent is the full content of a markdown file.
type MarkdownContent struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ListMarkdownFiles returns all .md files under the project workspace.
func (a *App) ListMarkdownFiles() []MarkdownFileEntry {
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return nil
	}
	root := ctrl.WorkspaceRoot()
	if root == "" {
		return nil
	}

	var entries []MarkdownFileEntry
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		entries = append(entries, MarkdownFileEntry{
			Name:    info.Name(),
			Path:    path,
			RelPath: rel,
			Size:    info.Size(),
		})
		return nil
	})
	return entries
}

// ReadMarkdownFile reads a markdown file by relative path and returns its content.
func (a *App) ReadMarkdownFile(relPath string) *MarkdownContent {
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return nil
	}
	root := ctrl.WorkspaceRoot()
	if root == "" {
		return nil
	}
	fullPath := filepath.Join(root, relPath)
	// Safety: ensure the resolved path is still under root.
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return nil
	}
	if !strings.HasPrefix(resolved, root) {
		return nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil
	}
	return &MarkdownContent{
		Path:    relPath,
		Name:    filepath.Base(relPath),
		Content: string(data),
	}
}

// SaveMarkdownFile writes content to a markdown file by relative path.
func (a *App) SaveMarkdownFile(relPath, content string) error {
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return nil
	}
	root := ctrl.WorkspaceRoot()
	if root == "" {
		return nil
	}
	fullPath := filepath.Join(root, relPath)
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		// File doesn't exist yet — ensure it's under root.
		abs, aerr := filepath.Abs(fullPath)
		if aerr != nil {
			return aerr
		}
		if !strings.HasPrefix(abs, root) {
			return os.ErrPermission
		}
		// Create parent directories.
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		return os.WriteFile(abs, []byte(content), 0o644)
	}
	if !strings.HasPrefix(resolved, root) {
		return os.ErrPermission
	}
	return os.WriteFile(resolved, []byte(content), 0o644)
}

// CreateMarkdownFile creates a new .md file with optional initial content.
// Returns the relative path on success.
func (a *App) CreateMarkdownFile(relPath, content string) (string, error) {
	if !strings.HasSuffix(strings.ToLower(relPath), ".md") {
		relPath += ".md"
	}
	ctrl := a.ctrlForTab("")
	if ctrl == nil {
		return "", nil
	}
	root := ctrl.WorkspaceRoot()
	if root == "" {
		return "", nil
	}
	fullPath := filepath.Join(root, relPath)
	abs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, root) {
		return "", os.ErrPermission
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return "", err
	}
	return relPath, nil
}
