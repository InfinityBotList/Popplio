package get_bot_reviews

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
		Summary:     "Get Bot Reviews",
		Description: "Gets the reviews of a bot by its ID or vanity.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ReviewList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id, err := utils.ResolveBot(d.Context, chi.URLParam(r, "id"))

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "rv-"+id).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+reviewCols+" FROM reviews WHERE bot_id = $1 ORDER BY created_at ASC", id)

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
		CacheKey:  "rv-" + id,
		CacheTime: time.Minute * 3,
	}
}
