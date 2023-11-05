package get_team

import (
	"errors"
	"net/http"
	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"
	"strings"

	"github.com/google/uuid"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

var (
	teamColsArr = db.GetCols(types.Team{})
	teamCols    = strings.Join(teamColsArr, ",")
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
			{
				Name:        "targets",
				Description: "Entities to get. Can be one of the following: `team_member`, `bot`, `server`. Comma-seperated",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Team{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")
	targetStr := r.URL.Query().Get("targets")
	targets := strings.Split(targetStr, ",")

	// Convert ID to UUID
	if _, err := uuid.Parse(id); err != nil {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+teamCols+" FROM teams WHERE id = $1", id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Error querying team [db query]", zap.Error(err), zap.String("id", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	team, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Team])

	if err != nil {
		state.Logger.Error("Error querying team [db collect]", zap.Error(err), zap.String("id", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	team.Banner = assetmanager.BannerInfo(assetmanager.AssetTargetTypeTeams, id)
	team.Avatar = assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, id)

	team.Entities, err = resolvers.GetTeamEntities(d.Context, id, targets)

	if err != nil {
		state.Logger.Error("Error resolving team entities", zap.Error(err), zap.String("id", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: team,
	}
}
