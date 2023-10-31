package tickets

import (
	"popplio/api"
	"popplio/routes/tickets/endpoints/get_ticket"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Tickets + Transcripts"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our ticketting and transcripts system"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/tickets/{id}",
		OpId:    "get_ticket",
		Method:  uapi.GET,
		Docs:    get_ticket.Docs,
		Handler: get_ticket.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
	}.Route(r)
}
