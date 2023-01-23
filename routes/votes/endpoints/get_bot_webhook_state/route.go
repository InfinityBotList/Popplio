package get_bot_webhook_state

import (
	"net/http"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bot Webhook State",
		Description: "Returns whether or not the bot uses webhooks or REST for vote handling. **Requires authentication**",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.WebhookState{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id, err := utils.ResolveBot(d.Context, chi.URLParam(r, "id"))

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var webhook string

	err = state.Pool.QueryRow(d.Context, "SELECT webhook FROM bots WHERE bot_id = $1", id).Scan(&webhook)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: types.WebhookState{
			HTTP: webhook == "httpUser",
		},
	}
}
