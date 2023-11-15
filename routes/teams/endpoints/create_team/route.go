package create_team

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateEditTeam{})
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
		Req:  types.CreateEditTeam{},
		Resp: types.CreateTeamResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.CreateEditTeam

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

	var el = []types.Link{}

	if payload.ExtraLinks != nil {
		err = validators.ValidateExtraLinks(*payload.ExtraLinks)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: err.Error()},
			}
		}

		el = *payload.ExtraLinks
	}

	// Create the team
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error starting transaction", zap.Error(err), zap.String("user_id", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	var teamId pgtype.UUID
	err = tx.QueryRow(d.Context, "INSERT INTO teams (name, short, tags, extra_links) VALUES ($1, $2, $3, $4) RETURNING id", payload.Name, payload.Short, payload.Tags, el).Scan(&teamId)

	if err != nil {
		state.Logger.Error("Error creating team", zap.Error(err), zap.String("user_id", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Add the user to the team
	_, err = tx.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, flags, data_holder) VALUES ($1, $2, $3, true)", teamId, d.Auth.ID, []string{"global." + teams.PermissionOwner})

	if err != nil {
		state.Logger.Error("Error adding user to team", zap.Error(err), zap.String("user_id", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error committing transaction", zap.Error(err), zap.String("user_id", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json: types.CreateTeamResponse{
			TeamID: teamId,
		},
	}
}
