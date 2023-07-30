package edit_team_info

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/webhooks/events"
	"popplio/webhooks/teamhooks"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type EditTeam struct {
	Name   string `json:"name" validate:"required,nonvulgar,min=3,max=32" msg:"Team name must be between 3 and 32 characters long"`
	Avatar string `json:"avatar" validate:"required,https" msg:"Avatar must be a valid HTTPS URL"`
}

var (
	compiledMessages = uapi.CompileValidationErrors(EditTeam{})
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

	var payload EditTeam

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

	// Ensure manager has perms to edit this team
	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !perms.Has("team", teams.PermissionEdit) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to edit this team's information (name/avatar)"},
		}
	}

	// Get current name and avatar
	var oldName, oldAvatar string

	err = state.Pool.QueryRow(d.Context, "SELECT name, avatar FROM teams WHERE id = $1", teamId).Scan(&oldName, &oldAvatar)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Update the team
	_, err = state.Pool.Exec(d.Context, "UPDATE teams SET name = $1, avatar = $2 WHERE id = $3", payload.Name, payload.Avatar, teamId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = teamhooks.Send(teamhooks.With[events.WebhookTeamEditData]{
		Data: events.WebhookTeamEditData{
			Name: events.Changeset[string]{
				Old: oldName,
				New: payload.Name,
			},
			Avatar: events.Changeset[string]{
				Old: oldAvatar,
				New: payload.Avatar,
			},
		},
		UserID: d.Auth.ID,
		TeamID: teamId,
	})

	if err != nil {
		state.Logger.Error(err)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
