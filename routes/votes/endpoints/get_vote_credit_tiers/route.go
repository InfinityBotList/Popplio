// Package get_vote_credit_tiers implements GET
// /{target_type}/{target_id}/votes/credit-tiers — "Get Vote Credit Tiers".
//
// Returns a summary of the tiers and the slab based breakdown of votes for a
// given entity
package get_vote_credit_tiers

import (
	"net/http"
	"popplio/api/resp"

	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/votes"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Vote Credit Tiers",
		Description: "Returns a summary of the tiers and the slab based breakdown of votes for a given entity",
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.VoteCreditTierRedeemSummary{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("target_id and target_type are required")
	}

	summary, err := votes.EntityGetVoteCreditsSummary(d.Context, state.Pool, targetId, targetType)

	if err != nil {
		return resp.ErrBody("An error occurred while summarizing vote credit tiers", "An error occurred while summarizing vote credit tiers.", err)
	}

	return uapi.HttpResponse{
		Json: summary,
	}
}
