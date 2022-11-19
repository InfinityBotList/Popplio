package diagnostics

import (
	"popplio/api"
	"popplio/routes/diagnostics/endpoints/ping"

	"github.com/go-chi/chi/v5"
)

const (
	tagName    = "Diagnostics"
	docsSite   = "https://spider.infinitybotlist.com/docs"
	mainSite   = "https://infinitybotlist.com"
	statusPage = "https://status.botlist.site"
	apiBot     = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints allow diagnosing potential connection issues to our API."
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/",
		Method:  api.GET,
		Docs:    ping.Docs,
		Handler: ping.Route,
		Setup:   ping.Setup,
	}.Route(r)
}
