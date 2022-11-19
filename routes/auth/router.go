package auth

import (
	"popplio/api"
	"popplio/routes/auth/endpoints/create_login.go"
	"popplio/routes/auth/endpoints/get_authorize_info"

	"github.com/go-chi/chi/v5"
)

const tagName = "Login"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to authorization and login (if this ever becomes publicly usable)"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/authorize", func(r chi.Router) {
		api.Route{
			Pattern: "/info",
			Method:  api.GET,
			Docs:    get_authorize_info.Docs,
			Handler: get_authorize_info.Route,
		}.Route(r)

		api.Route{
			Pattern: "/",
			Method:  api.GET,
			Docs:    create_login.Docs,
			Handler: create_login.Route,
		}.Route(r)
	})
}
