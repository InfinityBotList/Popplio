package get_reviews

import (
	"net/http"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

var (
	reviewColsArr = db.GetCols(types.Review{})
	reviewCols    = strings.Join(reviewColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Reviews",
		Description: "Gets the reviews of a bot by its ID or vanity.",
		Params: []docs.Parameter{
			{
				Name:        "target_id",
				Description: "The target id (currently only bot ID)",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type (currently only bot)",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ReviewList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := r.URL.Query().Get("target_type")

	if targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+reviewCols+" FROM reviews WHERE target_id = $1 AND target_type = $2 ORDER BY created_at ASC", targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to query reviews [db query]", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	reviews, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Review])

	if err != nil {
		state.Logger.Error("Failed to query reviews [collect]", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range reviews {
		user, err := dovewing.GetUser(d.Context, reviews[i].AuthorID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to get user [dovewing]", zap.Error(err), zap.String("author_id", reviews[i].AuthorID))
			continue
		}

		reviews[i].Author = user
	}

	var allReviews = types.ReviewList{
		Reviews: reviews,
	}

	return uapi.HttpResponse{
		Json: allReviews,
	}
}
