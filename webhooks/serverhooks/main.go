// Package serverhooks implements a webhook driver for servers.
//
// A new webhook handler for a different entity can be created by creating a new folder here
package serverhooks

import (
	"errors"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/events"
	"popplio/webhooks/sender"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
)

const EntityType = "server"

// Simple ergonomic webhook builder
type With struct {
	UserID   string
	ServerID string
	Metadata *events.WebhookMetadata
	Data     events.WebhookEvent
}

// Fills in Server and Creator from IDs
func Send(with With) error {
	if with.Data.TargetType() != EntityType {
		return errors.New("invalid event type")
	}

	var name, avatar, short string

	err := state.Pool.QueryRow(state.Context, "SELECT name, avatar, short FROM servers WHERE server_id = $1", with.ServerID).Scan(&name, &avatar, &short)

	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("server not found")
	}

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	user, err := dovewing.GetUser(state.Context, with.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	state.Logger.Info("Sending webhook for server " + with.ServerID)

	entity := sender.WebhookEntity{
		EntityID:   with.ServerID,
		EntityName: name,
		EntityType: EntityType,
	}

	resp := &events.WebhookResponse{
		Creator: user,
		Targets: events.Target{
			Server: &types.SEO{
				Name:   name,
				ID:     with.ServerID,
				Avatar: avatar,
				Short:  short,
			},
		},
		CreatedAt: time.Now().Unix(),
		Type:      with.Data.Event(),
		Data:      with.Data,
		Metadata:  events.ParseWebhookMetadata(with.Metadata),
	}

	return sender.Send(&sender.WebhookSendState{
		UserID: resp.Creator.ID,
		Entity: entity,
		Event:  resp,
	})
}
