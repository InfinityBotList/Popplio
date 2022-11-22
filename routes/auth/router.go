package auth

import (
	"popplio/api"
	"popplio/routes/auth/endpoints/get_authorize_info"

	"github.com/go-chi/chi/v5"
)

const tagName = "Login"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to authorization and login (if this ever becomes publicly usable)"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/authorize/info",
		Method:  api.GET,
		Docs:    get_authorize_info.Docs,
		Handler: get_authorize_info.Route,
	}.Route(r)
}
