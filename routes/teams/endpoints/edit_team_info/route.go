package edit_team_info

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type EditTeam struct {
	Name   string `json:"name" validate:"required,nonvulgar,min=3,max=32" msg:"Team name must be between 3 and 32 characters long"`
	Avatar string `json:"avatar" validate:"required,https" msg:"Avatar must be a valid HTTPS URL"`
}

var (
	compiledMessages = api.CompileValidationErrors(EditTeam{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edit Team Info",
		Description: "Edits a team. Returns a 204 on success.",
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
		Req:  EditTeam{},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

	var payload EditTeam

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	// Check team
	var count int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", teamId).Scan(&count)

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

	// Ensure the manager has the 'Edit Team Information' permission
	if !mp.Has(teams.TeamPermissionEditTeamInfo) {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You don't have permission to edit team information", Error: true},
		}
	}

	// Update the team
	_, err = state.Pool.Exec(d.Context, "UPDATE teams SET name = $1, avatar = $2 WHERE id = $3", payload.Name, payload.Avatar, teamId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
