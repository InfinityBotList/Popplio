package delete_team_member

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Team Member",
		Description: "Deletes a member from the team. Users can always delete themselves. Returns a 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "tid",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "mid",
				Description: "Member ID",
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
	var userId = chi.URLParam(r, "mid")

	userPerms, err := teams.GetEntityPerms(d.Context, userId, "team", teamId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if d.Auth.ID != userId {
		// Ensure manager has perms to delete members
		managerPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

		if err != nil {
			state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
			}
		}

		// Ensure manager has permissions to remove all user perms
		if err := kittycat.CheckPatchChanges(managerPerms, userPerms, []kittycat.Permission{}); err != nil {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to delete this member:" + err.Error()},
			}
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error starting transaction", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Ensure that if perm is owner, then there is another owner
	if !kittycat.HasPerm(userPerms, kittycat.Permission{Namespace: "global", Perm: teams.PermissionOwner}) {
		var ownerCount int

		err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND flags && $2", teamId, []string{kittycat.Permission{Namespace: "global", Perm: teams.PermissionOwner}.String()}).Scan(&ownerCount)

		if err != nil {
			state.Logger.Error("Error getting owner count", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if ownerCount < 2 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "There needs to be one other global owner before you can remove yourself from owner"},
			}
		}
	}

	_, err = tx.Exec(d.Context, "DELETE FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId)

	if err != nil {
		state.Logger.Error("Error deleting member", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error committing transaction", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
