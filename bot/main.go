// Discord Bot Gateway — thin standalone entry point.
//
// This binary exists for convenience (run the Discord bot without the full
// reasonix CLI). The real implementation lives in internal/bot/discord/ and
// internal/bot/gateway.go. For full multi-platform bot, use:
//
//	reasonix bot start --channels discord
//
// Environment variables:
//
//	DISCORD_BOT_TOKEN   Discord bot token (required)
//	DISCORD_SERVER_ID   Target server ID (optional)
//	DISCORD_CHANNEL_ID  Target channel ID (optional; empty = all channels)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reasonix/internal/bot"
	"reasonix/internal/bot/discord"
	"reasonix/internal/config"
)

func main() {
	token := flag.String("token", "", "Discord bot token (overrides DISCORD_BOT_TOKEN env)")
	serverID := flag.String("server", "", "Target server/guild ID")
	channelID := flag.String("channel", "", "Restrict to single channel (empty = all)")
	model := flag.String("model", "", "Model name (empty = default_model)")
	allowAll := flag.Bool("allow-all", false, "Allow all users (dangerous for public bots)")
	dir := flag.String("dir", "", "Workspace root directory")
	flag.Parse()

	// Resolve token
	botToken := resolveToken(*token)
	if botToken == "" {
		fmt.Fprintln(os.Stderr, "error: DISCORD_BOT_TOKEN is required. Set it via environment variable or --token flag.")
		os.Exit(1)
	}
	os.Setenv("DISCORD_BOT_TOKEN", botToken)

	// Load reasonix config for model resolution
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load config: %v (using defaults)\n", err)
		cfg = &config.Config{}
	}

	workspaceRoot := resolveWorkspaceRoot(*dir)

	modelName := resolveModelName(*model, cfg)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	discordCfg := config.DiscordBotConfig{
		Enabled:  true,
		TokenEnv: "DISCORD_BOT_TOKEN",
		ServerID: *serverID,
		ChannelID: *channelID,
		AllowDMs: true,
	}

	gwCfg := bot.GatewayConfig{
		Model:         modelName,
		MaxSteps:      cfg.Bot.MaxSteps,
		WorkspaceRoot: workspaceRoot,
		Enabled:       map[bot.Platform]bool{bot.PlatformDiscord: true},
		Allowlist: bot.AllowlistConfig{
			Enabled:  true,
			AllowAll: *allowAll,
			Users:    map[bot.Platform][]string{bot.PlatformDiscord: cfg.Bot.Allowlist.DiscordUsers},
			Groups:   map[bot.Platform][]string{bot.PlatformDiscord: cfg.Bot.Allowlist.DiscordGroups},
		},
		Debounce: time.Duration(cfg.Bot.DebounceMs) * time.Millisecond,
	}

	adapters := map[bot.Platform]bot.Adapter{
		bot.PlatformDiscord: discord.New(discordCfg, logger),
	}

	gw := bot.NewGateway(gwCfg, adapters, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nshutting down...")
		cancel()
		gw.Stop()
	}()

	fmt.Fprintf(os.Stderr, "reasonix discord bot starting (model: %s)...\n", modelName)

	if err := gw.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: start gateway: %v\n", err)
		os.Exit(1)
	}

	<-ctx.Done()
}

// resolveToken returns the Discord bot token. flagToken takes precedence;
// if empty, the DISCORD_BOT_TOKEN environment variable is consulted.
func resolveToken(flagToken string) string {
	if flagToken != "" {
		return flagToken
	}
	return os.Getenv("DISCORD_BOT_TOKEN")
}

// resolveModelName picks the model name from the most specific source:
// CLI flag → config Bot.Model → config DefaultModel → hardcoded fallback.
func resolveModelName(flagModel string, cfg *config.Config) string {
	if flagModel != "" {
		return flagModel
	}
	if cfg != nil && cfg.Bot.Model != "" {
		return cfg.Bot.Model
	}
	if cfg != nil && cfg.DefaultModel != "" {
		return cfg.DefaultModel
	}
	return "deepseek-flash"
}

// resolveWorkspaceRoot returns the workspace directory. flagDir takes
// precedence; if empty, the current working directory is used.
func resolveWorkspaceRoot(flagDir string) string {
	if flagDir != "" {
		return flagDir
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
