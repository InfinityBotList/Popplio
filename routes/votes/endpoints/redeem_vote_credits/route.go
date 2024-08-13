package redeem_vote_credits

import (
	"net/http"
	"strconv"

	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/votes"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
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
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "target_id and target_type are required"},
		}
	}

	votesParam := chi.URLParam(r, "votes")

	votesInt, err := strconv.Atoi(votesParam)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "votes must be an integer"},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error starting transaction", zap.Error(err))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "An error occurred while starting transaction: " + err.Error()},
		}
	}

	defer tx.Rollback(d.Context)

	err = votes.EntityRedeemVoteCredits(d.Context, tx, targetId, targetType, votesInt)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "An error occurred while redeeming vote credit tiers: " + err.Error()},
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error committing transaction", zap.Error(err))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "An error occurred while committing transaction: " + err.Error()},
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusNoContent,
	}
}
