package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"reasonix/internal/i18n"
	"reasonix/internal/publish"
)

// showMemory reports what memory is loaded and where it lives — the TUI analog
// of Claude Code's /memory. It surfaces the doc files and the auto-memory store
// path so the user can open and edit them directly, since the in-terminal UI
// doesn't shell out to an editor.
func (m *chatTUI) showMemory() {
	set := m.ctrl.Memory()
	if set == nil || (set.Empty() && len(set.Store.ListArchived()) == 0) {
		m.notice(i18n.M.MemoryNone)
		return
	}
	m.commitLine(renderMemory(m.width, set))
}

// forgetMemory deletes a saved auto-memory by name (the slug shown in /memory).
// It is the manual counterpart to the model's `forget` tool.
func (m *chatTUI) forgetMemory(name string) {
	if name == "" {
		m.notice(i18n.M.ForgetUsage)
		return
	}
	if err := m.ctrl.ForgetMemory(name); err != nil {
		m.notice(fmt.Sprintf("forget: %v", err))
		return
	}
	m.notice(fmt.Sprintf(i18n.M.ForgetDoneFmt, name))
}

// publishSession exports the current session as an HTML file and opens it.
func (m *chatTUI) publishSession() {
	msgs := m.ctrl.History()
	if len(msgs) == 0 {
		m.notice("nothing to publish — session is empty")
		return
	}
	s := publish.Session{
		Title:     "Reasonix Session",
		Model:     m.label,
		Date:      m.sessionStart,
		Messages:  msgs,
		TokensIn:  m.ctrl.SessionTokensIn(),
		TokensOut: m.ctrl.SessionTokensOut(),
		Turns:     m.ctrl.SessionTurns(),
		Cost:      m.ctrl.SessionCost(),
	}
	html := publish.ToHTML(s)

	outDir := filepath.Join(m.ctrl.SessionDir(), "published")
	_ = os.MkdirAll(outDir, 0o755)
	fname := fmt.Sprintf("session-%s.html", time.Now().Format("2006-01-02-150405"))
	outPath := filepath.Join(outDir, fname)
	if err := os.WriteFile(outPath, []byte(html), 0o644); err != nil {
		m.notice("publish: " + err.Error())
		return
	}
	m.notice("published → " + outPath)
}
