package edit_team_member_permissions

import (
	"errors"
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

	"github.com/go-chi/chi/v5"
	"golang.org/x/exp/slices"
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

	// Ensure manager has perms to edit member permissions etc.
	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

	if err != nil {
		state.Logger.Error(err)
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

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

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
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Check if manager has all perms they are trying to add
	for _, perm := range payload.Add {
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
			_, err = tx.Exec(d.Context, "UPDATE team_members SET flags = array_append(flags, $1) WHERE team_id = $2 AND user_id = $3", perm, teamId, userId)

			if err != nil {
				state.Logger.Error(err)
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}
		}
	}

	for _, perm := range payload.Remove {
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

		if slices.Contains(oldPerms, perm) {
			// Remove the perm from the old perms
			_, err = tx.Exec(d.Context, "UPDATE team_members SET flags = array_remove(flags, $1) WHERE team_id = $2 AND user_id = $3", perm, teamId, userId)

			if err != nil {
				state.Logger.Error(err)
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}
		}
	}

	// Ensure that if perms includes owner, that there is at least one other owner
	if slices.Contains(payload.Remove, "global."+teams.PermissionOwner) {
		var ownerCount int

		err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND flags && $2", teamId, []string{teams.PermissionOwner}).Scan(&ownerCount)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if ownerCount == 0 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "There needs to be one other owner before you can remove yourself from owner"},
			}
		}
	}

	// Try to fix permissions in case any removals etc.
	var flags []string

	err = tx.QueryRow(d.Context, "SELECT flags FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId).Scan(&flags)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	flags = teams.NewPermMan(flags).Perms()

	_, err = tx.Exec(d.Context, "UPDATE team_members SET flags = $1 WHERE team_id = $2 AND user_id = $3", flags, teamId, userId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
