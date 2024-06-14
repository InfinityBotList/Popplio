package votes

import (
	"popplio/api"
	"popplio/routes/votes/endpoints/create_entity_vote"
	"popplio/routes/votes/endpoints/get_all_votes"
	"popplio/routes/votes/endpoints/get_general_vote_credit_tiers"
	"popplio/routes/votes/endpoints/get_user_entity_votes"
	"popplio/routes/votes/endpoints/get_vote_credit_tiers"
	"popplio/routes/votes/endpoints/get_vote_redeem_logs"
	"popplio/routes/votes/endpoints/redeem_vote_credits"

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
		Pattern: "/votes/credit-tiers",
		OpId:    "get_general_vote_credit_tiers",
		Method:  uapi.GET,
		Docs:    get_general_vote_credit_tiers.Docs,
		Handler: get_general_vote_credit_tiers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/votes/credit-tiers",
		OpId:    "get_vote_credit_tiers",
		Method:  uapi.GET,
		Docs:    get_vote_credit_tiers.Docs,
		Handler: get_vote_credit_tiers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/votes/credits",
		OpId:    "get_vote_redeem_logs",
		Method:  uapi.GET,
		Docs:    get_vote_redeem_logs.Docs,
		Handler: get_vote_redeem_logs.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/votes/credits",
		OpId:    "redeem_vote_credits",
		Method:  uapi.POST,
		Docs:    redeem_vote_credits.Docs,
		Handler: redeem_vote_credits.Route,
		Auth:    api.GetAllAuthTypes(),
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/{target_type}/{target_id}/votes/@all",
		OpId:    "get_all_votes",
		Method:  uapi.GET,
		Docs:    get_all_votes.Docs,
		Handler: get_all_votes.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/{target_type}/{target_id}/votes",
		OpId:    "get_user_entity_votes",
		Method:  uapi.GET,
		Docs:    get_user_entity_votes.Docs,
		Handler: get_user_entity_votes.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/{target_type}/{target_id}/votes",
		OpId:    "create_entity_vote",
		Method:  uapi.PUT,
		Docs:    create_entity_vote.Docs,
		Handler: create_entity_vote.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
	}.Route(r)
}
