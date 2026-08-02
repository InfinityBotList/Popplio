// Package delete_user_reminders implements DELETE
// /users/{uid}/{target_type}/{target_id}/reminders — "Delete User
// Reminders".
//
// Deletes a users reminders. Returns 204 on success
package delete_user_reminders

import (
	"net/http"
	"popplio/api/resp"

	"popplio/state"
	"popplio/types"
	"popplio/validators"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete User Reminders",
		Description: "Deletes a users reminders. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
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
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ReminderList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	// Check count of deleted rows
	var count int64
	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM user_reminders WHERE user_id = $1 AND target_id = $2 AND target_type = $3", d.Auth.ID, targetId, targetType).Scan(&count)

	if err != nil {
		return resp.ErrBody("Error querying reminders [db count]", "Error while checking user reminder count.", err, zap.String("target_id", targetId), zap.String("target_type", targetType))
	}

	if count == 0 {
		return resp.NotFound("Reminder not found")
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM user_reminders WHERE user_id = $1 AND target_id = $2 AND target_type = $3", d.Auth.ID, targetId, targetType)

	if err != nil {
		return resp.ErrBody("Error deleting reminders", "Error while deleting user reminder.", err, zap.String("target_id", targetId), zap.String("target_type", targetType))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
