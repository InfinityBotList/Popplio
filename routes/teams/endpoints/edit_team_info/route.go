package edit_team_info

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/events"
	"popplio/webhooks/teamhooks"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateEditTeam{})
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
		Req:  types.CreateEditTeam{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

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
			Json:   types.ApiError{Message: "You do not have permission to edit this team's information (name/avatar/mention)"},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Get current name and avatar
	var oldName, oldAvatar string

	err = tx.QueryRow(d.Context, "SELECT name, avatar FROM teams WHERE id = $1", teamId).Scan(&oldName, &oldAvatar)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Update the team
	_, err = tx.Exec(d.Context, "UPDATE teams SET name = $1, avatar = $2 WHERE id = $3", payload.Name, payload.Avatar, teamId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if payload.Short != nil {
		_, err = tx.Exec(d.Context, "UPDATE teams SET short = $1 WHERE id = $2", payload.Short, teamId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Banner != nil {
		_, err = tx.Exec(d.Context, "UPDATE teams SET banner = $1 WHERE id = $2", payload.Banner, teamId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.Tags != nil {
		_, err = tx.Exec(d.Context, "UPDATE teams SET tags = $1 WHERE id = $2", payload.Tags, teamId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.ExtraLinks != nil {
		err = validators.ValidateExtraLinks(*payload.ExtraLinks)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: err.Error()},
			}
		}

		_, err = tx.Exec(d.Context, "UPDATE teams SET extra_links = $1 WHERE id = $2", payload.ExtraLinks, teamId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = teamhooks.Send(teamhooks.With{
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
