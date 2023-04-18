package webhooks

import (
	"popplio/api"
	"popplio/routes/webhooks/endpoints/test_vote_webhook"

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
		Pattern: "/users/{uid}/bots/{bid}/webhooks/test-vote",
		OpId:    "test_vote_webhook",
		Method:  uapi.POST,
		Docs:    test_vote_webhook.Docs,
		Handler: test_vote_webhook.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
