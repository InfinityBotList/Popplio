// Package vanity mounts the "Vanity" group of API routes.
//
// These API endpoints are related to vanity codes on IGL
package vanity

import (
	"net/http"
	"popplio/api"
	"popplio/perms"
	"popplio/routes/vanity/endpoints/patch_vanity"
	"popplio/routes/vanity/endpoints/redirect_vanity"
	"popplio/routes/vanity/endpoints/resolve_vanity"
	"popplio/validators"

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
		Pattern: "/{target_type}/{target_id}/vanity",
		OpId:    "patch_vanity",
		Method:  uapi.PATCH,
		Docs:    patch_vanity.Docs,
		Handler: patch_vanity.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: api.Needs(perms.EntitySetVanity),
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)
}
