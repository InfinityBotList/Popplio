package get_user_bot_votes

import (
	"net/http"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
		Path:        "/users/{uid}/bots/{bid}/votes",
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
			VoteInfo: types.VoteInfo{
				Weekend:  utils.GetDoubleVote(),
				VoteTime: utils.GetVoteTime(),
			},
			HasVoted: true,
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var vars = map[string]string{
		"uid": chi.URLParam(r, "uid"),
		"bid": chi.URLParam(r, "bid"),
	}

	var botId pgtype.Text

	err := state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bots WHERE "+constants.ResolveBotSQL, vars["bid"]).Scan(&botId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	voteParsed, err := utils.GetVoteData(d.Context, vars["uid"], botId.String)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: voteParsed,
	}
}
