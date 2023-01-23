package votes

import (
	"popplio/api"
	"popplio/routes/votes/endpoints/get_all_bot_votes"
	"popplio/routes/votes/endpoints/get_user_bot_votes"
	"popplio/routes/votes/endpoints/get_user_pack_votes"
	"popplio/routes/votes/endpoints/put_user_bot_votes"
	"popplio/routes/votes/endpoints/put_user_pack_votes"
	"popplio/routes/votes/endpoints/test_webhook"
	"popplio/types"

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
				Type:   types.TargetTypeBot,
			},
		},
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
				Type:   types.TargetTypeUser,
			},
			{
				URLVar: "bid",
				Type:   types.TargetTypeBot,
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
				Type:   types.TargetTypeUser,
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
				Type:   types.TargetTypeUser,
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
				Type:   types.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
