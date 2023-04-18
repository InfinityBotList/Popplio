package teams

import (
	"popplio/api"
	"popplio/routes/teams/endpoints/add_bot_to_team"
	"popplio/routes/teams/endpoints/add_team_member"
	"popplio/routes/teams/endpoints/create_team"
	"popplio/routes/teams/endpoints/delete_team"
	"popplio/routes/teams/endpoints/delete_team_member"
	"popplio/routes/teams/endpoints/edit_team_info"
	"popplio/routes/teams/endpoints/edit_team_member_permissions"
	"popplio/routes/teams/endpoints/get_team"
	"popplio/routes/teams/endpoints/get_team_permissions"
	"popplio/routes/teams/endpoints/get_team_seo"
	"popplio/routes/teams/endpoints/patch_bot_team"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Teams"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our teams system"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/teams/meta/permissions",
		OpId:    "get_team_permissions",
		Method:  uapi.GET,
		Docs:    get_team_permissions.Docs,
		Handler: get_team_permissions.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/teams/{id}",
		OpId:    "get_team",
		Method:  uapi.GET,
		Docs:    get_team.Docs,
		Handler: get_team.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/teams/{id}/seo",
		OpId:    "get_team_seo",
		Method:  uapi.GET,
		Docs:    get_team_seo.Docs,
		Handler: get_team_seo.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/teams",
		OpId:    "create_team",
		Method:  uapi.POST,
		Docs:    create_team.Docs,
		Handler: create_team.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "id",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/teams/{tid}",
		OpId:    "edit_team_info",
		Method:  uapi.PATCH,
		Docs:    edit_team_info.Docs,
		Handler: edit_team_info.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/teams/{tid}",
		OpId:    "delete_team",
		Method:  uapi.DELETE,
		Docs:    delete_team.Docs,
		Handler: delete_team.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/teams/{tid}/members",
		OpId:    "add_team_member",
		Method:  uapi.PUT,
		Docs:    add_team_member.Docs,
		Handler: add_team_member.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/teams/{tid}/members/{mid}/permissions",
		OpId:    "edit_team_member_permissions",
		Method:  uapi.PATCH,
		Docs:    edit_team_member_permissions.Docs,
		Handler: edit_team_member_permissions.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/teams/{tid}/members/{mid}",
		OpId:    "delete_team_member",
		Method:  uapi.DELETE,
		Docs:    delete_team_member.Docs,
		Handler: delete_team_member.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/teams",
		OpId:    "add_bot_to_team",
		Method:  uapi.PUT,
		Docs:    add_bot_to_team.Docs,
		Handler: add_bot_to_team.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/teams",
		OpId:    "patch_bot_team",
		Method:  uapi.PATCH,
		Docs:    patch_bot_team.Docs,
		Handler: patch_bot_team.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
