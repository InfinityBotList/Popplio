package assets

import (
	"net/http"
	"popplio/api"
	"popplio/routes/assets/endpoints/delete_asset"
	"popplio/routes/assets/endpoints/upload_asset"
	"popplio/teams"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
)

const tagName = "Assets"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to assets (images etc.) on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/{target_type}/{target_id}/assets",
		OpId:    "upload_asset",
		Method:  uapi.POST,
		Docs:    upload_asset.Docs,
		Handler: upload_asset.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionUploadAssets,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/assets",
		OpId:    "delete_asset",
		Method:  uapi.DELETE,
		Docs:    delete_asset.Docs,
		Handler: delete_asset.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionDeleteAssets,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)
}
