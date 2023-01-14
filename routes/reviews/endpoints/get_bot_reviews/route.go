package get_bot_reviews

import (
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	reviewColsArr = utils.GetCols(types.Review{})
	reviewCols    = strings.Join(reviewColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
		Path:        "/bots/{id}/reviews",
		Summary:     "Get Bot Reviews",
		Description: "Gets the reviews of a bot by its ID or vanity",
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id, err := utils.ResolveBot(state.Context, chi.URLParam(r, "id"))

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "rv-"+id).Val()
	if cache != "" {
		return api.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
			},
		}
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+reviewCols+" FROM reviews WHERE bot_id = $1 ORDER BY created_at ASC", id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	var reviews []types.Review = []types.Review{}

	err = pgxscan.ScanAll(&reviews, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range reviews {
		reviews[i].Author, err = utils.GetDiscordUser(reviews[i].AuthorID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	reviews, err = assets.GarbageCollect(d.Context, reviews)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var allReviews = types.ReviewList{
		Reviews: reviews,
	}

	return api.HttpResponse{
		Json:      allReviews,
		CacheKey:  "rv-" + id,
		CacheTime: time.Minute * 3,
	}
}
