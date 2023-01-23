package get_all_bot_votes

import (
	"net/http"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Bot Votes",
		Description: "Gets all votes which can be used as an alternative to webhooks. **Requires authentication**",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AllVotes{},
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

	rows, err := state.Pool.Query(state.Context, "SELECT user_id FROM votes WHERE bot_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	var voteParsed []types.UserVote
	for rows.Next() {
		var userId string

		err = rows.Scan(&userId)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		voteData, err := utils.GetVoteData(d.Context, userId, id)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		voteParsed = append(voteParsed, *voteData)
	}

	return api.HttpResponse{
		Json: types.AllVotes{
			Votes: voteParsed,
			Count: int64(len(voteParsed)),
		},
	}
}
