package get_user_bot_perms

import (
	"net/http"
	"strings"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

// A bot is a Discord bot that is on the infinitybotlist.

var (
	botColsArr = utils.GetCols(types.Bot{})
	botCols    = strings.Join(botColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Bot Perms",
		Description: "Returns the permissions a user has on a bot",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "The bots ID, vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserBotPerms{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	name := chi.URLParam(r, "id")

	// Resolve bot ID
	id, err := utils.ResolveBot(d.Context, name)

	if err != nil {
		state.Logger.Error("Resolve Error", err)
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

	return uapi.HttpResponse{
		Json: types.UserBotPerms{
			Perms: perms.Perms(),
		},
	}
}
