package legacy_votes

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
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/votes/{bot_id}/{user_id}",
		OpId:        "legacy_votes",
		Summary:     "Legacy Get Votes",
		Description: "This endpoint has been replaced with Get User Votes. This endpoint will be removed in the next major API version.",
		Tags:        []string{api.CurrentTag},
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
		AuthType: []types.TargetType{types.TargetTypeBot},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var botId = chi.URLParam(r, "bot_id")
	var userId = chi.URLParam(r, "user_id")

	// To try and push users into new API, vote ban and approved check on GET is enforced on the old API
	var voteBannedState bool

	err := state.Pool.QueryRow(d.Context, "SELECT vote_banned FROM bots WHERE bot_id = $1", d.Auth.ID).Scan(&voteBannedState)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusUnauthorized)
		return
	}

	var botType pgtype.Text

	state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", botId).Scan(&botType)

	if botType.String != "approved" || botType.String != "certified" {
		d.Resp <- api.HttpResponse{
			Status: http.StatusBadRequest,
			Data:   constants.NotApproved,
		}
		return
	}

	voteParsed, err := utils.GetVoteData(d.Context, userId, botId)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	var compatData = types.UserVoteCompat{
		HasVoted: voteParsed.HasVoted,
	}

	d.Resp <- api.HttpResponse{
		Status: http.StatusOK,
		Json:   compatData,
	}
}
