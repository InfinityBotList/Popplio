package duser

import (
	"popplio/routes/duser/endpoints/get_duser"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Discord User (Deprecated)"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "This API class is deprecated, use `Platform` instead"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/_duser/{id}",
		OpId:    "get_duser",
		Method:  uapi.GET,
		Docs:    get_duser.Docs,
		Handler: get_duser.Route,
	}.Route(r)
}
