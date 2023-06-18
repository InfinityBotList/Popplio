// Package bothooks implements a webhook driver for bots.
//
// A new webhook handler for a different entity can be created by creating a new folder here
package bothooks

import (
	"errors"
	"popplio/state"
	"popplio/webhooks/events"
	"popplio/webhooks/sender"
	"strings"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

const EntityType = "BOT"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Simple ergonomic webhook builder
type With[T events.WebhookEvent] struct {
	UserID string
	BotID  string
	Data   T
}

// Fills in Bot and Creator from IDs
func Send[T events.WebhookEvent](with With[T]) error {
	if !strings.HasPrefix(string(with.Data.Event()), EntityType) {
		return errors.New("invalid event type")
	}

	bot, err := dovewing.GetUser(state.Context, with.BotID, state.Discord)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	user, err := dovewing.GetUser(state.Context, with.UserID, state.Discord)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	state.Logger.Info("Sending webhook for bot " + bot.ID)

	entity := sender.WebhookEntity{
		EntityID:   bot.ID,
		EntityName: bot.Username,
		EntityType: EntityType,
		DeleteWebhook: func() error {
			_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", bot.ID)

			if err != nil {
				return err
			}

			return nil
		},
	}

	resp := &events.WebhookResponse[T]{
		Creator: user,
		Targets: events.Target{
			Bot: bot,
		},
		CreatedAt: time.Now().Unix(),
		Type:      with.Data.Event(),
		Data:      with.Data,
	}

	// Fetch the webhook url from db
	var webhookURL string
	var webhooksV2 bool
	err = state.Pool.QueryRow(state.Context, "SELECT webhook, webhooks_v2 FROM bots WHERE bot_id = $1", bot.ID).Scan(&webhookURL, &webhooksV2)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to fetch webhook url")
	}

	if !webhooksV2 {
		state.Logger.Warn("webhooks v2 is not enabled for this bot, ignoring")
		return nil
	}

	params := with.Data.CreateHookParams(resp.Creator, resp.Targets)

	ok, err := sender.SendDiscord(
		user.ID,
		bot.Username,
		webhookURL,
		func() error {
			_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", bot.ID)

			if err != nil {
				return err
			}

			return nil
		},
		params,
	)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	if ok {
		return nil
	}

	var webhookSecret pgtype.Text
	err = state.Pool.QueryRow(state.Context, "SELECT web_auth FROM bots WHERE bot_id = $1", bot.ID).Scan(&webhookSecret)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to fetch webhook secret")
	}

	payload, err := json.Marshal(resp)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to marshal webhook payload")
	}

	return sender.Send(&sender.WebhookSendState{
		Url: webhookURL,
		Sign: sender.Secret{
			Raw: webhookSecret.String,
		},
		Data:   payload,
		UserID: resp.Creator.ID,
		Entity: entity,
	})
}
