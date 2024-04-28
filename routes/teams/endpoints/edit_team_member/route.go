package edit_team_member

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

	// Check if user+manager are on the team before doing anything else
	var count int

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND (user_id = $2 OR user_id = $3)", teamId, d.Auth.ID, userId).Scan(&count)

	if err != nil {
		state.Logger.Error("Error checking if user is on team", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Error checking if user is on team: " + err.Error()},
		}
	}

	if d.Auth.ID != userId {
		if count != 2 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Either the manager or the user is not on this team"},
			}
		}
		// count == 1 if the user is the manager
	} else if count != 1 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "User is not on this team"},
		}
	}

	var payload types.EditTeamMember

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Get team permissions for manager
	managerPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

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

	if payload.Perms != nil {
		if !kittycat.HasPerm(managerPerms, kittycat.Build("team_member", teams.PermissionEdit)) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to edit team members"},
			}
		}

		// Get the old permissions of the user
		currentUserPerms, err := teams.GetEntityPerms(d.Context, userId, "team", teamId)

		if err != nil {
			state.Logger.Error("Error getting old perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Error getting old perms: " + err.Error()},
			}
		}

		// Perform initial checks
		for _, perm := range *payload.Perms {
			if !teams.IsValidPerm(perm) {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Invalid permission: " + perm},
				}
			}
		}

		// Resolve the permissions
		//
		// Right now, we use perm overrides for this
		// as we do not have a hierarchy system yet
		newPermsResolved := kittycat.StaffPermissions{
			PermOverrides: *payload.Perms,
		}.Resolve()

		// First ensure that the manager can set these permissions
		if err = kittycat.CheckPatchChanges(managerPerms, currentUserPerms, newPermsResolved); err != nil {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to set these permissions: " + err.Error()},
			}
		}

		if !kittycat.HasPerm(newPermsResolved, kittycat.Build("global", teams.PermissionOwner)) && kittycat.HasPerm(currentUserPerms, kittycat.Build("global", teams.PermissionOwner)) {
			// Ensure that if perm is owner, then there is another owner
			var ownerCount int

			err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND flags && $2", teamId, []string{kittycat.Build("global", teams.PermissionOwner)}).Scan(&ownerCount)

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

		_, err = tx.Exec(d.Context, "UPDATE team_members SET flags = $1 WHERE team_id = $2 AND user_id = $3", payload.Perms, teamId, userId)

		if err != nil {
			state.Logger.Error("Error updating perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Mentionable != nil {
		// All members can update their own mentionable status
		if d.Auth.ID != userId {
			if !kittycat.HasPerm(managerPerms, kittycat.Build("team_member", teams.PermissionEdit)) {
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
		if !kittycat.HasPerm(managerPerms, kittycat.Build("global", teams.PermissionOwner)) {
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
