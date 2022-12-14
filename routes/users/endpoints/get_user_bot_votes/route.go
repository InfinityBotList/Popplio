package get_user_bot_votes

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{uid}/bots/{bid}/votes",
		OpId:        "get_user_bot_votes",
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
			VoteTime:   12,
			HasVoted:   true,
		},
		AuthType: []types.TargetType{types.TargetTypeUser, types.TargetTypeBot},
		Tags:     []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var vars = map[string]string{
		"uid": chi.URLParam(r, "uid"),
		"bid": chi.URLParam(r, "bid"),
	}

	var botId pgtype.Text

	err := state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", vars["bid"]).Scan(&botId)

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
