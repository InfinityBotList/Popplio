package get_premium_plans

import (
	"net/http"
	"popplio/routes/payments/assets"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Premium Plans",
		Description: "Gets the current set of premium plans available.",
		Resp:        types.PlanList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: types.PlanList{
			Plans: assets.Plans,
		},
	}
}
