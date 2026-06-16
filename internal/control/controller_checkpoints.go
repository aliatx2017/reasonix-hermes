// Package control is the transport-agnostic session driver.
package control

import "reasonix/internal/checkpoint"

// CheckpointFileSnaps returns file snapshots for a specific turn, or nil.
func (c *Controller) CheckpointFileSnaps(turn int) []checkpoint.FileSnap {
	if c.cp == nil {
		return nil
	}
	return c.cp.FileSnaps(turn)
}
