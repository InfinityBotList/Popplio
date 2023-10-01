package resolvers

import (
	"context"
	"popplio/assets"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
)

var (
	tmColsArr = db.GetCols(types.TeamMember{})
	tmCols    = strings.Join(tmColsArr, ",")

	indexBotColsArr = db.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")

	indexServerColsArr = db.GetCols(types.IndexServer{})
	indexServerCols    = strings.Join(indexServerColsArr, ",")
)

func GetTeamEntities(ctx context.Context, teamId string, targets []string) (*types.TeamEntities, error) {
	eto := &types.TeamEntities{}

	for _, st := range targets {
		var isInvalid bool
		switch st {
		case "team_member":
			// Get team members
			memberRows, err := state.Pool.Query(ctx, "SELECT "+tmCols+" FROM team_members WHERE team_id = $1", teamId)

			if err != nil {
				return nil, err
			}

			eto.Members, err = pgx.CollectRows(memberRows, pgx.RowToStructByName[types.TeamMember])

			if err != nil {
				return nil, err
			}

			for i := range eto.Members {
				eto.Members[i].User, err = dovewing.GetUser(ctx, eto.Members[i].UserID, state.DovewingPlatformDiscord)

				if err != nil {
					return nil, err
				}
			}
		case "bot":
			indexBotsRows, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE team_owner = $1", teamId)

			if err != nil {
				return nil, err
			}

			eto.Bots, err = pgx.CollectRows(indexBotsRows, pgx.RowToStructByName[types.IndexBot])

			if err != nil {
				return nil, err
			}

			for i := range eto.Bots {
				eto.Bots[i].User, err = dovewing.GetUser(ctx, eto.Bots[i].BotID, state.DovewingPlatformDiscord)

				if err != nil {
					return nil, err
				}

				var code string

				err = state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", eto.Bots[i].VanityRef).Scan(&code)

				if err != nil {
					return nil, err
				}

				eto.Bots[i].Vanity = code
				eto.Bots[i].Banner = assets.BannerInfo("bots", eto.Bots[i].BotID)
			}
		case "server":
			indexServerRows, err := state.Pool.Query(ctx, "SELECT "+indexServerCols+" FROM servers WHERE team_owner = $1", teamId)

			if err != nil {
				return nil, err
			}

			eto.Servers, err = pgx.CollectRows(indexServerRows, pgx.RowToStructByName[types.IndexServer])

			if err != nil {
				return nil, err
			}

			for i := range eto.Servers {
				var code string

				err = state.Pool.QueryRow(ctx, "SELECT code FROM vanity WHERE itag = $1", eto.Servers[i].VanityRef).Scan(&code)

				if err != nil {
					return nil, err
				}

				eto.Servers[i].Vanity = code
				eto.Servers[i].Banner = assets.BannerInfo("servers", eto.Servers[i].ServerID)
			}
		default:
			isInvalid = true
		}

		if !isInvalid {
			eto.Targets = append(eto.Targets, st)
		}
	}

	return eto, nil
}
