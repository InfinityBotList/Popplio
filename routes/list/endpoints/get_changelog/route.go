package get_changelog

import (
	"net/http"
	"popplio/changelogs"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Changelog",
		Description: "Gets the changelog of the list",
		Resp:        types.Changelog{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   changelogs.Changelog,
	}
}
