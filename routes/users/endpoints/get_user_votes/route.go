package get_user_votes

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() {
	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{uid}/bots/{bid}/votes",
		OpId:        "get_user_votes",
		Summary:     "Get User Votes",
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
		AuthType: []string{"User", "Bot"},
		Tags:     []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var vars = map[string]string{
		"uid": chi.URLParam(r, "uid"),
		"bid": chi.URLParam(r, "bid"),
	}

	userAuth := strings.HasPrefix(r.Header.Get("Authorization"), "User ")

	var botId pgtype.Text
	var botType pgtype.Text

	if r.Header.Get("Authorization") == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
		return
	}

	var err error

	if userAuth {
		uid := utils.AuthCheck(r.Header.Get("Authorization"), false)

		if uid == nil || *uid != vars["uid"] {
			d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
			return
		}

		err = state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", vars["bid"]).Scan(&botId)

		if err != nil || !botId.Valid {
			state.Logger.Error(err)
			d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
			return
		}

		vars["bid"] = botId.String
	} else {
		err = state.Pool.QueryRow(d.Context, "SELECT bot_id, type FROM bots WHERE (lower(vanity) = $1 OR bot_id = $1)", vars["bid"]).Scan(&botId, &botType)

		if err != nil || !botId.Valid || !botType.Valid {
			state.Logger.Error(err)
			d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
			return
		}

		id := utils.AuthCheck(r.Header.Get("Authorization"), true)

		if id == nil || *id != vars["bid"] {
			d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
			return
		}

		vars["bid"] = botId.String
	}

	voteParsed, err := utils.GetVoteData(d.Context, vars["uid"], vars["bid"])

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	d.Resp <- types.HttpResponse{
		Json: voteParsed,
	}
}
