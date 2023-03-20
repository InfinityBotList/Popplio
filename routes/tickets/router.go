package tickets

import (
	"popplio/api"
	"popplio/routes/tickets/endpoints/get_all_tickets"
	"popplio/routes/tickets/endpoints/get_ticket"

	"github.com/go-chi/chi/v5"
)

const tagName = "Tickets + Transcripts"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our ticketting and transcripts system"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/tickets/all",
		OpId:    "get_all_tickets",
		Method:  api.GET,
		Docs:    get_all_tickets.Docs,
		Handler: get_all_tickets.Route,
		Auth: []api.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/tickets/{id}",
		OpId:    "get_ticket",
		Method:  api.GET,
		Docs:    get_ticket.Docs,
		Handler: get_ticket.Route,
		Auth: []api.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
	}.Route(r)
}
