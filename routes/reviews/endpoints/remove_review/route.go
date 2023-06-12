package remove_review

import (
	"net/http"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Review",
		Description: "Deletes a review by review ID. The user must be the author of this review. This will automatically trigger a garbage collection task and returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "rid",
				Description: "The review ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rid := chi.URLParam(r, "rid")

	var author string
	var targetId string
	var targetType string

	err := state.Pool.QueryRow(d.Context, "SELECT author, target_id, target_type FROM reviews WHERE id = $1", rid).Scan(&author, &targetId, &targetType)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if author != d.Auth.ID {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Error:   true,
				Message: "You are not the author of this review",
			},
		}
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM reviews WHERE id = $1", rid)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Trigger a garbage collection step to remove any orphaned reviews
	go assets.GCTrigger(targetId, targetType)

	state.Redis.Del(d.Context, "rv-"+targetId+"-"+targetType)

	return uapi.DefaultResponse(http.StatusNoContent)
}
