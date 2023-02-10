package get_partners

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/partners"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Partners",
		Description: "Gets the official partners of the list",
		Resp:        partners.PartnerList{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Status: http.StatusOK,
		Json: partners.Partners,
	}
}
