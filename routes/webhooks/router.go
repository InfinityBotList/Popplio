package webhooks

import (
	"net/http"
	"popplio/api"
	"popplio/routes/webhooks/endpoints/add_webhook"
	"popplio/routes/webhooks/endpoints/delete_webhook"
	"popplio/routes/webhooks/endpoints/get_test_webhook_meta"
	"popplio/routes/webhooks/endpoints/get_webhook_logs"
	"popplio/routes/webhooks/endpoints/get_webhooks"
	"popplio/routes/webhooks/endpoints/patch_webhook"
	"popplio/routes/webhooks/endpoints/test_webhook"
	"popplio/teams"
	"popplio/validators"

	perms "github.com/infinitybotlist/kittycat/go"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Webhooks"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to webhooks on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/{target_type}/{target_id}/webhooks",
		OpId:    "get_webhook",
		Method:  uapi.GET,
		Docs:    get_webhooks.Docs,
		Handler: get_webhooks.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionGetWebhooks,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/webhooks",
		OpId:    "add_webhook",
		Method:  uapi.POST,
		Docs:    add_webhook.Docs,
		Handler: add_webhook.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionCreateWebhooks,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/webhooks/{webhook_id}",
		OpId:    "patch_webhook",
		Method:  uapi.POST,
		Docs:    patch_webhook.Docs,
		Handler: patch_webhook.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionEditWebhooks,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/webhooks/{webhook_id}",
		OpId:    "delete_webhook",
		Method:  uapi.DELETE,
		Docs:    delete_webhook.Docs,
		Handler: delete_webhook.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionDeleteWebhooks,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/webhooks/logs",
		OpId:    "get_webhook_logs",
		Method:  uapi.GET,
		Docs:    get_webhook_logs.Docs,
		Handler: get_webhook_logs.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionGetWebhookLogs,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/webhooks/test",
		OpId:    "get_test_webhook_meta",
		Method:  uapi.GET,
		Docs:    get_test_webhook_meta.Docs,
		Handler: get_test_webhook_meta.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/webhooks/test",
		OpId:    "test_webhook",
		Method:  uapi.POST,
		Docs:    test_webhook.Docs,
		Handler: test_webhook.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionTestWebhooks,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)
}
