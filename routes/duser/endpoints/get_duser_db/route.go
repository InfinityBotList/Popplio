package get_duser_db

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
		Summary:     "Get Discord User From Database",
		Description: "This endpoint will return a `DatabaseDiscordUser` object. This is useful for getting a user's (not bot and must be on db) avatar, username or discriminator quickly as it is stored in the database",
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The user's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.DatabaseDiscordUser{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	user, err := utils.GetDatabaseDiscordUser(d.Context, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	if !user.FoundInDB {
		return api.DefaultResponse(http.StatusNotFound)
	}

	return api.HttpResponse{
		Json: user,
	}
}
