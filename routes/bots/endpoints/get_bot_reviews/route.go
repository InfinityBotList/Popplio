package get_bot_reviews

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	reviewColsArr = utils.GetCols(types.Review{})
	reviewCols    = strings.Join(reviewColsArr, ",")
)

func Docs() {
	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/bots/{id}/reviews",
		OpId:        "get_bot_reviews",
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
		Tags: []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	name := chi.URLParam(r, "id")

	name = strings.ToLower(name)

	if name == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+reviewCols+" FROM reviews WHERE (lower(vanity) = $1 OR bot_id = $1)", name)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	var reviews []types.Review = []types.Review{}

	err = pgxscan.ScanAll(&reviews, rows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	var allReviews = types.ReviewList{
		Reviews: reviews,
	}

	d.Resp <- types.HttpResponse{
		Json: allReviews,
	}
}
