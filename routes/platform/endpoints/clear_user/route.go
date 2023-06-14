package clear_user

import (
	"net/http"

	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Clear Platform User Cache",
		Description: "This endpoint will clear the cache for a user id on a given platform. This is useful if the user's data has changes",
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The user's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
			{
				Name:        "platform",
				In:          "query",
				Description: "The platform to get the user from.",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")
	var platform = r.URL.Query().Get("platform")

	switch platform {
	case "discord":
		state.Redis.Del(d.Context, "uobj:"+id)

		// Delete from internal_user_cache
		state.Pool.Exec(d.Context, "DELETE FROM internal_user_cache WHERE id = $1", id)

		return uapi.HttpResponse{}
	default:
		return uapi.HttpResponse{
			Status: http.StatusUnsupportedMediaType,
			Json: types.ApiError{
				Error:   true,
				Message: "Unsupported platform. Only `discord` is supported at this time as a platform.",
			},
		}
	}
}
