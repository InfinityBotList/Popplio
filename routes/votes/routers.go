package votes

import (
	"popplio/api"
	"popplio/routes/votes/endpoints/get_all_bot_votes"
	"popplio/routes/votes/endpoints/get_bot_webhook_state"
	"popplio/routes/votes/endpoints/get_user_bot_votes"
	"popplio/routes/votes/endpoints/get_user_pack_votes"
	"popplio/routes/votes/endpoints/put_user_bot_votes"
	"popplio/routes/votes/endpoints/put_user_pack_votes"
	"popplio/routes/votes/endpoints/test_webhook"

	"github.com/go-chi/chi/v5"
)

const tagName = "Votes"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to votes and voting on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/bots/{id}/votes",
		OpId:    "get_all_bot_votes",
		Method:  api.GET,
		Docs:    get_all_bot_votes.Docs,
		Handler: get_all_bot_votes.Route,
		Auth: []api.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeBot,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/bots/{id}/webhook-state",
		OpId:    "get_bot_webhook_state",
		Method:  api.GET,
		Docs:    get_bot_webhook_state.Docs,
		Handler: get_bot_webhook_state.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/votes",
		OpId:    "get_user_bot_votes",
		Method:  api.GET,
		Docs:    get_user_bot_votes.Docs,
		Handler: get_user_bot_votes.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
			{
				URLVar: "bid",
				Type:   api.TargetTypeBot,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/votes",
		OpId:    "put_user_bot_votes",
		Method:  api.PUT,
		Docs:    put_user_bot_votes.Docs,
		Handler: put_user_bot_votes.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/packs/{url}/votes",
		OpId:    "get_user_pack_votes",
		Method:  api.GET,
		Docs:    get_user_pack_votes.Docs,
		Handler: get_user_pack_votes.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/packs/{url}/votes",
		OpId:    "put_user_pack_votes",
		Method:  api.PUT,
		Docs:    put_user_pack_votes.Docs,
		Handler: put_user_pack_votes.Route,
		Auth: []api.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{uid}/bots/{bid}/test-webhook",
		OpId:    "test_webhook",
		Method:  api.POST,
		Docs:    test_webhook.Docs,
		Handler: test_webhook.Route,
		Auth: []api.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
