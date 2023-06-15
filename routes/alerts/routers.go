package alerts

import (
	"popplio/api"
	"popplio/routes/alerts/endpoints/get_featured_user_alerts"
	"popplio/routes/alerts/endpoints/get_user_alerts"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Alerts"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to user alerts on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/users/{id}/alerts",
		OpId:    "get_user_alerts",
		Method:  uapi.GET,
		Docs:    get_user_alerts.Docs,
		Handler: get_user_alerts.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/alerts/@featured",
		OpId:    "get_featured_user_alerts",
		Method:  uapi.GET,
		Docs:    get_featured_user_alerts.Docs,
		Handler: get_featured_user_alerts.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
