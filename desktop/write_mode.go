package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// FIMResult is the response from a Fill-in-the-Middle completion request.
type FIMResult struct {
	Text string `json:"text"`
}

// FIMComplete requests a Fill-in-the-Middle completion for the given file content
// at the cursor position. prefix = content before cursor, suffix = content after.
func (a *App) FIMComplete(relPath string, cursorPos int) *FIMResult {
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
		return nil
	}
	if !strings.HasPrefix(resolved, root) {
		return nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil
	}
	content := string(data)
	if cursorPos < 0 {
		cursorPos = 0
	}
	if cursorPos > len(content) {
		cursorPos = len(content)
	}
	prefix := content[:cursorPos]
	suffix := content[cursorPos:]

	// Read API config from loaded user config.
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	model := os.Getenv("DEEPSEEK_FIM_MODEL")
	if model == "" {
		model = "deepseek-chat" // FIM-compatible model
	}
	if apiKey == "" {
		return nil
	}
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	completion, err := fimRequest(baseURL, apiKey, model, prefix, suffix)
	if err != nil {
		return nil
	}
	return &FIMResult{Text: completion}
}

// fimRequest calls the DeepSeek API FIM completions endpoint.
func fimRequest(baseURL, apiKey, model, prefix, suffix string) (string, error) {
	body := map[string]any{
		"model":  model,
		"prompt": prefix,
		"suffix": suffix,
		"max_tokens": 256,
		"temperature": 0.0,
		"stop": []string{"\n\n\n"},
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", strings.TrimRight(baseURL, "/")+"/v1/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", nil
	}
	return result.Choices[0].Text, nil
}
