package get_apps_meta

import (
	"net/http"
	"popplio/api"
	"popplio/apps"
	"popplio/docs"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/apps/meta",
		OpId:        "get_apps_meta",
		Summary:     "Get Apps Meta",
		Description: "Gets the current applications metadata. Returns a ``AppMeta`` object. See schema for more info.",
		Tags:        []string{api.CurrentTag},
		Resp:        apps.AppMeta{},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Json: apps.AppMeta{
			Positions: apps.Apps,
			Stable:    apps.Stable,
		},
	}
}
