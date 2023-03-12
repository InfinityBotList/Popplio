package remove_review

import (
	"net/http"
	"popplio/api"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/doclib"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	reviewColsArr = utils.GetCols(types.Review{})
	reviewCols    = strings.Join(reviewColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Review",
		Description: "Deletes a review by review ID. The user must be the author of this review. This will automatically trigger a garbage collection task and returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "rid",
				Description: "The review ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	rid := chi.URLParam(r, "rid")

	var author string
	var botId string

	err := state.Pool.QueryRow(d.Context, "SELECT author, bot_id FROM reviews WHERE id = $1", rid).Scan(&author, &botId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	if author != d.Auth.ID {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Error:   true,
				Message: "You are not the author of this review",
			},
		}
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM reviews WHERE id = $1", rid)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	// Trigger a garbage collection step to remove any orphaned reviews
	go func() {
		rows, err := state.Pool.Query(state.Context, "SELECT "+reviewCols+" FROM reviews WHERE bot_id = $1 ORDER BY created_at ASC", botId)

		if err != nil {
			state.Logger.Error(err)
		}

		var reviews []types.Review = []types.Review{}

		err = pgxscan.ScanAll(&reviews, rows)

		if err != nil {
			state.Logger.Error(err)
		}

		err = assets.GarbageCollect(state.Context, reviews)

		if err != nil {
			state.Logger.Error(err)
		}
	}()

	state.Redis.Del(d.Context, "rv-"+botId)

	return api.DefaultResponse(http.StatusNoContent)
}
