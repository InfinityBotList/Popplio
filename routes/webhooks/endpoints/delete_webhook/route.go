// Package delete_webhook implements DELETE
// /{target_type}/{target_id}/webhooks/{webhook_id} — "Delete Webhook".
//
// Updates an existing webhook on an entity. Returns 204 on success.
package delete_webhook

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
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	switch targetType {
	case "bot":
	case "server":
	case "team":
	default:
		return resp.Status(http.StatusNotImplemented, "Creating webhooks for this target type is not yet supported")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID))
	}

	var count int64

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3", targetId, targetType, webhookId).Scan(&count)

	if err != nil {
		return resp.Err("Error while checking webhook", err, zap.String("userID", d.Auth.ID))
	}

	if count == 0 {
		return resp.NotFound("Webhook not found")
	}

	_, err = tx.Exec(d.Context, "DELETE FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3", targetId, targetType, webhookId)

	if err != nil {
		return resp.Err("Error while inserting webhook", err, zap.String("userID", d.Auth.ID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
