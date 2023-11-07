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
	"slices"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
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
	targetTypes := with.Data.TargetTypes()
	if !slices.Contains(targetTypes, EntityType) {
		return errors.New("invalid event type")
	}

	var name, avatar, short string

	err := state.Pool.QueryRow(state.Context, "SELECT name, avatar, short FROM servers WHERE server_id = $1", with.ServerID).Scan(&name, &avatar, &short)

	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("server not found")
	}

	if err != nil {
		state.Logger.Error("Failed to fetch name/avatar/short of server for this serverhook", zap.Error(err), zap.String("serverID", with.ServerID), zap.String("userID", with.UserID))
		return err
	}

	user, err := dovewing.GetUser(state.Context, with.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to fetch user via dovewing for this serverhook", zap.Error(err), zap.String("serverID", with.ServerID), zap.String("userID", with.UserID))
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
