package teams

import (
	"net/http"
	"popplio/api"
	"popplio/routes/teams/endpoints/add_team_member"
	"popplio/routes/teams/endpoints/create_team"
	"popplio/routes/teams/endpoints/delete_team"
	"popplio/routes/teams/endpoints/delete_team_member"
	"popplio/routes/teams/endpoints/edit_team_info"
	"popplio/routes/teams/endpoints/edit_team_member"
	"popplio/routes/teams/endpoints/get_entity_permissions"
	"popplio/routes/teams/endpoints/get_team"
	"popplio/routes/teams/endpoints/get_team_permissions"
	"popplio/routes/teams/endpoints/get_team_seo"
	"popplio/teams"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
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
		Pattern: "/teams",
		OpId:    "create_team",
		Method:  uapi.POST,
		Docs:    create_team.Docs,
		Handler: create_team.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	// Intentionally without authentication
	uapi.Route{
		Pattern: "/users/{id}/{target_type}/{target_id}/perms",
		OpId:    "get_entity_permissions",
		Method:  uapi.GET,
		Docs:    get_entity_permissions.Docs,
		Handler: get_entity_permissions.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/teams/{tid}",
		OpId:    "edit_team_info",
		Method:  uapi.PATCH,
		Docs:    edit_team_info.Docs,
		Handler: edit_team_info.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request) (perms.Permission, error) {
					return perms.Permission{
						Namespace: api.TargetTypeTeam,
						Perm:      teams.PermissionEdit,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request) (string, string) {
					return api.TargetTypeTeam, chi.URLParam(r, "tid")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/teams/{tid}",
		OpId:    "delete_team",
		Method:  uapi.DELETE,
		Docs:    delete_team.Docs,
		Handler: delete_team.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request) (perms.Permission, error) {
					// global.* is needed
					return perms.Permission{
						Namespace: "global",
						Perm:      teams.PermissionOwner,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request) (string, string) {
					return api.TargetTypeTeam, chi.URLParam(r, "tid")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/teams/{tid}/members",
		OpId:    "add_team_member",
		Method:  uapi.PUT,
		Docs:    add_team_member.Docs,
		Handler: add_team_member.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request) (perms.Permission, error) {
					return perms.Permission{
						Namespace: "team_member",
						Perm:      teams.PermissionAdd,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request) (string, string) {
					return api.TargetTypeTeam, chi.URLParam(r, "tid")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/teams/{tid}/members/{mid}",
		OpId:    "edit_team_member",
		Method:  uapi.PATCH,
		Docs:    edit_team_member.Docs,
		Handler: edit_team_member.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request) (perms.Permission, error) {
					return perms.Permission{
						Namespace: "team_member",
						Perm:      teams.PermissionEdit,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request) (string, string) {
					return api.TargetTypeTeam, chi.URLParam(r, "tid")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/teams/{tid}/members/{mid}",
		OpId:    "delete_team_member",
		Method:  uapi.DELETE,
		Docs:    delete_team_member.Docs,
		Handler: delete_team_member.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request) (perms.Permission, error) {
					return perms.Permission{
						Namespace: "team_member",
						Perm:      teams.PermissionDelete,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request) (string, string) {
					return api.TargetTypeTeam, chi.URLParam(r, "tid")
				},
			},
		},
	}.Route(r)
}
