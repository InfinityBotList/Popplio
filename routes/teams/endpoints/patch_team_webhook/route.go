package patch_team_webhook

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch Team Webhook",
		Description: "Edits the webhook information for a team. You must have 'Edit Team Webhooks' in the team. Set `clear` to `true` to clear webhook settings. Returns 204 on success",
		Req:         types.PatchTeamWebhook{},
		Resp:        types.ApiError{},
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
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	teamId := chi.URLParam(r, "tid")

	// Check team
	var count int

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", teamId).Scan(&count)

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
			Json:   types.ApiError{Message: "You are not a member of this team"},
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

	if !mp.Has(teams.TeamPermissionEditTeamWebhooks) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to edit team webhooks"},
		}
	}

	// Read payload from body
	var payload types.PatchTeamWebhook

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Update the team
	if payload.Clear {
		_, err = state.Pool.Exec(d.Context, "UPDATE teams SET webhook = NULL, web_auth = NULL WHERE id = $1", teamId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		return uapi.DefaultResponse(http.StatusNoContent)
	}

	if payload.WebhookURL != "" {
		if !(strings.HasPrefix(payload.WebhookURL, "http://") || strings.HasPrefix(payload.WebhookURL, "https://")) {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Webhook URL must start with http:// or https://"},
			}
		}

		_, err = state.Pool.Exec(d.Context, "UPDATE teams SET webhook = $1 WHERE id = $2", payload.WebhookURL, teamId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.WebhookSecret != "" {
		_, err = state.Pool.Exec(d.Context, "UPDATE teams SET web_auth = $1 WHERE id = $2", payload.WebhookSecret, teamId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
