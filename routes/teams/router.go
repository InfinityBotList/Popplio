package teams

import (
	"popplio/api"
	"popplio/routes/teams/endpoints/create_team"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

const tagName = "Tickets + Transcripts"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our ticketting and transcripts system"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/users/{id}/teams",
		OpId:    "create_team",
		Method:  api.POST,
		Docs:    create_team.Docs,
		Handler: create_team.Route,
		Auth: []api.AuthType{
			{
				Type:   types.TargetTypeUser,
				URLVar: "id",
			},
		},
	}.Route(r)
}
