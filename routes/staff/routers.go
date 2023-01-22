package staff

import (
	"popplio/api"
	"popplio/routes/staff/endpoints/get_staff_onboard_code"
	"popplio/routes/staff/endpoints/get_staff_onboard_data"

	"github.com/go-chi/chi/v5"
)

const tagName = "Staff"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to staff on IBL and are mostly internal"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/users/{id}/staff-onboard-code",
		OpId:    "get_staff_onboard_code",
		Method:  api.GET,
		Docs:    get_staff_onboard_code.Docs,
		Handler: get_staff_onboard_code.Route,
	}.Route(r)

	api.Route{
		Pattern: "/staff-onboard-data/{id}",
		OpId:    "get_staff_onboard_data",
		Method:  api.GET,
		Docs:    get_staff_onboard_data.Docs,
		Handler: get_staff_onboard_data.Route,
	}.Route(r)
}
