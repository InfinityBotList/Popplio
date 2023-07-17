package webhooks

import (
	"popplio/api"
	"popplio/routes/webhooks/endpoints/get_webhook_logs"
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
