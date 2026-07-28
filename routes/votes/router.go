package votes

import (
	"net/http"
	"popplio/api"
	"popplio/routes/votes/endpoints/create_user_entity_vote"
	"popplio/routes/votes/endpoints/get_all_user_votes"
	"popplio/routes/votes/endpoints/get_general_vote_credit_tiers"
	"popplio/routes/votes/endpoints/get_user_entity_votes"
	"popplio/routes/votes/endpoints/get_vote_credit_tiers"
	"popplio/routes/votes/endpoints/get_vote_redeem_logs"
	"popplio/routes/votes/endpoints/get_votes_user_list"
	"popplio/routes/votes/endpoints/redeem_vote_credits"
	"popplio/teams"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
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
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Permission, error) {
					return &perms.Permission{
						Namespace: validators.NormalizeTargetType(chi.URLParam(r, "target_type")),
						Perm:      teams.PermissionRedeemVoteCredits,
					}, nil
				},
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/{target_type}/{target_id}/votes/@all",
		OpId:    "get_all_user_votes",
		Method:  uapi.GET,
		Docs:    get_all_user_votes.Docs,
		Handler: get_all_user_votes.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/votes/user-list",
		OpId:    "get_votes_user_list",
		Method:  uapi.GET,
		Docs:    get_votes_user_list.Docs,
		Handler: get_votes_user_list.Route,
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
		OpId:    "create_user_entity_vote",
		Method:  uapi.PUT,
		Docs:    create_user_entity_vote.Docs,
		Handler: create_user_entity_vote.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)
}
