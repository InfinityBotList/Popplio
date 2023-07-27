package vanity

import (
	"popplio/routes/vanity/endpoints/patch_vanity"
	"popplio/routes/vanity/endpoints/redirect_vanity"
	"popplio/routes/vanity/endpoints/resolve_vanity"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Vanity"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to vanity codes on IGL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/vanity/{code}",
		OpId:    "resolve_vanity",
		Method:  uapi.GET,
		Docs:    resolve_vanity.Docs,
		Handler: resolve_vanity.Route,
	}.Route(r)

	uapi.Route{
		Pattern:               "/@{code}",
		OpId:                  "redirect_vanity",
		Method:                uapi.GET,
		Docs:                  redirect_vanity.Docs,
		Handler:               redirect_vanity.Route,
		DisablePathSlashCheck: true,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/vanity/{target_id}",
		OpId:    "patch_vanity",
		Method:  uapi.PATCH,
		Docs:    patch_vanity.Docs,
		Handler: patch_vanity.Route,
	}.Route(r)
}
