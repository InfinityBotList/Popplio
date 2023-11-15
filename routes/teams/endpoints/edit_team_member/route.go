package edit_team_member

import (
	"errors"
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"slices"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edit Team Member Permissions",
		Description: "Edits a members permissions on a team. Returns a 204 on success",
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
			{
				Name:        "mid",
				Description: "Team Member ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.EditTeamMember{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")
	var userId = chi.URLParam(r, "mid")

	var payload types.EditTeamMember

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Get team permissions for user
	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error starting transaction", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	if payload.PermUpdate != nil {
		if !perms.Has("team_member", teams.PermissionEdit) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to edit this member"},
			}
		}

		// Get the old permissions of the user
		var oldPerms []string
		err = tx.QueryRow(d.Context, "SELECT flags FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId).Scan(&oldPerms)

		if errors.Is(err, pgx.ErrNoRows) {
			return uapi.HttpResponse{
				Status: http.StatusNotFound,
				Json:   types.ApiError{Message: "User is not a member of this team"},
			}
		}

		if err != nil {
			state.Logger.Error("Error getting old perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		var newPerms = oldPerms // The new permissions of the user

		// Check if manager has all perms they are trying to add
		for _, perm := range payload.PermUpdate.Add {
			if !teams.IsValidPerm(perm) {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Invalid permission: " + perm},
				}
			}

			if !perms.HasRaw(perm) {
				return uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json:   types.ApiError{Message: "You do not have permission to add " + perm},
				}
			}

			if !slices.Contains(oldPerms, perm) {
				// Add perm
				newPerms = append(newPerms, perm)
			}
		}

		for _, perm := range payload.PermUpdate.Remove {
			// Ensure that if perm is owner, then there is another owner
			if perm == "global."+teams.PermissionOwner {
				var ownerCount int

				err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND flags && $2", teamId, []string{teams.PermissionOwner}).Scan(&ownerCount)

				if err != nil {
					state.Logger.Error("Error getting owner count", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
					return uapi.DefaultResponse(http.StatusInternalServerError)
				}

				if ownerCount == 0 {
					return uapi.HttpResponse{
						Status: http.StatusBadRequest,
						Json:   types.ApiError{Message: "There needs to be one other global owner before you can remove yourself from owner"},
					}
				}
			}

			if !teams.IsValidPerm(perm) {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Invalid permission: " + perm},
				}
			}

			if !perms.HasRaw(perm) {
				return uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json:   types.ApiError{Message: "You do not have permission to remove " + perm},
				}
			}

			// Remove perm
			filteredPerms := []string{}
			for _, p := range newPerms {
				if p != perm {
					filteredPerms = append(filteredPerms, p)
				}
			}
			newPerms = filteredPerms
		}

		newPerms = teams.NewPermMan(newPerms).Perms()

		_, err = tx.Exec(d.Context, "UPDATE team_members SET flags = $1 WHERE team_id = $2 AND user_id = $3", newPerms, teamId, userId)

		if err != nil {
			state.Logger.Error("Error updating perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Mentionable != nil {
		// All members can update their own mentionable status
		if d.Auth.ID != userId {
			// Ensure manager has perms to edit member permissions etc.
			perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

			if err != nil {
				state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
				}
			}

			if !perms.Has("team_member", teams.PermissionEdit) {
				return uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json:   types.ApiError{Message: "You do not have permission to edit this member"},
				}
			}
		}

		_, err = tx.Exec(d.Context, "UPDATE team_members SET mentionable = $1 WHERE team_id = $2 AND user_id = $3", *payload.Mentionable, teamId, userId)

		if err != nil {
			state.Logger.Error("Error updating mentionable", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.DataHolder != nil {
		if !perms.HasRaw("global." + teams.PermissionOwner) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "Only global owners can set a data holder"},
			}
		}

		// Ensure that if dataholder is false, there is another dataholder
		if !*payload.DataHolder {
			var dataHolderCount int

			err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND data_holder = $2 AND user_id != $3", teamId, true, userId).Scan(&dataHolderCount)

			if err != nil {
				state.Logger.Error("Error getting data holder count", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			if dataHolderCount == 0 {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "There needs to be one other data holder before you can remove someone from data holder"},
				}
			}
		}

		// Set data holder
		_, err = tx.Exec(d.Context, "UPDATE team_members SET data_holder = $1 WHERE team_id = $2 AND user_id = $3", *payload.DataHolder, teamId, userId)

		if err != nil {
			state.Logger.Error("Error updating data holder", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error committing transaction", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
