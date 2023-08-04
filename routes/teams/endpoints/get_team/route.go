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
		Description: "Gets a team by ID",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Team{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	// Convert ID to UUID
	if !utils.IsValidUUID(id) {
		return uapi.DefaultResponse(http.StatusNotFound)
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

	// Get team members
	memberRows, err := state.Pool.Query(d.Context, "SELECT "+tmCols+" FROM team_members WHERE team_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	team.Members, err = pgx.CollectRows(memberRows, pgx.RowToStructByName[types.TeamMember])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range team.Members {
		team.Members[i].User, err = dovewing.GetUser(d.Context, team.Members[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	indexBotsRows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE team_owner = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	team.Bots, err = pgx.CollectRows(indexBotsRows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range team.Bots {
		team.Bots[i].User, err = dovewing.GetUser(d.Context, team.Bots[i].BotID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		var code string

		err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", team.Bots[i].VanityRef).Scan(&code)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		team.Bots[i].Vanity = code
	}

	return uapi.HttpResponse{
		Json: team,
	}
}
