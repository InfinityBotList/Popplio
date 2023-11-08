package staff

import (
	"popplio/routes/staff/endpoints/get_app_list"
	"popplio/routes/staff/endpoints/manage_app"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const (
	tagName = "Staff"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "Staff-only IBL endpoints. Only usable from staff panel using panelapi credentials"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/staff/apps",
		OpId:    "get_app_list",
		Method:  uapi.GET,
		Docs:    get_app_list.Docs,
		Handler: get_app_list.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/apps/{app_id}",
		OpId:    "manage_app",
		Method:  uapi.PATCH,
		Docs:    manage_app.Docs,
		Handler: manage_app.Route,
	}.Route(r)
}
