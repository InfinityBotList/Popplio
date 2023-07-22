package staff_service_directory

import (
	"net/http"

	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

var srvs = types.ServiceDiscovery{
	Services: map[string]types.SDService{
		"arcadia": {
			Url:         "https://rpc.infinitybots.gg",
			Description: "Staff RPC API",
		},
		"persepolis": {
			Url:         "https://persepolis.infinitybots.gg",
			Description: "Responsible for handling onboarding of staff",
		},
		"ashfur": {
			Url:                "https://ashfur.infinitybots.gg",
			Description:        "Responsible for handling data aggregation (modcases) on MongoDB",
			PlannedMaintenance: true,
			NeedsStaging:       true,
		},
	},
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Staff Service Directory",
		Description: "Returns a list of available RPC services",
		Resp:        types.ServiceDiscovery{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: srvs,
	}
}
