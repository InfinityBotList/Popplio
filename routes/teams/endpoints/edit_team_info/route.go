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

	// Check team
	var count int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", teamId).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	// Ensure manager is a member of the team
	var managerCount int

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, d.Auth.ID).Scan(&managerCount)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if managerCount == 0 {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You are not a member of this team", Error: true},
		}
	}

	// Get the manager's permissions
	var managerPerms []types.TeamPermission
	err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", teamId, d.Auth.ID).Scan(&managerPerms)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	mp := teams.NewPermissionManager(managerPerms)

	// Ensure the manager has the 'Edit Team Information' permission
	if !mp.Has(teams.TeamPermissionEditTeamInfo) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You don't have permission to edit team information", Error: true},
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
