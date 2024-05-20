package add_team_member

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
		Summary:     "Add Team Member",
		Description: "Adds a member to a team. Returns a 204 on success",
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
		Req:  types.AddTeamMember{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

	var payload types.AddTeamMember

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Ensure manager has perms to edit member permissions etc.
	managerPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !kittycat.HasPerm(managerPerms, kittycat.Permission{Namespace: "team_member", Perm: teams.PermissionAdd}) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to add members to this team"},
		}
	}

	for _, perm := range payload.Perms {
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
		PermOverrides: kittycat.PFSS(payload.Perms),
	}.Resolve()

	// Check if the manager has perms to give all permissions in newPermsResolved
	//
	// This is equivalent to going from no perms to the selected permset
	if err = kittycat.CheckPatchChanges(managerPerms, []kittycat.Permission{}, newPermsResolved); err != nil {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to give out permissions: " + err.Error()},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error starting transaction", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Check if user exists on IBL
	var userExists bool

	err = tx.QueryRow(d.Context, "SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", payload.UserID).Scan(&userExists)

	if err != nil {
		state.Logger.Error("Error checking if user exists", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !userExists {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "User must login here at least once before you can add them"},
		}
	}

	// Check that they aren't already a member
	var memberExists bool

	err = tx.QueryRow(d.Context, "SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2)", teamId, payload.UserID).Scan(&memberExists)

	if err != nil {
		state.Logger.Error("Error checking if user is already a member", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if memberExists {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "User is already a member of this team"},
		}
	}

	_, err = tx.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, flags) VALUES ($1, $2, $3)", teamId, payload.UserID, payload.Perms)

	if err != nil {
		state.Logger.Error("Error adding member", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error committing transaction", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
