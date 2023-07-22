package public_service_directory

import (
	"net/http"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Public Service Directory",
		Description: "Returns a list of available public services",
		Resp:        types.ServiceDiscovery{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var srvs = types.ServiceDiscovery{
		Services: map[string]types.SDService{
			"htmlsanitize": {
				Url:         state.Config.Sites.HtmlSanitize,
				Description: "HTML->MD",
			},
		},
	}

	return uapi.HttpResponse{
		Json: srvs,
	}
}
