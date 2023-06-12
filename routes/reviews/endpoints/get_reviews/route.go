package get_reviews

import (
	"net/http"
	"strings"
	"time"

	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	reviewColsArr = utils.GetCols(types.Review{})
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

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "rv-"+targetId+"-"+targetType).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+reviewCols+" FROM reviews WHERE target_id = $1 AND target_type = $2 ORDER BY created_at ASC", targetId, targetType)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	var reviews []types.Review = []types.Review{}

	err = pgxscan.ScanAll(&reviews, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range reviews {
		user, err := dovewing.GetDiscordUser(d.Context, reviews[i].AuthorID)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		reviews[i].Author = user
	}

	var allReviews = types.ReviewList{
		Reviews: reviews,
	}

	return uapi.HttpResponse{
		Json:      allReviews,
		CacheKey:  "rv-" + targetId + "-" + targetType,
		CacheTime: time.Minute * 3,
	}
}