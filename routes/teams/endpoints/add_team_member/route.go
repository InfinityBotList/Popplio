package add_team_member

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/routes/teams/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type AddTeamMember struct {
	UserID string                 `json:"user_id" validate:"required" msg:"User ID must be a valid snowflake"`
	Perms  []teams.TeamPermission `json:"perms" validate:"required" msg:"Permissions must be a valid array of strings"`
}

var (
	compiledMessages = api.CompileValidationErrors(AddTeamMember{})
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
		Req:  AddTeamMember{},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

	var payload AddTeamMember

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

	var managerPerms []teams.TeamPermission

	err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, d.Auth.ID).Scan(&managerPerms)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	perms, err := assets.CheckPerms(managerPerms, payload.Perms)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error(), Error: true},
		}
	}

	_, err = state.Pool.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, perms) VALUES ($1, $2, $3)", teamId, payload.UserID, perms)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
