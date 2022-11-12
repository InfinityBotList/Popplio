package compat

import (
	"net/http"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgtype"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

const tagName = "Legacy"

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are set to be removed in the next major API version. Please use the new endpoints instead."
}

func (b Router) Routes(r *chi.Mux) {
	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/votes/{bot_id}/{user_id}",
		OpId:        "legacy_votes",
		Summary:     "Legacy Get Votes",
		Description: "This endpoint has been replaced with Get User Votes. This endpoint will be removed in the next major API version.",
		Tags:        []string{tagName},
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				In:          "path",
				Description: "The user's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bot_id",
				In:          "path",
				Description: "The bot's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp:     types.UserVoteCompat{},
		AuthType: []string{"User"},
	})
	r.Get("/votes/{bot_id}/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		var botId = chi.URLParam(r, "bot_id")
		var userId = chi.URLParam(r, "user_id")

		if r.Header.Get("Authorization") == "" {
			utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			id := utils.AuthCheck(r.Header.Get("Authorization"), true)

			if id == nil || *id != botId {
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}

			// To try and push users into new API, vote ban and approved check on GET is enforced on the old API
			var voteBannedState bool

			err := state.Pool.QueryRow(state.Context, "SELECT vote_banned FROM bots WHERE bot_id = $1", id).Scan(&voteBannedState)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		var botType pgtype.Text

		state.Pool.QueryRow(state.Context, "SELECT type FROM bots WHERE bot_id = $1", botId).Scan(&botType)

		if botType.String != "approved" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(constants.NotApproved))
			return
		}

		voteParsed, err := utils.GetVoteData(state.Context, userId, botId)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var compatData = types.UserVoteCompat{
			HasVoted: voteParsed.HasVoted,
		}

		bytes, err := json.Marshal(compatData)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	})
}
