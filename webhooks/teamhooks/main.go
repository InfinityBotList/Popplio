// Package teamhooks implements a webhook driver for teams.
//
// A new webhook handler for a different entity can be created by creating a new folder here
package teamhooks

import (
	"errors"
	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/events"
	"popplio/webhooks/sender"
	"strings"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
)

const EntityType = "team"

var (
	teamColsArr = db.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ", ")
)

// Simple ergonomic webhook builder
type With struct {
	UserID   string
	TeamID   string
	Metadata *events.WebhookMetadata
	Data     events.WebhookEvent
}

// Fills in Team and Creator from IDs
func Send(with With) error {
	if with.Data.TargetType() != EntityType {
		return errors.New("invalid event type")
	}

	row, err := state.Pool.Query(state.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", with.TeamID)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	team, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.Team])

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	team.Entities = &types.TeamEntities{
		Targets: []string{}, // We don't provide any entities right now, may change
	}

	team.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeTeams, team.ID)
	team.Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, team.ID)

	user, err := dovewing.GetUser(state.Context, with.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	state.Logger.Info("Sending webhook for team " + team.ID)

	entity := sender.WebhookEntity{
		EntityID:   team.ID,
		EntityName: team.Name,
		EntityType: EntityType,
	}

	resp := &events.WebhookResponse{
		Creator: user,
		Targets: events.Target{
			Team: &team,
		},
		CreatedAt: time.Now().Unix(),
		Type:      with.Data.Event(),
		Data:      with.Data,
		Metadata:  events.ParseWebhookMetadata(with.Metadata),
	}

	return sender.Send(&sender.WebhookSendState{
		Event:  resp,
		UserID: resp.Creator.ID,
		Entity: entity,
	})
}
