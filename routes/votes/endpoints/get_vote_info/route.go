package get_vote_info

import (
	"net/http"

	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Vote Info",
		Description: "Returns basic voting info for an entity",
		Params: []docs.Parameter{
			{
				Name:        "target_id",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the tntity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "user_id",
				Description: "The users ID, if you wish the api to take into account user-special perks",
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.VoteInfo{DoubleVotes: true, VoteTime: 6},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := chi.URLParam(r, "target_type")

	if targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	uid := r.URL.Query().Get("user_id")

	vi, err := votes.EntityVoteInfo(d.Context, uid, targetId, targetType)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: vi,
	}
}
