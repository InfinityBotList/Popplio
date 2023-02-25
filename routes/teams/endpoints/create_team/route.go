package create_team

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"

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
	var teamId pgtype.UUID
	err = state.Pool.QueryRow(d.Context, "INSERT INTO teams (owner, name, avatar) VALUES ($1, $2, $3) RETURNING id", d.Auth.ID, payload.Name, payload.Avatar).Scan(&teamId)

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
