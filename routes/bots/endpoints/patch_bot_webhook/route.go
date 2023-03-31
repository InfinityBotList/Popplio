package patch_bot_webhook

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	docs "github.com/infinitybotlist/doclib"
)

type PatchBotWebhook struct {
	WebhookURL    string `json:"webhook_url" validate:"httporhttps" msg:"Webhook URL is required and must be HTTP/HTTPS"`
	WebhookSecret string `json:"webhook_secret"`
	WebhooksV2    bool   `json:"webhooks_v2"`
	Clear         bool   `json:"clear"`
}

var compiledMessages = api.CompileValidationErrors(PatchBotWebhook{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch Bot Webhook",
		Description: "Edits the webhook information for a bot. You must have 'Edit Bot Webhooks' in the team if the bot is in a team. Set `clear` to `true` to clear webhook settings. Returns 204 on success",
		Req:         PatchBotWebhook{},
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "bid")

	// Resolve bot ID
	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has(teams.TeamPermissionEditBotWebhooks) {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to edit bot webhooks", Error: true},
		}
	}

	// Read payload from body
	var payload PatchBotWebhook

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	// Update the bot
	if payload.Clear {
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET webhook = NULL, web_auth = NULL WHERE bot_id = $1", id)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	} else {
		if payload.WebhookURL != "" {
			_, err = state.Pool.Exec(d.Context, "UPDATE bots SET webhook = $1 WHERE bot_id = $2", payload.WebhookURL, id)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}
		}

		if payload.WebhookSecret != "" {
			_, err = state.Pool.Exec(d.Context, "UPDATE bots SET web_auth = $1 WHERE bot_id = $2", payload.WebhookSecret, id)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}
		}

		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET webhooks_v2 = $1 WHERE bot_id = $2", payload.WebhooksV2, id)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return api.DefaultResponse(http.StatusNoContent)
}
