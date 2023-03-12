package get_user_pack_votes

import (
	"net/http"
	"time"

	"popplio/api"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Pack Votes",
		Description: "Gets the users votes. **Does not require authentication**",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "url",
				Description: "The pack URL",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.PackVote{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	userId := chi.URLParam(r, "uid")
	packUrl := chi.URLParam(r, "url")

	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM pack_votes WHERE user_id = $1 AND url = $2", userId, packUrl).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var upvote bool
	var createdAt time.Time

	err = state.Pool.QueryRow(d.Context, "SELECT upvote, created_at FROM pack_votes WHERE user_id = $1 AND url = $2", userId, packUrl).Scan(&upvote, &createdAt)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: types.PackVote{
			UserID:    userId,
			Upvote:    upvote,
			CreatedAt: createdAt,
		},
	}
}
