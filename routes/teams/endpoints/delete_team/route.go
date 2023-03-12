package delete_team

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

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

	// Get the manager's permissions
	var managerPerms []teams.TeamPermission
	err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, d.Auth.ID).Scan(&managerPerms)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	mp := teams.NewPermissionManager(managerPerms)

	// Ensure the manager has the 'Owner' permission
	if !mp.Has(teams.TeamPermissionOwner) {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "Only owners can delete teams", Error: true},
		}
	}

	var botCount int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE team_owner = $1", teamId).Scan(&botCount)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if botCount > 0 {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "You cannot delete a team with bots on it", Error: true},
		}
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM teams WHERE id = $1", teamId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
