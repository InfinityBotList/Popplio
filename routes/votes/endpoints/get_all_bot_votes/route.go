package get_all_bot_votes

import (
	"net/http"
	"strconv"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
)

const perPage = 10

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Bot Votes",
		Description: "Gets all votes (paginated by 100) which can be used as an alternative to webhooks. **Requires authentication**",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AllVotes{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rows, err := state.Pool.Query(d.Context, "SELECT user_id FROM votes WHERE bot_id = $1 LIMIT $2 OFFSET $3", d.Auth.ID, limit, offset)

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

		voteData, err := utils.GetVoteData(d.Context, userId, d.Auth.ID, false)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		voteParsed = append(voteParsed, *voteData)
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM votes WHERE bot_id = $1", d.Auth.ID).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: types.AllVotes{
			Votes:      voteParsed,
			Count:      count,
			PerPage:    perPage,
			TotalPages: uint64((count / perPage) + 1),
		},
	}
}