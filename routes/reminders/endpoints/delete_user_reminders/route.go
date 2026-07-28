package delete_user_reminders

import (
	"net/http"

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
		state.Logger.Error("Error querying reminders [db count]", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Error while checking user reminder count: " + err.Error()},
		}
	}

	if count == 0 {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "Reminder not found"},
		}
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM user_reminders WHERE user_id = $1 AND target_id = $2 AND target_type = $3", d.Auth.ID, targetId, targetType)

	if err != nil {
		state.Logger.Error("Error deleting reminders", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Error while deleting user reminder: " + err.Error()},
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
