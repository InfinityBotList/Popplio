package get_premium_plans

import (
	"net/http"
	"popplio/api"
	"popplio/payments"

	docs "github.com/infinitybotlist/doclib"
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Json: PlanList{
			Plans: payments.Plans,
		},
	}
}
