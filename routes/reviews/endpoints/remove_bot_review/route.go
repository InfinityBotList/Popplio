package remove_bot_review

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Bot Review",
		Description: "Deletes a bot review by review ID. This will automatically trigger a garbage collection task and returns 204 on success",
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	rid := chi.URLParam(r, "rid")

	var author string

	err := state.Pool.QueryRow(d.Context, "SELECT author FROM reviews WHERE id = $1", rid).Scan(&author)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	if author != d.Auth.ID {
		return api.HttpResponse{
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
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.DefaultResponse(http.StatusNoContent)
}
