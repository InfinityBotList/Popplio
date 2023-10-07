package servers

import (
	"popplio/api"
	"popplio/routes/servers/endpoints/get_server"
	"popplio/routes/servers/endpoints/get_server_seo"
	"popplio/routes/servers/endpoints/get_servers_index"
	"popplio/routes/servers/endpoints/patch_server_settings"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Servers"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to servers on IBL"
}

func (b Router) Routes(r *chi.Mux) {
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
		Pattern: "/users/{uid}/servers/{sid}/settings",
		OpId:    "patch_bot_settings",
		Method:  uapi.PATCH,
		Docs:    patch_server_settings.Docs,
		Handler: patch_server_settings.Route,
		Setup:   patch_server_settings.Setup,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
