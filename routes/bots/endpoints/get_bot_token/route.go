package get_bot_token

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bot Webhook",
		Description: "Gets the API token of a bot. You must have 'View Existing Bot Tokens' in the team if the bot is in a team.",
		Resp:        types.UserLogin{},
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
	name := chi.URLParam(r, "bid")

	// Resolve bot ID
	id, err := utils.ResolveBot(d.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has(teams.TeamPermissionViewExistingBotTokens) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to view existing tokens of this bot"},
		}
	}

	var token string

	err = state.Pool.QueryRow(d.Context, "SELECT api_token FROM bots WHERE bot_id = $1", id).Scan(&token)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   types.UserLogin{UserID: id, Token: token},
	}
}
