//go:build darwin

package notify

import (
	"os/exec"
	"strings"
	"time"
)

// PlatformSender delivers notifications through the host OS.
type PlatformSender struct{}

// NewPlatformSender returns the best-effort sender for the current platform.
func NewPlatformSender() PlatformSender { return PlatformSender{} }

func (PlatformSender) Send(m Message) error {
	script := `display notification "` + appleScriptString(m.Body) + `" with title "` + appleScriptString(m.Title) + `"`
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}()
	return nil
}

func appleScriptString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}
