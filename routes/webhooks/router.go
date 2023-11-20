package webhooks

import (
	"popplio/api"
	"popplio/routes/webhooks/endpoints/get_test_webhook_meta"
	"popplio/routes/webhooks/endpoints/get_webhook_list"
	"popplio/routes/webhooks/endpoints/get_webhook_logs"
	"popplio/routes/webhooks/endpoints/patch_webhook"
	"popplio/routes/webhooks/endpoints/test_webhook"

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
		Pattern: "/users/{uid}/webhooks/{target_id}",
		OpId:    "get_webhook",
		Method:  uapi.GET,
		Docs:    get_webhook_list.Docs,
		Handler: get_webhook_list.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/webhooks/{target_id}",
		OpId:    "patch_webhook",
		Method:  uapi.PATCH,
		Docs:    patch_webhook.Docs,
		Handler: patch_webhook.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/webhooks/{target_id}/logs",
		OpId:    "get_webhook_logs",
		Method:  uapi.GET,
		Docs:    get_webhook_logs.Docs,
		Handler: get_webhook_logs.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/webhooks/test/meta",
		OpId:    "get_test_webhook_meta",
		Method:  uapi.GET,
		Docs:    get_test_webhook_meta.Docs,
		Handler: get_test_webhook_meta.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/webhooks/{target_id}/test",
		OpId:    "test_webhook",
		Method:  uapi.POST,
		Docs:    test_webhook.Docs,
		Handler: test_webhook.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
