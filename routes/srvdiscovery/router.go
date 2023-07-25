package list

import (
	"popplio/routes/srvdiscovery/endpoints/list_service_directories"
	"popplio/routes/srvdiscovery/endpoints/service_directory"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Service Discovery"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to service discovery on the list."
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/services/@directories",
		OpId:    "list_service_directories",
		Method:  uapi.GET,
		Docs:    list_service_directories.Docs,
		Handler: list_service_directories.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/services/{directory}",
		OpId:    "service_directory",
		Method:  uapi.GET,
		Docs:    service_directory.Docs,
		Handler: service_directory.Route,
	}.Route(r)
}
