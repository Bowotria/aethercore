package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fzihak/aethercore/core"
	"github.com/fzihak/aethercore/gateway/discord"
	"github.com/fzihak/aethercore/sdk"
)

// handleDiscordCmd parses 'aether discord' sub-flags and starts the Discord
// gateway bot, blocking until SIGINT/SIGTERM is received.
func handleDiscordCmd(args []string) {
	dcCmd := flag.NewFlagSet("discord", flag.ContinueOnError)
	token := dcCmd.String("token", "", "Discord bot token (or set DISCORD_TOKEN env var) [required]")

	dcCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aether discord --token <BOT_TOKEN>\n\n")
		fmt.Fprintf(os.Stderr, "Starts the AetherCore Discord gateway.\n")
		fmt.Fprintf(os.Stderr, "The bot responds to !start, !help, !run, and !modules.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		dcCmd.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment:\n")
		fmt.Fprintf(os.Stderr, "  DISCORD_TOKEN   Alternative to --token flag\n")
	}

	if err := dcCmd.Parse(args); err != nil {
		core.Logger().Error("discord_parse_flags_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// --token flag takes precedence; fall back to environment variable.
	botToken := *token
	if botToken == "" {
		botToken = os.Getenv("DISCORD_TOKEN")
	}
	if botToken == "" {
		fmt.Fprintln(os.Stderr, "Error: Discord bot token required (--token or DISCORD_TOKEN env var)")
		dcCmd.Usage()
		os.Exit(1)
	}

	// Build an empty module registry; operators can pre-load modules before
	// handing off to handleDiscordCmd by extending this function in future.
	registry := sdk.NewModuleRegistry()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	core.Logger().Info("discord_gateway_starting")

	bot := discord.NewBot(botToken, registry)
	err := bot.Start(ctx)
	stop() // release signal resources regardless of outcome
	if err != nil {
		core.Logger().Error("discord_gateway_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	core.Logger().Info("discord_gateway_stopped")
}
