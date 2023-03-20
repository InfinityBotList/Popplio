package apps

import (
	"popplio/api"
	"popplio/routes/apps/endpoints/create_app"
	"popplio/routes/apps/endpoints/get_app"
	"popplio/routes/apps/endpoints/get_apps_list"
	"popplio/routes/apps/endpoints/get_apps_meta"
	"popplio/routes/apps/endpoints/manage_app"

	"github.com/go-chi/chi/v5"
)

const tagName = "Apps"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to apps and interviews for positions on our list."
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/apps/meta",
		OpId:    "get_apps_meta",
		Method:  api.GET,
		Docs:    get_apps_meta.Docs,
		Handler: get_apps_meta.Route,
	}.Route(r)
	api.Route{
		Pattern: "/apps/{id}",
		OpId:    "get_app",
		Method:  api.GET,
		Docs:    get_app.Docs,
		Handler: get_app.Route,
	}.Route(r)
	api.Route{
		Pattern: "/users/{user_id}/apps",
		OpId:    "get_apps_list",
		Method:  api.GET,
		Docs:    get_apps_list.Docs,
		Handler: get_apps_list.Route,
		Auth: []api.AuthType{
			{
				URLVar:       "user_id",
				Type:         api.TargetTypeUser,
				AllowedScope: "ban_exempt", // Ensure banned users can view their own apps
			},
		},
	}.Route(r)
	api.Route{
		Pattern: "/users/{user_id}/apps",
		OpId:    "create_app",
		Method:  api.POST,
		Docs:    create_app.Docs,
		Handler: create_app.Route,
		Auth: []api.AuthType{
			{
				URLVar:       "user_id",
				Type:         api.TargetTypeUser,
				AllowedScope: "ban_exempt", // Ensure banned users can create apps
			},
		},
	}.Route(r)
	api.Route{
		Pattern: "/users/{user_id}/apps/{app_id}",
		OpId:    "manage_app",
		Method:  api.PATCH,
		Docs:    manage_app.Docs,
		Handler: manage_app.Route,
		Auth: []api.AuthType{
			{
				URLVar: "user_id",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
