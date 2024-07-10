package put_user_reminders

import (
	"net/http"

	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/votes"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create User Reminder",
		Description: "Creates a new user reminders of an entity",
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

	entityInfo, err := votes.GetEntityInfo(d.Context, state.Pool, targetId, targetType)

	if err != nil {
		state.Logger.Error("Error getting entity info", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Error: " + err.Error()},
		}
	}

	// Get count of old
	var count int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM user_reminders WHERE user_id = $1 AND target_id = $2 AND target_type = $3", d.Auth.ID, targetId, targetType).Scan(&count)

	if err != nil {
		state.Logger.Error("Error selecting count of user_reminders", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Error getting current user reminder count: " + err.Error()},
		}
	}

	if count > 0 {
		return uapi.DefaultResponse(http.StatusNoContent)
	}

	// Add new
	_, err = state.Pool.Exec(d.Context, "INSERT INTO user_reminders (user_id, target_id, target_type) VALUES ($1, $2, $3)", d.Auth.ID, targetId, targetType)

	if err != nil {
		state.Logger.Error("Error inserting new reminder", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Error adding new reminder: " + err.Error()},
		}
	}

	// Fan out notification
	err = notifications.PushNotification(d.Auth.ID, types.Alert{
		Type:    types.AlertTypeSuccess,
		Icon:    entityInfo.Avatar,
		Title:   "Added Reminder: " + entityInfo.Name + "(" + targetType + ":" + targetId + ")",
		Message: "This is an automated message to let you know that vote reminders have been setup for this " + targetType + "!",
	})

	if err != nil {
		state.Logger.Error("Error pushing notification", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
