package create_team

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateTeam{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Team",
		Description: "Creates a team. Returns a 201 with the team ID on success.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.CreateTeam{},
		Resp: types.CreateTeamResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.CreateTeam

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	// Create the team
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	var teamId pgtype.UUID
	err = tx.QueryRow(d.Context, "INSERT INTO teams (name, avatar) VALUES ($1, $2) RETURNING id", payload.Name, payload.Avatar).Scan(&teamId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Add the user to the team
	_, err = tx.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, perms) VALUES ($1, $2, $3)", teamId, d.Auth.ID, []string{teams.PermissionOwner})

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json: types.CreateTeamResponse{
			TeamID: teamId,
		},
	}
}
