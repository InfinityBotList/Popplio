package duser

import (
	"popplio/routes/duser/endpoints/clear_duser"
	"popplio/routes/duser/endpoints/get_duser"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Discord User"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our discord user system"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/_duser/{id}",
		OpId:    "get_duser",
		Method:  uapi.GET,
		Docs:    get_duser.Docs,
		Handler: get_duser.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/_duser/{id}",
		OpId:    "clear_duser",
		Method:  uapi.DELETE,
		Docs:    clear_duser.Docs,
		Handler: clear_duser.Route,
	}.Route(r)
}
