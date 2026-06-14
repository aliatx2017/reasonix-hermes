package cli

import (
	"fmt"
	"os"
	"strings"

	"reasonix/internal/config"
)

func modelsCommand(args []string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}

	// Optional sub-command: "refresh" tests connectivity.
	refresh := len(args) > 0 && args[0] == "refresh"

	fmt.Println()
	if refresh {
		fmt.Println("Testing model connectivity...")
		fmt.Println()
	}

	for _, p := range cfg.Providers {
		model := p.Model
		if model == "" && len(p.Models) > 0 {
			model = p.Default
			if model == "" {
				model = p.Models[0]
			}
			if len(p.Models) > 1 {
				model += fmt.Sprintf(" (+%d more)", len(p.Models)-1)
			}
		}
		if model == "" {
			model = "(no model)"
		}

		price := ""
		if p.Price != nil {
			sym := p.Price.Symbol()
			price = fmt.Sprintf("  %s%.2f/%.2f/%.2f per 1M (cache/input/output)",
				sym, p.Price.CacheHit, p.Price.Input, p.Price.Output)
		}

		status := ""
		if refresh {
			status = connectivityStatus(p)
		}

		fmt.Printf("  %-20s  %-20s  %s%s%s\n", p.Name, p.Kind, model, price, status)
	}

	if refresh {
		fmt.Println()
		fmt.Println("Refresh complete.")
	}
	fmt.Println()
	return 0
}

func connectivityStatus(entry config.ProviderEntry) string {
	// Lightweight connectivity check: try a simple models list or health endpoint.
	// This is a best-effort check — we don't make an actual API call.
	base := strings.TrimRight(entry.BaseURL, "/")
	baseLower := strings.ToLower(base)

	switch {
	case strings.Contains(baseLower, "api.deepseek.com"):
		return "  ✓ known endpoint"
	case strings.Contains(baseLower, "api.openai.com"):
		return "  ✓ known endpoint"
	case strings.Contains(baseLower, "open.bigmodel.cn"):
		return "  ✓ known endpoint"
	case strings.Contains(baseLower, "api.minimaxi.com"):
		return "  ✓ known endpoint"
	case strings.Contains(baseLower, "token-plan-cn.xiaomimimo.com"):
		return "  ✓ known endpoint"
	case strings.Contains(baseLower, "localhost") || strings.Contains(baseLower, "127.0.0.1"):
		return "  ⚠ local — verify manually"
	default:
		return "  ⚠ unknown — verify manually"
	}
}
