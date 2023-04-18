package get_premium_plans

import (
	"net/http"
	"popplio/payments"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

type PlanList struct {
	Plans []payments.Plan `json:"plans"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Premium Plans",
		Description: "Gets the current set of premium plans available.",
		Resp:        PlanList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: PlanList{
			Plans: payments.Plans,
		},
	}
}
