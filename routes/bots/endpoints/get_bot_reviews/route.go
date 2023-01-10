package get_bot_reviews

import (
	"net/http"
	"strings"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
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
	name := chi.URLParam(r, "id")

	name = strings.ToLower(name)

	if name == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+reviewCols+" FROM reviews WHERE "+constants.ResolveBotSQL, name)

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

	var allReviews = types.ReviewList{
		Reviews: reviews,
	}

	return api.HttpResponse{
		Json: allReviews,
	}
}
