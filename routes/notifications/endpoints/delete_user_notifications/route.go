// Package delete_user_notifications implements DELETE
// /users/{id}/notification — "Delete User Notifications".
//
// Deletes a users notification. Returns 204 on success
package delete_user_notifications

import (
	"net/http"
	"popplio/api/resp"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
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
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")
	notifId := r.URL.Query().Get("notif_id")

	// Check for notif_id
	if notifId == "" {
		return resp.BadRequest("`notif_id` is required in query params and must be set to the notification ID to delete")
	}

	// Check count of deleted rows
	var count int64
	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM user_notifications WHERE user_id = $1 AND notif_id = $2", id, r.URL.Query().Get("notif_id")).Scan(&count)

	if err != nil {
		return resp.ErrDetail("Error while checking user notification count", err, zap.String("userID", id), zap.String("notifID", r.URL.Query().Get("notif_id")))
	}

	if count == 0 {
		return resp.NotFound("Notification not found")
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM user_notifications WHERE user_id = $1 AND notif_id = $2", id, r.URL.Query().Get("notif_id"))

	if err != nil {
		return resp.ErrDetail("Error while deleting user notification", err, zap.String("userID", id), zap.String("notifID", r.URL.Query().Get("notif_id")))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
