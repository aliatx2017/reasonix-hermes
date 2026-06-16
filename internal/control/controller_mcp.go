// Package control is the transport-agnostic session driver.
package control

import (
	"fmt"
	"os"
	"strings"

	"reasonix/internal/codegraph"
	"reasonix/internal/config"
)











// ConnectCodegraphMCPServer connects the built-in CodeGraph server using an
// already-resolved config. Desktop uses this after saving user-level settings so
// a stale project config cannot override the just-applied choice.
func (c *Controller) ConnectCodegraphMCPServer(cfg *config.Config) (int, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	return c.ConnectCodegraphMCPServerForRoot(cfg, cwd)
}

// ConnectCodegraphMCPServerForRoot connects CodeGraph pinned to root. Desktop
// project tabs use this after hot updates so a reconnect keeps the same project
// scope as boot-time CodeGraph startup.
func (c *Controller) ConnectCodegraphMCPServerForRoot(cfg *config.Config, root string) (int, error) {
	return c.connectCodegraphMCPServerForRoot(cfg, root)
}

func (c *Controller) connectCodegraphMCPServer(cfg *config.Config) (int, error) {
	return c.ConnectCodegraphMCPServer(cfg)
}

func (c *Controller) connectCodegraphMCPServerForRoot(cfg *config.Config, root string) (int, error) {
	if !cfg.Codegraph.Enabled {
		return 0, fmt.Errorf("codegraph is disabled in config")
	}
	bin, ok := codegraph.Resolve(cfg.Codegraph.Path)
	if !ok {
		return 0, fmt.Errorf("codegraph is not installed")
	}
	root = strings.TrimSpace(root)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return 0, err
		}
		root = cwd
	}
	if !codegraph.IndexableRoot(root) {
		return 0, fmt.Errorf("codegraph: refusing to index %q — a filesystem root would index the whole volume", root)
	}
	if err := codegraph.EnsureInit(c.pluginCtx, bin, root); err != nil {
		return 0, fmt.Errorf("codegraph init: %w", err)
	}
	return c.connectMCPSpec(codegraph.MCPSpec(bin, root))
}




