package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/fzihak/aethercore/sdk"
)

// DefaultCommandPrefix is the command prefix used by the AetherCore Discord bot.
// Users send "!run", "!help", "!modules", etc.
const DefaultCommandPrefix = "!"

// Bot is the top-level AetherCore Discord gateway. It wires together the
// REST Client, WebSocket Gateway, Router, and Adapter into a single Start/Stop
// unit with the same lifecycle contract as the Telegram gateway.
//
// Typical usage:
//
//	registry := sdk.NewModuleRegistry()
//	// … load modules into registry …
//	bot := discord.NewBot(os.Getenv("DISCORD_TOKEN"), registry)
//	if err := bot.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
type Bot struct {
	token    string
	registry *sdk.ModuleRegistry
	log      *slog.Logger
}

// NewBot constructs a Bot for the given Discord bot token and module registry.
// The logger is pre-tagged with service.name=aethercore and
// component=gateway.discord to match the OpenTelemetry conventions used by the
// rest of the AetherCore kernel.
func NewBot(token string, registry *sdk.ModuleRegistry) *Bot {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})).With(
		slog.String("service.name", "aethercore"),
		slog.String("component", "gateway.discord"),
	)
	return &Bot{token: token, registry: registry, log: log}
}

// Start validates the bot token by fetching the Gateway URL, wires the Router
// and Adapter, and begins the WebSocket event loop. It blocks until ctx is
// cancelled.
//
// Returns an error only if the initial REST call (GetGatewayURL) fails, which
// indicates a bad token or network issue before the session even begins.
func (b *Bot) Start(ctx context.Context) error {
	if b.token == "" {
		return errors.New("discord: bot token must not be empty")
	}

	client := NewClient(b.token)

	// Validate the token and retrieve the preferred Gateway WebSocket URL.
	gatewayURL, err := client.GetGatewayURL(ctx)
	if err != nil {
		return fmt.Errorf("discord: token validation failed: %w", err)
	}
	b.log.Info("discord_gateway_url_resolved", slog.String("url", gatewayURL))

	adapter := NewAdapter(client, b.registry, b.log)

	router := NewRouter(DefaultCommandPrefix)
	router.Register("start", adapter.HandleHelp)
	router.Register("help", adapter.HandleHelp)
	router.Register("run", adapter.HandleRun)
	router.Register("modules", adapter.HandleModules)

	gateway := NewGateway(b.token, DefaultIntents, router.Handle, b.log)
	gateway.Run(ctx, gatewayURL) // blocks until ctx cancelled
	return nil
}
