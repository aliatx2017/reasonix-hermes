// Package control is the transport-agnostic session driver.
package control

import (
	"context"
	"fmt"
	"strings"

	"reasonix/internal/learn"
	"reasonix/internal/mesh"
)

// --- Mesh / Council ---

// SetMesh stores the mesh instance for council operations. nil disables mesh.
func (c *Controller) SetMesh(m *mesh.Mesh) {
	c.mesh = m
}

// SetLearner attaches a self-improving learner for pattern detection.
func (c *Controller) SetLearner(l *learn.Learner) {
	c.learner = l
}

// MeshPeers returns the list of configured peer names for the mesh council.
func (c *Controller) MeshPeers() []string {
	if c.mesh == nil {
		return nil
	}
	return c.mesh.Peers()
}

// Mesh returns the mesh instance (may be nil) — internal use only (ConfineCouncil wiring).
func (c *Controller) Mesh() *mesh.Mesh {
	return c.mesh
}

// Learner returns the self-improving learner instance (may be nil).
func (c *Controller) Learner() *learn.Learner {
	return c.learner
}

// Council dispatches a task to all mesh peers and returns the consensus.
// Returns an error when mesh is disabled or has no peers.
func (c *Controller) Council(ctx context.Context, task string) (string, error) {
	if c.mesh == nil {
		return "", fmt.Errorf("mesh is not enabled — configure [mesh] in reasonix.toml")
	}
	council := mesh.NewCouncil(c.mesh)
	if err := council.Convene(ctx, task); err != nil {
		return "", err
	}
	return council.Consensus(), nil
}

// MeshStatus returns a human-readable status string for the mesh.
func (c *Controller) MeshStatus() string {
	if c.mesh == nil {
		return "mesh: disabled"
	}
	peerNames := c.mesh.Peers()
	return fmt.Sprintf("mesh: %d peer(s) — %s", len(peerNames), strings.Join(peerNames, ", "))
}
