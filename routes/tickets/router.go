package tickets

import (
	"popplio/routes/tickets/endpoints/get_ticket"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Tickets"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to tickets on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/tickets/{id}",
		OpId:    "get_ticket",
		Method:  uapi.GET,
		Docs:    get_ticket.Docs,
		Handler: get_ticket.Route,
	}.Route(r)
}
