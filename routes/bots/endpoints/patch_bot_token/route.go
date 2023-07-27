package patch_bot_token

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch Bot Webhook",
		Description: "Resets a bots token. You must have 'Reset Bot Tokens' in the team if the bot is in a team. Returns the new token on success",
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

	if !perms.Has(teams.TeamPermissionResetBotTokens) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to reset this bots token"},
		}
	}

	token := crypto.RandString(128)

	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET api_token = $1 WHERE bot_id = $2", token, id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   types.UserLogin{UserID: id, Token: token},
	}
}
