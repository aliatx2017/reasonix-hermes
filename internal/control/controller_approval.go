// Package control is the transport-agnostic session driver.
package control

import (
	"fmt"

	"reasonix/internal/config"
)









// ToggleSound turns the completion-sound setting on or off and persists the change
// to the user config file. Returns true on success (setting changed and saved).
func (c *Controller) ToggleSound(on bool) bool {
	cfg, err := config.Load()
	if err != nil {
		c.notice("sound: " + err.Error())
		return false
	}
	cfg.Notifications.Sound = on
	if err := cfg.Save(); err != nil {
		c.notice("sound: save failed: " + err.Error())
		return false
	}
	if on {
		c.notice("completion sound on — bell chimes when each turn finishes")
	} else {
		c.notice("completion sound off")
	}
	return true
}

// beep writes an ASCII bell character to stdout when the completion-sound config
// flag is enabled. Called after each turn completes.
func (c *Controller) beep() {
	cfg, err := config.Load()
	if err != nil {
		return
	}
	if cfg.Notifications.Sound {
		fmt.Print("\a")
	}
}
