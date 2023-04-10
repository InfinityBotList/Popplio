package delete_team_member

import (
	"net/http"
	"popplio/api"
	"popplio/routes/teams/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Team Member",
		Description: "Deletes a member from the team. Users can always delete themselves. Returns a 204 on success",
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
				Description: "Member ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var teamId = chi.URLParam(r, "tid")
	var userId = chi.URLParam(r, "mid")

	// Convert ID to UUID
	if !utils.IsValidUUID(teamId) {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var count int

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", teamId).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Ensure manager is a member of the team
	var managerCount int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, d.Auth.ID).Scan(&managerCount)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if managerCount == 0 {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You are not a member of this team", Error: true},
		}
	}

	// Ensure user is a member of the team
	var userCount int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId).Scan(&userCount)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if userCount == 0 {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "User is not a member of this team", Error: true},
		}
	}

	// Get the manager's permissions
	var managerPerms []teams.TeamPermission
	err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, d.Auth.ID).Scan(&managerPerms)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	mp := teams.NewPermissionManager(managerPerms)

	if d.Auth.ID != userId {
		// Ensure that all permissions
		if !mp.Has(teams.TeamPermissionRemoveTeamMembers) {
			return api.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to remove team members", Error: true},
			}
		}

		// Get the old permissions of the user
		var oldPerms []teams.TeamPermission

		err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId).Scan(&oldPerms)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		// if the user can remove all permissions of the target, then only can deleting the team member occur
		_, err = assets.CheckPerms(managerPerms, oldPerms, []teams.TeamPermission{})

		if err != nil {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: err.Error(), Error: true},
			}
		}
	}

	// Ensure that if perms includes owner, that there is at least one other owner
	if mp.Has(teams.TeamPermissionOwner) {
		var ownerCount int

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id != $2 AND perms && $3", teamId, userId, []teams.TeamPermission{teams.TeamPermissionOwner}).Scan(&ownerCount)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		if ownerCount == 0 {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "There needs to be one other owner before you can remove yourself from owner", Error: true},
			}
		}
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, userId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
