package duser

import (
	"net/http"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"
)

const tagName = "Discord User"

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our discord user system"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/_duser/{id}", func(r chi.Router) {
		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/_duser/{id}",
			OpId:        "get_duser",
			Summary:     "Get Discord User",
			Description: "This endpoint will return a discord user object. This is useful for getting a user's avatar, username or discriminator etc.",
			Tags:        []string{tagName},
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
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var id = chi.URLParam(r, "id")

			user, err := utils.GetDiscordUser(id)

			if err != nil {
				state.Logger.Error(err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			bytes, err := json.Marshal(user)

			if err != nil {
				state.Logger.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		})
		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/_duser/{id}/clear",
			OpId:        "clear_duser",
			Summary:     "Clear Discord User Cache",
			Description: "This endpoint will clear the cache for a specific discord user. This is useful if you the user's data has changes",
			Tags:        []string{tagName},
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "The ID of the user to clear the cache for",
					In:          "path",
					Required:    true,
					Schema:      docs.IdSchema,
				},
			},
			Resp: types.ApiError{},
		})
		r.Get("/clear", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			state.Redis.Del(state.Context, "uobj:"+id)
			w.Write([]byte(constants.Success))
		})
	})
}
