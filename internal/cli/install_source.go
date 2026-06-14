// Package cli: install_source.go — CLI frontend for the install_source tool.
// It allows users to install skills and MCP servers directly from the command
// line without starting a chat session. This is the CLI entry point for
// "reasonix install-source install --source <url>".
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"reasonix/internal/installsource"
)

func installSourceCommand(args []string) int {
	if len(args) == 0 {
		fmt.Println("reasonix install-source — install skills and MCP servers")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  reasonix install-source install --source <url|path> [--apply] [--scope project|global]")
		fmt.Println("  reasonix install-source install --source <url|path> --kind mcp [--apply]")
		fmt.Println("  reasonix install-source uninstall --name <name>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  reasonix install-source install --source https://raw.githubusercontent.com/aliatx2017/reasonix-hermes/main/skills-hub/skills/golang-testing/SKILL.md")
		fmt.Println("  reasonix install-source install --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills/golang-testing")
		fmt.Println("  reasonix install-source uninstall --name golang-testing")
		return 0
	}

	op := args[0]
	switch op {
	case "install":
		return installSourceInstall(args[1:])
	case "uninstall":
		return installSourceUninstall(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown install-source command: %s\n", op)
		return 2
	}
}

func installSourceInstall(args []string) int {
	var (
		source      string
		apply       bool
		scope       string
		kind        string
		mode        string
		name        string
		transport   string
		command     string
		tier        string
		envKeys     []string
		envVals     []string
		headerKeys  []string
		headerVals  []string
		replace     bool
		strict      *bool
	)

	// Simple flag parsing — CLI is a thin wrapper, so we keep it light.
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "--source" && i+1 < len(args):
			i++
			source = args[i]
		case a == "--apply":
			apply = true
		case a == "--scope" && i+1 < len(args):
			i++
			scope = args[i]
		case a == "--kind" && i+1 < len(args):
			i++
			kind = args[i]
		case a == "--mode" && i+1 < len(args):
			i++
			mode = args[i]
		case a == "--name" && i+1 < len(args):
			i++
			name = args[i]
		case a == "--transport" && i+1 < len(args):
			i++
			transport = args[i]
		case a == "--command" && i+1 < len(args):
			i++
			command = args[i]
		case a == "--tier" && i+1 < len(args):
			i++
			tier = args[i]
		case a == "--env" && i+1 < len(args):
			i++
			kv := args[i]
			parts := strings.SplitN(kv, "=", 2)
			envKeys = append(envKeys, parts[0])
			if len(parts) > 1 {
				envVals = append(envVals, parts[1])
			} else {
				envVals = append(envVals, "")
			}
		case a == "--header" && i+1 < len(args):
			i++
			kv := args[i]
			parts := strings.SplitN(kv, "=", 2)
			headerKeys = append(headerKeys, parts[0])
			if len(parts) > 1 {
				headerVals = append(headerVals, parts[1])
			} else {
				headerVals = append(headerVals, "")
			}
		case a == "--replace":
			replace = true
		case a == "--strict":
			t := true
			strict = &t
		case a == "--no-strict":
			f := false
			strict = &f
		case a == "--help", a == "-h":
			fmt.Println("reasonix install-source install — install skills and MCP servers")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --source <url|path>    URL, local path, or package name (required)")
			fmt.Println("  --apply                Actually write/connect (default: plan only)")
			fmt.Println("  --scope project|global Install scope (default: project)")
			fmt.Println("  --kind auto|skill|mcp  Capability kind (default: auto-detect)")
			fmt.Println("  --mode auto|copy|link  Skill install mode")
			fmt.Println("  --name <name>          Override discovered name")
			fmt.Println("  --tier lazy|bg|eager   MCP startup tier")
			fmt.Println("  --replace              Allow replacing existing MCP config")
			fmt.Println("  --env KEY=VALUE        Environment variable for MCP server")
			fmt.Println("  --header KEY=VALUE     HTTP header for remote MCP")
			fmt.Println("  --transport stdio|http Transport override")
			fmt.Println("  --command <cmd>        Command override for stdio MCP")
			fmt.Println("  --strict/--no-strict   Require skill frontmatter (default: strict)")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", a)
			return 2
		}
		i++
	}

	if source == "" {
		fmt.Fprintln(os.Stderr, "install-source: --source is required")
		return 2
	}

	if scope == "" {
		scope = "project"
	}

	// Build env and headers maps.
	env := make(map[string]string, len(envKeys))
	for j := range envKeys {
		env[envKeys[j]] = envVals[j]
	}
	headers := make(map[string]string, len(headerKeys))
	for j := range headerKeys {
		headers[headerKeys[j]] = headerVals[j]
	}

	tool := installsource.NewTool(installsource.Options{})
	r := installsource.RunCLI(context.Background(), tool, installsource.CLIRequest{
		Op:        "install",
		Source:    source,
		Apply:     apply,
		Scope:     scope,
		Kind:      kind,
		Mode:      mode,
		Name:      name,
		Transport: transport,
		Command:   command,
		Tier:      tier,
		Env:       env,
		Headers:   headers,
		Replace:   replace,
		Strict:    strict,
	})

	fmt.Println(r.Output)
	if !r.OK {
		return 1
	}
	return 0
}

func installSourceUninstall(args []string) int {
	var name string
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "--name" && i+1 < len(args):
			i++
			name = args[i]
		case a == "--help", a == "-h":
			fmt.Println("reasonix install-source uninstall — remove skills and MCP servers")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --name <name>    Name of skill or MCP server to remove (required)")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", a)
			return 2
		}
		i++
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "install-source uninstall: --name is required")
		return 2
	}

	tool := installsource.NewTool(installsource.Options{})
	r := installsource.RunCLI(context.Background(), tool, installsource.CLIRequest{
		Op:   "uninstall",
		Name: name,
	})

	fmt.Println(r.Output)
	if !r.OK {
		return 1
	}
	return 0
}
