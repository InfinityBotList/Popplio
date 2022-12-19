package tickets

import (
	"popplio/api"
	"popplio/routes/tickets/endpoints/get_ticket"

	"github.com/go-chi/chi/v5"
)

const tagName = "Tickets + Transcripts"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our ticketting and transcripts system"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/tickets", func(r chi.Router) {
		api.Route{
			Pattern: "/{id}",
			OpId:    "get_ticket",
			Method:  api.GET,
			Docs:    get_ticket.Docs,
			Handler: get_ticket.Route,
		}.Route(r)
	})
}
