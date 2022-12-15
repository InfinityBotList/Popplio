package get_bot_reviews

import (
	"net/http"
	"strings"

	"popplio/api"
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

	replyColsArr = utils.GetCols(types.Reply{})
	replyCols    = strings.Join(replyColsArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "id")

	name = strings.ToLower(name)

	if name == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+reviewCols+" FROM reviews WHERE (lower(vanity) = $1 OR bot_id = $1)", name)

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

	for i, review := range reviews {
		rows, err := state.Pool.Query(d.Context, "SELECT "+replyCols+" FROM replies WHERE parent = $1", review.ID)

		if err != nil {
			continue
		}

		var replies []types.Reply = []types.Reply{}

		err = pgxscan.ScanAll(&replies, rows)

		if err != nil {
			continue
		}

		reviews[i].Replies = replies
	}

	var allReviews = types.ReviewList{
		Reviews: reviews,
	}

	return api.HttpResponse{
		Json: allReviews,
	}
}
