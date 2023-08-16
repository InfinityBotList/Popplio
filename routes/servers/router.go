package servers

import (
	"popplio/routes/servers/endpoints/get_server"
	"popplio/routes/servers/endpoints/get_server_seo"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Servers"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to servers on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/servers/{id}",
		OpId:    "get_server",
		Method:  uapi.GET,
		Docs:    get_server.Docs,
		Handler: get_server.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/seo",
		OpId:    "get_server_seo",
		Method:  uapi.GET,
		Docs:    get_server_seo.Docs,
		Handler: get_server_seo.Route,
	}.Route(r)
}
