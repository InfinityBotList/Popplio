package special

import (
	"popplio/api"
	"popplio/routes/special/endpoints/get_cosmog_task_tid"
	"popplio/routes/special/endpoints/get_special_login"
	"popplio/routes/special/endpoints/get_special_login_resp"

	"github.com/go-chi/chi/v5"
)

const (
	tagName = "Special Routes"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "Special endpoints, these don't return JSONs and are purely for browser use."
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/login/{act}",
		Method:  api.GET,
		Docs:    get_special_login.Docs,
		Handler: get_special_login.Route,
	}.Route(r)

	api.Route{
		Pattern: "/cosmog",
		Method:  api.GET,
		Docs:    get_special_login_resp.Docs,
		Handler: get_special_login_resp.Route,
	}.Route(r)

	api.Route{
		Pattern: "/cosmog/tasks/{tid}",
		Method:  api.GET,
		Docs:    get_cosmog_task_tid.Docs,
		Handler: get_cosmog_task_tid.Route,
	}.Route(r)
}
