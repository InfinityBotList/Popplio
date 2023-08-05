// Package teamhooks implements a webhook driver for teams.
//
// A new webhook handler for a different entity can be created by creating a new folder here
package teamhooks

import (
	"errors"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks/events"
	"popplio/webhooks/sender"
	"strings"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

const EntityType = "team"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	teamColsArr = utils.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ", ")
)

// Simple ergonomic webhook builder
type With[T events.WebhookEvent] struct {
	UserID   string
	TeamID   string
	Metadata *events.WebhookMetadata
	Data     T
}

// Fills in Team and Creator from IDs
func Send[T events.WebhookEvent](with With[T]) error {
	if !strings.HasPrefix(string(with.Data.Event()), strings.ToUpper(EntityType)) {
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
		DeleteWebhook: func() error {
			_, err := state.Pool.Exec(state.Context, "UPDATE teams SET webhook = NULL WHERE id = $1", with.TeamID)

			if err != nil {
				return err
			}

			return nil
		},
	}

	resp := &events.WebhookResponse[T]{
		Creator: user,
		Targets: events.Target{
			Team: &team,
		},
		CreatedAt: time.Now().Unix(),
		Type:      with.Data.Event(),
		Data:      with.Data,
		Metadata:  events.ParseWebhookMetadata(with.Metadata),
	}

	// Fetch the webhook url from db
	var webhookURL string
	err = state.Pool.QueryRow(state.Context, "SELECT webhook FROM teams WHERE id = $1", team.ID).Scan(&webhookURL)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to fetch webhook url")
	}

	if utils.IsNone(webhookURL) {
		return errors.New("no webhook set")
	}

	params := with.Data.CreateHookParams(resp.Creator, resp.Targets)

	ok, err := sender.SendDiscord(
		user.ID,
		team.Name,
		webhookURL,
		func() error {
			_, err := state.Pool.Exec(state.Context, "UPDATE teams SET webhook = NULL WHERE id = $1", team.ID)

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
	err = state.Pool.QueryRow(state.Context, "SELECT web_auth FROM teams WHERE id = $1", team.ID).Scan(&webhookSecret)

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
