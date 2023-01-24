package duser

import (
	"popplio/api"
	"popplio/routes/duser/endpoints/clear_duser"
	"popplio/routes/duser/endpoints/get_duser"
	"popplio/routes/duser/endpoints/get_duser_db"

	"github.com/go-chi/chi/v5"
)

const tagName = "Discord User"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our discord user system"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/_duser/{id}",
		OpId:    "get_duser",
		Method:  api.GET,
		Docs:    get_duser.Docs,
		Handler: get_duser.Route,
	}.Route(r)

	api.Route{
		Pattern: "/_duser/{id}/db",
		OpId:    "get_duser",
		Method:  api.GET,
		Docs:    get_duser_db.Docs,
		Handler: get_duser_db.Route,
	}.Route(r)

	api.Route{
		Pattern: "/_duser/{id}/clear",
		OpId:    "clear_duser",
		Method:  api.GET,
		Docs:    clear_duser.Docs,
		Handler: clear_duser.Route,
	}.Route(r)
}
