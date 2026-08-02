// Package delete_team implements DELETE /teams/{tid} — "Delete Team".
//
// Deletes the team. Requires the 'Owner' permission. Returns a 204 on
// success
package delete_team

import (
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Team",
		Description: "Deletes the team. Requires the 'Owner' permission. Returns a 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "tid",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error beginning transaction", err, zap.String("tid", teamId))
	}

	var botCount int

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE team_owner = $1", teamId).Scan(&botCount)

	if err != nil {
		return resp.Err("Error getting bot count [db count]", err, zap.String("tid", teamId))
	}

	if botCount > 0 {
		return resp.BadRequest("You cannot delete a team with bots in it")
	}

	var serverCount int

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE team_owner = $1", teamId).Scan(&serverCount)

	if err != nil {
		return resp.Err("Error getting server count [db count]", err, zap.String("tid", teamId))
	}

	if serverCount > 0 {
		return resp.BadRequest("You cannot delete a team with servers in it")
	}

	_, err = tx.Exec(d.Context, "DELETE FROM team_members WHERE team_id = $1", teamId)

	if err != nil {
		return resp.Err("Error deleting team members", err, zap.String("tid", teamId))
	}

	_, err = tx.Exec(d.Context, "DELETE FROM teams WHERE id = $1", teamId)

	if err != nil {
		return resp.Err("Error deleting team", err, zap.String("tid", teamId))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error committing transaction", err, zap.String("tid", teamId))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
