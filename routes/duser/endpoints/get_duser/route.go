package get_duser

import (
	"net/http"

	"popplio/api"
	"popplio/state"

	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Discord User",
		Description: "This endpoint will return a discord user object. This is useful for getting a user's avatar, username or discriminator etc.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The user's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: dovewing.DiscordUser{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	user, err := dovewing.GetDiscordUser(d.Context, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	return api.HttpResponse{
		Json: user,
	}
}
