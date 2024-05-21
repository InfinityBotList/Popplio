package get_vote_credit_tiers

import (
	"errors"
	"net/http"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

var (
	voteCreditTiersColsArr = db.GetCols(types.VoteCreditTier{})
	voteCreditTiersCols    = strings.Join(voteCreditTiersColsArr, ",")
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
	targetType := chi.URLParam(r, "target_type")

	if targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "target_id and target_type are required"},
		}
	}

	targetType = strings.TrimSuffix(targetType, "s")

	/*perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

	if err != nil {
		state.Logger.Error("Error getting entity perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: targetType, Perm: teams.PermissionRedeemVoteCredits}) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to redeem vote credits for this " + targetType},
		}
	}*/

	rows, err := state.Pool.Query(d.Context, "SELECT "+voteCreditTiersCols+" FROM vote_credit_tiers WHERE target_type = $1 ORDER BY position ASC", targetType)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "An error occurred while fetching vote credit tiers: " + err.Error()},
		}
	}

	vcts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.VoteCreditTier])

	if errors.Is(err, pgx.ErrNoRows) {
		vcts = []*types.VoteCreditTier{}
	}

	voteCount, err := votes.EntityGetVoteCount(d.Context, state.Pool, targetId, targetType)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "An error occurred while fetching vote count: " + err.Error()},
		}
	}

	slabOverview := votes.SlabSplitVotes(voteCount, vcts)

	return uapi.HttpResponse{
		Json: types.VoteCreditTierRedeemSummary{
			Tiers:        vcts,
			VoteCount:    voteCount,
			SlabOverview: slabOverview,
		},
	}
}
