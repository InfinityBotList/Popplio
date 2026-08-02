// Package patch_webhook implements POST
// /{target_type}/{target_id}/webhooks/{webhook_id} — "Update Webhook".
//
// Updates an existing webhook on an entity. Returns 204 on success.
package patch_webhook

import (
	"fmt"
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/utils"

	"github.com/go-playground/validator/v10"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

const MaximumWebhookCount = 5

var compiledMessages = uapi.CompileValidationErrors(types.CreateEditWebhook{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Webhook",
		Description: "Updates an existing webhook on an entity. Returns 204 on success. **Requires Edit Webhooks permission**",
		Req:         types.CreateEditWebhook{},
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
				Description: "The ID of the webhook to update",
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

	// Read payload from body
	var payload types.CreateEditWebhook

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	if !(strings.HasPrefix(payload.Url, "https://")) {
		return resp.BadRequest("Webhook URL must start with https://. Insecure HTTP webhooks are no longer supported")
	}

	if len(payload.EventWhitelist) == 0 {
		payload.EventWhitelist = []string{}
	}

	if payload.Secret == "" {
		if prefix, err := utils.GetDiscordWebhookInfo(payload.Url); prefix != "" && err == nil {
			payload.Secret = "discordWebhook"
		}
	}

	if payload.Secret == "" {
		return resp.BadRequest(fmt.Sprintf("A secret must be specified for new webhooks: %s", payload.Name))
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

	_, err = tx.Exec(d.Context, "UPDATE webhooks SET name = $1, url = $2, secret = $3, event_whitelist = $4, broken = false, failed_requests = 0 WHERE target_id = $5 AND target_type = $6 AND id = $7", payload.Name, payload.Url, payload.Secret, payload.EventWhitelist, targetId, targetType, webhookId)

	if err != nil {
		return resp.Err("Error while inserting webhook", err, zap.String("userID", d.Auth.ID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
