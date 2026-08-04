// Package dclient owns the staff bot's Discord connection.
//
// Arcadia runs its own gateway connection under its own bot identity, separate
// from Popplio's. Keeping the client here rather than in popplio/state means:
//
//   - the staff bot's presence, prefix and slash commands belong to the Arcadia
//     application, so mod-log embeds and audit-log entries are attributed to it
//   - Popplio's session keeps its narrower intents; only this client asks for the
//     staff-side ones
//   - either half can be restarted without dropping the other's gateway
//
// Everything in arcadia/ talks to Discord through Get().
package dclient

import (
	"context"
	"errors"
	"fmt"

	"popplio/state"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/gateway"
	"go.uber.org/zap"
)

// client is the staff bot's Discord client. It is nil until Setup runs.
var client bot.Client

// Get returns the staff bot's Discord client.
//
// It panics if called before Setup, which would be a wiring mistake rather than
// a runtime condition: every caller runs after arcadia.Start.
func Get() bot.Client {
	if client == nil {
		panic("arcadia: Discord client used before dclient.Setup")
	}
	return client
}

// Ready reports whether the client has been built, so callers on Popplio's side
// can degrade instead of panicking if the staff bot failed to start.
func Ready() bool {
	return client != nil
}

// Setup builds the staff bot client and opens its gateway connection.
//
// listeners are attached before connecting so no early event is missed.
func Setup(ctx context.Context, listeners ...bot.EventListener) error {
	token := state.Config.Arcadia.Token.Parse()

	if token == "" {
		return errors.New("arcadia.token is empty: the staff bot needs its own Discord token")
	}

	opts := []bot.ConfigOpt{
		bot.WithRestClientConfigOpts(state.ProxyRestOpts(token)...),
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(intents()),
			gateway.WithCompress(true),
		),
		bot.WithCacheConfigOpts(
			// Guilds, members and presences back dovewing's live tier and the
			// member checks in the RPC layer; roles back the staff position
			// validation in the panel.
			cache.WithCaches(cache.FlagGuilds | cache.FlagMembers | cache.FlagPresences | cache.FlagRoles),
		),
		bot.WithEventListeners(listeners...),
	}

	c, err := disgo.New(token, opts...)

	if err != nil {
		return fmt.Errorf("failed to build the staff bot client: %w", err)
	}

	if err := c.OpenGateway(ctx); err != nil {
		return fmt.Errorf("failed to connect the staff bot to the gateway: %w", err)
	}

	client = c

	state.Logger.Info("Staff bot connected", zap.String("applicationID", c.ApplicationID().String()))

	return nil
}

// intents returns the gateway intents the staff bot needs.
//
// Message Content is privileged and only requested when prefix commands are
// enabled; slash commands are the primary interface and need none of it.
func intents() gateway.Intents {
	i := gateway.IntentGuilds |
		gateway.IntentGuildMembers |
		gateway.IntentGuildPresences |
		gateway.IntentGuildModeration

	if state.Config.Arcadia.PrefixCommands {
		i |= gateway.IntentGuildMessages | gateway.IntentMessageContent
	}

	return i
}

// Close shuts the gateway connection down.
func Close(ctx context.Context) {
	if client == nil {
		return
	}

	client.Close(ctx)
	client = nil
}
