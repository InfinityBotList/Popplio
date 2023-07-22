package votes

import (
	"popplio/api"
	"popplio/routes/votes/endpoints/get_all_bot_votes"
	"popplio/routes/votes/endpoints/get_hcaptcha_info"
	"popplio/routes/votes/endpoints/get_user_bot_votes"
	"popplio/routes/votes/endpoints/get_user_pack_votes"
	"popplio/routes/votes/endpoints/put_user_bot_votes"
	"popplio/routes/votes/endpoints/put_user_pack_votes"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Votes"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to votes and voting on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/security/hcaptcha",
		OpId:    "get_hcaptcha_info",
		Method:  uapi.GET,
		Docs:    get_hcaptcha_info.Docs,
		Handler: get_hcaptcha_info.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/bots/{id}/votes",
		OpId:    "get_all_bot_votes",
		Method:  uapi.GET,
		Docs:    get_all_bot_votes.Docs,
		Handler: get_all_bot_votes.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeBot,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/votes",
		OpId:    "get_user_bot_votes",
		Method:  uapi.GET,
		Docs:    get_user_bot_votes.Docs,
		Handler: get_user_bot_votes.Route,
		Auth: []uapi.AuthType{
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

	uapi.Route{
		Pattern: "/users/{uid}/bots/{bid}/votes",
		OpId:    "put_user_bot_votes",
		Method:  uapi.PUT,
		Docs:    put_user_bot_votes.Docs,
		Handler: put_user_bot_votes.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/packs/{url}/votes",
		OpId:    "get_user_pack_votes",
		Method:  uapi.GET,
		Docs:    get_user_pack_votes.Docs,
		Handler: get_user_pack_votes.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/packs/{url}/votes",
		OpId:    "put_user_pack_votes",
		Method:  uapi.PUT,
		Docs:    put_user_pack_votes.Docs,
		Handler: put_user_pack_votes.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
	}.Route(r)
}
