package clear_duser

import (
	"net/http"
	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/_duser/{id}/clear",
		OpId:        "clear_duser",
		Summary:     "Clear Discord User Cache",
		Description: "This endpoint will clear the cache for a specific discord user. This is useful if you the user's data has changes",
		Tags:        []string{api.CurrentTag},
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
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id := chi.URLParam(r, "id")
	state.Redis.Del(d.Context, "uobj:"+id)
	return api.HttpResponse{
		Status: http.StatusOK,
		Data:   constants.Success,
	}
}
