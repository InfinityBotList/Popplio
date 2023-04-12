package clear_duser

import (
	"net/http"
	"time"

	"popplio/api"
	"popplio/ratelimit"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Clear Discord User Cache",
		Description: "This endpoint will clear the cache for a specific discord user. This is useful if you the user's data has changes",
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
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id := chi.URLParam(r, "id")
	state.Redis.Del(d.Context, "uobj:"+id)

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 3,
		Bucket:      "clear_duser",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	// Delete from internal_user_cache
	state.Pool.Exec(d.Context, "DELETE FROM internal_user_cache WHERE id = $1", id)

	return api.HttpResponse{
		Json: types.ApiError{
			Error:   false,
			Message: "Successfully cleared cache",
		},
	}
}
