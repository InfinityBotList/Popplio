package get_duser

import (
	"net/http"

	"popplio/config"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Discord User",
		Description: "Deprecated, use Get Platform User instead.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The user's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: dovewing.PlatformUser{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if config.CurrentEnv == config.CurrentEnvStaging {
		return uapi.HttpResponse{
			Status: http.StatusUnsupportedMediaType,
			Json: types.ApiError{
				Error:   true,
				Message: "Deprecated endpoint, please use Platform APIs instead",
				Context: map[string]string{
					"try": "https://reedwhisker.infinitybots.gg",
				},
			},
		}
	}

	var id = chi.URLParam(r, "id")

	user, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	return uapi.HttpResponse{
		Json: user,
	}
}
