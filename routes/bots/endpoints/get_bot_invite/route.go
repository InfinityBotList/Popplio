package get_bot_invite

import (
	"net/http"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
)

// A bot is a Discord bot that is on the infinitybotlist.

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary: "Get Bot Invite",
		Description: `
Gets a bot invite by id or name

`,
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	id, err := utils.ResolveBot(d.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var invite string
	err = state.Pool.QueryRow(d.Context, "SELECT invite FROM bots WHERE bot_id = $1", id).Scan(&invite)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if d.IsClient {
		// Update clicks
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET invite_clicks = invite_clicks + 1 WHERE bot_id = $1", id)

		if err != nil {
			state.Logger.Error(err)
			return api.HttpResponse{
				Status: http.StatusInternalServerError,
				Json: types.ApiError{
					Message: "Failed to update invite clicks",
					Error:   true,
				},
			}
		}
	}

	return api.HttpResponse{
		Json: types.Invite{
			Invite: invite,
		},
	}
}
