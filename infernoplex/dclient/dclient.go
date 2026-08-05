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

var client bot.Client

func Get() bot.Client {
	if client == nil {
		panic("infernoplex: Discord client used before dclient.Setup")
	}
	return client
}

func Ready() bool {
	return client != nil
}

func Setup(ctx context.Context, listeners ...bot.EventListener) error {
	token := state.Config.Infernoplex.Token.Parse()

	if token == "" {
		return errors.New("infernoplex.token is empty: Infernoplex needs its own Discord token")
	}

	opts := []bot.ConfigOpt{
		bot.WithRestClientConfigOpts(state.ProxyRestOpts(token)...),
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuilds),
			gateway.WithCompress(true),
		),
		bot.WithCacheConfigOpts(

			cache.WithCaches(cache.FlagGuilds),
		),
		bot.WithEventListeners(listeners...),
	}

	c, err := disgo.New(token, opts...)

	if err != nil {
		return fmt.Errorf("failed to build the Infernoplex client: %w", err)
	}

	if err := c.OpenGateway(ctx); err != nil {
		return fmt.Errorf("failed to connect Infernoplex to the gateway: %w", err)
	}

	client = c

	state.Logger.Info("Infernoplex connected", zap.String("applicationID", c.ApplicationID().String()))

	return nil
}

func Close(ctx context.Context) {
	if client == nil {
		return
	}

	client.Close(ctx)
	client = nil
}
