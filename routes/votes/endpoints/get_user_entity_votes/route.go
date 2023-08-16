package get_user_entity_votes

import (
	"net/http"
	"strings"

	"popplio/state"
	"popplio/types"
	"popplio/votes"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Entity Votes",
		Description: "Gets a vote a user has made for an entity. Note that for compatibility, a trailing 's' is removed",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserVote{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	uid := chi.URLParam(r, "uid")
	targetId := chi.URLParam(r, "target_id")
	targetType := chi.URLParam(r, "target_type")

	if uid == "" || targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	targetType = strings.TrimSuffix(targetType, "s")

	uv, err := votes.EntityVoteCheck(d.Context, uid, targetId, targetType)

	if err != nil {
		state.Logger.Error(err)
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	return uapi.HttpResponse{
		Json: uv,
	}
}
