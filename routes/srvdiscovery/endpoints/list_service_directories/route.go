package list_service_directories

import (
	"net/http"

	"popplio/srvdirectory"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "List Service Directory",
		Description: "List all service directories",
		Resp:        types.SDList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	dir := make([]string, 0, len(srvdirectory.Directory))

	for k := range srvdirectory.Directory {
		dir = append(dir, k)
	}

	return uapi.HttpResponse{
		Json: types.SDList{
			Services: dir,
		},
	}
}
