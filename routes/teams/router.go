package teams

import (
	"popplio/api"
	"popplio/routes/teams/endpoints/add_team_member"
	"popplio/routes/teams/endpoints/create_team"
	"popplio/routes/teams/endpoints/get_team"
	"popplio/routes/teams/endpoints/get_team_permissions"
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
		Pattern: "/teams/{id}/permissions",
		OpId:    "get_team_permissions",
		Method:  api.GET,
		Docs:    get_team_permissions.Docs,
		Handler: get_team_permissions.Route,
	}.Route(r)

	api.Route{
		Pattern: "/teams/{id}",
		OpId:    "get_team",
		Method:  api.GET,
		Docs:    get_team.Docs,
		Handler: get_team.Route,
	}.Route(r)

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

	api.Route{
		Pattern: "/users/{uid}/teams/{tid}/members",
		OpId:    "add_team_member",
		Method:  api.PUT,
		Docs:    add_team_member.Docs,
		Handler: add_team_member.Route,
		Auth: []api.AuthType{
			{
				Type:   types.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
