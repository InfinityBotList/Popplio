package get_team

import (
	"errors"
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

	"github.com/go-chi/chi/v5"
)

var (
	teamColsArr = utils.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ",")

	tmColsArr = utils.GetCols(types.TeamMember{})
	tmCols    = strings.Join(tmColsArr, ",")

	indexBotColsArr = utils.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team",
		Description: "Gets a team by ID(s)",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Team ID. Can be a comma-seperated list of IDs",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "targets",
				Description: "Entities to get. Can be one of the following: `team_member`, `bot`. Comma-seperated",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.TeamResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	idStr := chi.URLParam(r, "id")
	targetStr := r.URL.Query().Get("targets")

	ids := strings.Split(idStr, ",")
	targets := strings.Split(targetStr, ",")

	var tr = types.TeamResponse{}

	for _, id := range ids {
		// Convert ID to UUID
		if !utils.IsValidUUID(id) {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: "Team ID: " + id + " is not a valid UUID",
				},
			}
		}

		rows, err := state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", id)

		if errors.Is(err, pgx.ErrNoRows) {
			return uapi.DefaultResponse(http.StatusNotFound)
		}

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		team, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Team])

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		team.Entities = &types.TeamEntities{}

		for _, st := range targets {
			var isInvalid bool
			switch st {
			case "team_member":
				// Get team members
				memberRows, err := state.Pool.Query(d.Context, "SELECT "+tmCols+" FROM team_members WHERE team_id = $1", id)

				if err != nil {
					state.Logger.Error(err)
					return uapi.DefaultResponse(http.StatusInternalServerError)
				}

				team.Entities.Members, err = pgx.CollectRows(memberRows, pgx.RowToStructByName[types.TeamMember])

				if err != nil {
					state.Logger.Error(err)
					return uapi.DefaultResponse(http.StatusInternalServerError)
				}

				for i := range team.Entities.Members {
					team.Entities.Members[i].User, err = dovewing.GetUser(d.Context, team.Entities.Members[i].UserID, state.DovewingPlatformDiscord)

					if err != nil {
						state.Logger.Error(err)
						return uapi.DefaultResponse(http.StatusInternalServerError)
					}
				}
			case "bot":
				indexBotsRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE team_owner = $1", id)

				if err != nil {
					state.Logger.Error(err)
					return uapi.DefaultResponse(http.StatusInternalServerError)
				}

				team.Entities.Bots, err = pgx.CollectRows(indexBotsRows, pgx.RowToStructByName[types.IndexBot])

				if err != nil {
					state.Logger.Error(err)
					return uapi.DefaultResponse(http.StatusInternalServerError)
				}

				for i := range team.Entities.Bots {
					team.Entities.Bots[i].User, err = dovewing.GetUser(d.Context, team.Entities.Bots[i].BotID, state.DovewingPlatformDiscord)

					if err != nil {
						state.Logger.Error(err)
						return uapi.DefaultResponse(http.StatusInternalServerError)
					}

					var code string

					err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", team.Entities.Bots[i].VanityRef).Scan(&code)

					if err != nil {
						state.Logger.Error(err)
						return uapi.DefaultResponse(http.StatusInternalServerError)
					}

					team.Entities.Bots[i].Vanity = code
				}
			default:
				isInvalid = true
			}

			if !isInvalid {
				team.Entities.Targets = append(team.Entities.Targets, st)
			}
		}

		tr.Teams = append(tr.Teams, team)
	}
	return uapi.HttpResponse{
		Json: tr,
	}
}
