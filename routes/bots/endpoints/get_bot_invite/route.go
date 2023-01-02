package get_bot_invite

import (
	"net/http"
	"strings"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

// A bot is a Discord bot that is on the infinitybotlist.

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:  "GET",
		Path:    "/bots/{id}/invite",
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

	name = strings.ToLower(name)

	if name == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	// First check count so we can avoid expensive DB calls
	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE "+constants.ResolveBotSQL, name).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	if count > 1 {
		// Delete one of the bots
		_, err := state.Pool.Exec(d.Context, "DELETE FROM bots WHERE "+constants.ResolveBotSQL+" LIMIT 1", name)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	var botId int64
	var invite string
	err = state.Pool.QueryRow(d.Context, "SELECT bot_id, invite FROM bots WHERE "+constants.ResolveBotSQL, name).Scan(&botId, &invite)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if d.IsClient {
		// Update clicks
		_, err = state.Pool.Exec(state.Context, "UPDATE bots SET invite_clicks = invite_clicks + 1 WHERE bot_id = $1", botId)

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
