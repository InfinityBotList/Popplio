package resolvers

import (
	"context"
	"fmt"
	"popplio/db"
	botAssets "popplio/routes/bots/assets"
	serverAssets "popplio/routes/servers/assets"
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
				// Set the user for each bot
				err = botAssets.ResolveIndexBot(ctx, &eto.Bots[i])

				if err != nil {
					return nil, fmt.Errorf("error occurred while resolving index bot: " + err.Error() + " botID: " + eto.Bots[i].BotID)
				}
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
				err := serverAssets.ResolveIndexServer(ctx, &eto.Servers[i])

				if err != nil {
					return nil, fmt.Errorf("error occurred while resolving index server: " + err.Error() + " serverID: " + eto.Servers[i].ServerID)
				}
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
