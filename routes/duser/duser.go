package duser

import (
	"net/http"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

const tagName = "Discord User"

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
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				var id = chi.URLParam(r, "id")

				user, err := utils.GetDiscordUser(id)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				resp <- types.HttpResponse{
					Status: http.StatusOK,
					Json:   user,
				}
			}()

			utils.Respond(ctx, w, resp)
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
			ctx := r.Context()
			resp := make(chan types.HttpResponse)

			go func() {
				id := chi.URLParam(r, "id")
				state.Redis.Del(ctx, "uobj:"+id)
				resp <- types.HttpResponse{
					Status: http.StatusOK,
					Data:   constants.Success,
				}
			}()

			utils.Respond(ctx, w, resp)
		})
	})
}
