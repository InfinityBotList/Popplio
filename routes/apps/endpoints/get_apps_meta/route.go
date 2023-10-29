package get_apps_meta

import (
	"net/http"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Apps Meta",
		Description: "Gets the current applications metadata. Returns a ``AppMeta`` object. See schema for more info.",
		Resp:        types.AppMeta{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: http.StatusServiceUnavailable,
		Json: types.ApiError{
			Message: "Applications are currently disabled due to an ongoing rewrite.",
		},
	}
}
