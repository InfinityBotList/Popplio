package delete_team

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Team",
		Description: "Deletes the team. Requires the 'Owner' permission. Returns a 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
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

	// Ensure manager has perms to edit member permissions etc.
	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !kittycat.HasPerm(perms, kittycat.Build("global", teams.PermissionOwner)) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "Only global owners can delete teams"},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error beginning transaction", zap.Error(err), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var botCount int

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE team_owner = $1", teamId).Scan(&botCount)

	if err != nil {
		state.Logger.Error("Error getting bot count [db count]", zap.Error(err), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if botCount > 0 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "You cannot delete a team with bots in it"},
		}
	}

	var serverCount int

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE team_owner = $1", teamId).Scan(&serverCount)

	if err != nil {
		state.Logger.Error("Error getting server count [db count]", zap.Error(err), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if serverCount > 0 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "You cannot delete a team with servers in it"},
		}
	}

	_, err = tx.Exec(d.Context, "DELETE FROM team_members WHERE team_id = $1", teamId)

	if err != nil {
		state.Logger.Error("Error deleting team members", zap.Error(err), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = tx.Exec(d.Context, "DELETE FROM teams WHERE id = $1", teamId)

	if err != nil {
		state.Logger.Error("Error deleting team", zap.Error(err), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error committing transaction", zap.Error(err), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
