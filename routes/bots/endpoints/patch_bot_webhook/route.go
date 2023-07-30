package patch_bot_webhook

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch Bot Webhook",
		Description: "Edits the webhook information for a bot. You must have 'Edit Bot Webhooks' in the team if the bot is in a team. Set `clear` to `true` to clear webhook settings. Returns 204 on success",
		Req:         types.PatchBotWebhook{},
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
				Name:        "bid",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "bid")

	// Validate for current team
	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "bot", id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has("bot", teams.PermissionEditWebhooks) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to edit this bots webhook"},
		}
	}

	// Read payload from body
	var payload types.PatchBotWebhook

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	// Clear cache
	utils.ClearBotCache(d.Context, id)

	// Update the bot
	if payload.Clear {
		_, err = tx.Exec(d.Context, "UPDATE bots SET webhook = NULL, web_auth = NULL WHERE bot_id = $1", id)

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

		_, err = tx.Exec(d.Context, "UPDATE bots SET webhook = $1 WHERE bot_id = $2", payload.WebhookURL, id)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if payload.WebhookSecret != "" {
		_, err = tx.Exec(d.Context, "UPDATE bots SET web_auth = $1 WHERE bot_id = $2", payload.WebhookSecret, id)

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

	return uapi.DefaultResponse(http.StatusNoContent)
}
