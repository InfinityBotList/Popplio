package delete_webhook

import (
	"net/http"

	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

const MaximumWebhookCount = 5

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Webhook",
		Description: "Updates an existing webhook on an entity. Returns 204 on success. **Requires Edit Webhooks permission**",
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
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
			{
				Name:        "webhook_id",
				Description: "The ID of the webhook to delete",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))
	webhookId := chi.URLParam(r, "webhook_id")

	if targetId == "" || targetType == "" || webhookId == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	switch targetType {
	case "bot":
	case "server":
	case "team":
	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "Creating webhooks for this target type is not yet supported"},
		}
	}

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: targetType, Perm: teams.PermissionCreateWebhooks}) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to update this entities webhook settings"},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var count int64

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3", targetId, targetType, webhookId).Scan(&count)

	if err != nil {
		state.Logger.Error("Error while checking webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "Webhook not found"},
		}
	}

	_, err = tx.Exec(d.Context, "DELETE FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3", targetId, targetType, webhookId)

	if err != nil {
		state.Logger.Error("Error while inserting webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error while committing transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
