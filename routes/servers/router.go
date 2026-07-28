package servers

import (
	"net/http"
	"popplio/api"
	"popplio/routes/servers/endpoints/get_all_servers"
	"popplio/routes/servers/endpoints/get_random_servers"
	"popplio/routes/servers/endpoints/get_server"
	"popplio/routes/servers/endpoints/get_server_seo"
	"popplio/routes/servers/endpoints/get_servers_index"
	"popplio/routes/servers/endpoints/patch_server_settings"
	"popplio/teams"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
)

const tagName = "Servers"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to servers on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/servers/@all",
		OpId:    "get_all_servers",
		Method:  uapi.GET,
		Docs:    get_all_servers.Docs,
		Handler: get_all_servers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/@random",
		OpId:    "get_random_servers",
		Method:  uapi.GET,
		Docs:    get_random_servers.Docs,
		Handler: get_random_servers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/@index",
		OpId:    "get_servers_index",
		Method:  uapi.GET,
		Docs:    get_servers_index.Docs,
		Handler: get_servers_index.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}",
		OpId:    "get_server",
		Method:  uapi.GET,
		Docs:    get_server.Docs,
		Handler: get_server.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/seo",
		OpId:    "get_server_seo",
		Method:  uapi.GET,
		Docs:    get_server_seo.Docs,
		Handler: get_server_seo.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/settings",
		OpId:    "patch_server_settings",
		Method:  uapi.PATCH,
		Docs:    patch_server_settings.Docs,
		Handler: patch_server_settings.Route,
		Setup:   patch_server_settings.Setup,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
			{
				Type: api.TargetTypeServer,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: api.TargetTypeServer,
						Perm:      teams.PermissionEdit,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return api.TargetTypeServer, chi.URLParam(r, "id")
				},
			},
		},
	}.Route(r)
}
