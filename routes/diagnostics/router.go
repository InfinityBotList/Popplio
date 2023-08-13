package diagnostics

import (
	"popplio/routes/diagnostics/endpoints/failure_management"
	"popplio/routes/diagnostics/endpoints/ping"
	"popplio/routes/diagnostics/endpoints/ping_head"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Diagnostics"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints allow diagnosing potential issues within our API or within any frontend application."
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/",
		OpId:    "ping",
		Method:  uapi.GET,
		Docs:    ping.Docs,
		Handler: ping.Route,
		Setup:   ping.Setup,
	}.Route(r)

	uapi.Route{
		Pattern: "/",
		OpId:    "ping",
		Method:  uapi.HEAD,
		Docs:    ping_head.Docs,
		Handler: ping_head.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/failure-management",
		OpId:    "failure_management",
		Method:  uapi.POST,
		Docs:    failure_management.Docs,
		Handler: failure_management.Route,
	}.Route(r)
}
