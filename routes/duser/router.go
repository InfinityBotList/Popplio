package duser

import (
	"popplio/api"
	"popplio/routes/duser/endpoints/clear_duser"
	"popplio/routes/duser/endpoints/get_duser"

	"github.com/go-chi/chi/v5"
)

const tagName = "Discord User"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our discord user system"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/_duser/{id}", func(r chi.Router) {
		api.Route{
			Pattern: "/",
			Method:  api.GET,
			Docs:    get_duser.Docs,
			Handler: get_duser.Route,
		}.Route(r)

		api.Route{
			Pattern: "/clear",
			Method:  api.GET,
			Docs:    clear_duser.Docs,
			Handler: clear_duser.Route,
		}.Route(r)
	})
}
