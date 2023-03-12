package get_user_bot_votes

import (
	"net/http"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Bot Votes",
		Description: "Gets the users votes. **Requires authentication**",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserVote{
			Timestamps: []int64{},
			VoteInfo:   types.VoteInfo{},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	id, err := utils.ResolveBot(d.Context, chi.URLParam(r, "bid"))

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	userId := chi.URLParam(r, "uid")

	voteParsed, err := utils.GetVoteData(d.Context, userId, id, true)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: voteParsed,
	}
}
