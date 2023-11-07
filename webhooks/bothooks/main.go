// Package bothooks implements a webhook driver for bots.
//
// A new webhook handler for a different entity can be created by creating a new folder here
package bothooks

import (
	"errors"
	"popplio/config"
	"popplio/state"
	"popplio/webhooks/events"
	"popplio/webhooks/sender"
	"slices"

	"github.com/infinitybotlist/eureka/dovewing"
	"go.uber.org/zap"
)

const EntityType = "bot"

// Simple ergonomic webhook builder
type With struct {
	UserID   string
	BotID    string
	Metadata *events.WebhookMetadata
	Data     events.WebhookEvent
}

// Fills in Bot and Creator from IDs
func Send(with With) error {
	if config.UseLegacyWebhooks(with.BotID) {
		state.Logger.Warn("webhooks v2 is not enabled for this bot, ignoring")
		return nil
	}

	targetTypes := with.Data.TargetTypes()
	if !slices.Contains(targetTypes, EntityType) {
		return errors.New("invalid event type")
	}

	bot, err := dovewing.GetUser(state.Context, with.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to fetch bot via dovewing for this bothook", zap.Error(err), zap.String("botID", with.BotID), zap.String("userID", with.UserID))
		return err
	}

	user, err := dovewing.GetUser(state.Context, with.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to fetch user via dovewing for this bothook", zap.Error(err), zap.String("botID", with.BotID), zap.String("userID", with.UserID))
		return err
	}

	state.Logger.Info("Sending webhook for bot " + bot.ID)

	entity := sender.WebhookEntity{
		EntityID:   bot.ID,
		EntityName: bot.Username,
		EntityType: EntityType,
	}

	resp := &events.WebhookResponse{
		Creator: user,
		Targets: events.Target{
			Bot: bot,
		},
		Type:     with.Data.Event(),
		Data:     with.Data,
		Metadata: events.ParseWebhookMetadata(with.Metadata),
	}

	return sender.Send(&sender.WebhookSendState{
		UserID: resp.Creator.ID,
		Entity: entity,
		Event:  resp,
	})
}
