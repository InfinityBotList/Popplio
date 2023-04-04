package webhooks

import (
	"popplio/api"
	"popplio/routes/webhooks/endpoints/test_vote_webhook"

	"github.com/go-chi/chi/v5"
)

const tagName = "Webhooks"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to webhooks on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/webhooks/test-vote",
		OpId:    "test_vote_webhook",
		Method:  api.POST,
		Docs:    test_vote_webhook.Docs,
		Handler: test_vote_webhook.Route,
		Auth: []api.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
