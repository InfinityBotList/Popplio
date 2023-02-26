package create_team

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/teams"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateTeam struct {
	Name   string `json:"name" validate:"required,nonvulgar,min=3,max=32" msg:"Team name must be between 3 and 32 characters long"`
	Avatar string `json:"avatar" validate:"required,https" msg:"Avatar must be a valid HTTPS URL"`
}

type CreateTeamResponse struct {
	TeamID pgtype.UUID `json:"team_id"`
}

var (
	compiledMessages = api.CompileValidationErrors(CreateTeam{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Team",
		Description: "Creates a team. Returns a 206 with the team ID on success.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  CreateTeam{},
		Resp: CreateTeamResponse{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var payload CreateTeam

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

	// Create the team
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	var teamId pgtype.UUID
	err = tx.QueryRow(d.Context, "INSERT INTO teams (name, avatar) VALUES ($1, $2) RETURNING id", payload.Name, payload.Avatar).Scan(&teamId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Add the user to the team
	_, err = tx.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, perms) VALUES ($1, $2, $3)", teamId, d.Auth.ID, []teams.TeamPermission{teams.TeamPermissionOwner})

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Status: http.StatusPartialContent,
		Json: CreateTeamResponse{
			TeamID: teamId,
		},
	}
}
