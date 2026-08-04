// Package redeem_vote_credits implements POST
// /{target_type}/{target_id}/votes/credits — "Redeem Vote Credits".
//
// Redeems votes into credits towards the shop based on the vote credit tiers
package redeem_vote_credits

import (
	"net/http"
	"popplio/api/resp"
	"strconv"

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
		Summary:     "Redeem Vote Credits",
		Description: "Redeems votes into credits towards the shop based on the vote credit tiers",
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
			{
				Name:        "votes",
				Description: "The number of votes to redeem",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("target_id and target_type are required")
	}

	votesParam := r.URL.Query().Get("votes")

	votesInt, err := strconv.Atoi(votesParam)

	if err != nil {
		return resp.BadRequest("votes must be an integer")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.ErrBody("Error starting transaction", "An error occurred while starting transaction.", err)
	}

	defer tx.Rollback(d.Context)

	err = votes.EntityRedeemVoteCredits(d.Context, tx, targetId, targetType, votesInt)

	if err != nil {
		return resp.ErrBody("An error occurred while redeeming vote credit tiers", "An error occurred while redeeming vote credit tiers.", err)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.ErrBody("Error committing transaction", "An error occurred while committing transaction.", err)
	}

	return uapi.HttpResponse{
		Status: http.StatusNoContent,
	}
}
