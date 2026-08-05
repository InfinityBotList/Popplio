package bot

import (
	"context"

	"popplio/state"

	"github.com/disgoorg/disgo/events"
	"go.uber.org/zap"
)

func onReady(_ context.Context, e *events.Ready) {
	state.Logger.Info("Infernoplex is ready!", zap.String("username", e.User.Username))

	loadOwners()

	if err := SyncCommands(); err != nil {
		state.Logger.Error("Failed to sync Infernoplex application commands", zap.Error(err))
	}
}
