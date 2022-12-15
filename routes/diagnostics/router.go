package diagnostics

import (
	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/routes/diagnostics/endpoints/ping"

	"github.com/go-chi/chi/v5"
)

const tagName = "Diagnostics"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints allow diagnosing potential connection issues to our API."
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/",
		OpId:    "ping",
		Method:  api.GET,
		Docs:    ping.Docs,
		Handler: ping.Route,
		Setup:   ping.Setup,
	}.Route(r)
}
