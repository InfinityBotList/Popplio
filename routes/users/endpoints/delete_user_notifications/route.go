package delete_user_notifications

import (
	"net/http"

	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/docs"
	"github.com/infinitybotlist/popplio/state"
	"github.com/infinitybotlist/popplio/types"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "DELETE",
		Path:        "/users/{id}/notification",
		OpId:        "delete_user_notifications",
		Summary:     "Delete User Notifications",
		Description: "Deletes a users notification. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "notif_id",
				Description: "Notification ID",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp:     types.ApiError{},
		Tags:     []string{api.CurrentTag},
		AuthType: []types.TargetType{types.TargetTypeUser},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	// Check for notif_id
	if r.URL.Query().Get("notif_id") == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	_, err := state.Pool.Exec(d.Context, "DELETE FROM poppypaw WHERE user_id = $1 AND notif_id = $2", id, r.URL.Query().Get("notif_id"))

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Status: http.StatusNoContent,
	}
}
