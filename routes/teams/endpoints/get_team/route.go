// Package get_team implements GET /teams/{id} — "Get Team".
//
// Gets a team by ID
package get_team

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"
	"popplio/votes"
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
		return resp.Err("Error querying team [db query]", err, zap.String("id", id))
	}

	team, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Team])

	if err != nil {
		return resp.Err("Error querying team [db collect]", err, zap.String("id", id))
	}

	// Ensure these always marshal as `[]` rather than `null` — a nil Go
	// slice serializes to JSON null, which crashes frontend consumers that
	// call .length/.map on it without a null check.
	if team.Tags == nil {
		team.Tags = []string{}
	}
	if team.ExtraLinks == nil {
		team.ExtraLinks = []types.Link{}
	}

	team.Entities, err = resolvers.GetTeamEntities(d.Context, id, targets)

	if err != nil {
		return resp.Err("Error resolving team entities", err, zap.String("id", id))
	}

	var code string

	err = state.Pool.QueryRow(d.Context, "SELECT code FROM vanity WHERE itag = $1", team.VanityRef).Scan(&code)

	if err != nil {
		return resp.Err("Error while getting bot vanity code [db fetch]", err, zap.String("id", id), zap.String("teamId", team.ID))
	}

	team.Vanity = code

	team.Votes, err = votes.EntityGetVoteCount(d.Context, state.Pool, id, "team")

	if err != nil {
		return resp.Err("Error while getting team vote count", err, zap.String("id", id))
	}

	return uapi.HttpResponse{
		Json: team,
	}
}
