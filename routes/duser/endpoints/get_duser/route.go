package get_duser

import (
	"net/http"

	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/docs"
	"github.com/infinitybotlist/popplio/state"
	"github.com/infinitybotlist/popplio/types"
	"github.com/infinitybotlist/popplio/utils"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/_duser/{id}",
		OpId:        "get_duser",
		Summary:     "Get Discord User",
		Description: "This endpoint will return a discord user object. This is useful for getting a user's avatar, username or discriminator etc.",
		Tags:        []string{api.CurrentTag},
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The user's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.DiscordUser{},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	user, err := utils.GetDiscordUser(id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	return api.HttpResponse{
		Status: http.StatusOK,
		Json:   user,
	}
}
