// Package add_webhook implements POST /{target_type}/{target_id}/webhooks —
// "Create Webhook".
//
// Creates a new webhook for an entity. Returns 204 on success.
package add_webhook

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
		Summary:     "Create Webhook",
		Description: "Creates a new webhook for an entity. Returns 204 on success. **Requires Create Webhooks permission**",
		Req:         types.CreateEditWebhook{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the tntity",
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
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	switch targetType {
	case "bot":
	case "server":
	case "team":
	default:
		return resp.Status(http.StatusNotImplemented, "Creating webhooks for this target type is not yet supported")
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

	if payload.SimpleAuth && payload.HmacAuth {
		return resp.BadRequest("simple_auth and hmac_auth cannot both be set. Use hmac_auth unless your endpoint cannot implement a signature check")
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

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2", targetId, targetType).Scan(&count)

	if err != nil {
		return resp.Err("Error while checking webhook", err, zap.String("userID", d.Auth.ID))
	}

	if count >= MaximumWebhookCount {
		return resp.BadRequest(fmt.Sprintf("An entity may only have a maximum of %d webhooks", MaximumWebhookCount))
	}

	_, err = tx.Exec(d.Context, "INSERT INTO webhooks (target_id, target_type, url, secret, simple_auth, hmac_auth, name, event_whitelist) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", targetId, targetType, payload.Url, payload.Secret, payload.SimpleAuth, payload.HmacAuth, payload.Name, payload.EventWhitelist)

	if err != nil {
		return resp.Err("Error while inserting webhook", err, zap.String("userID", d.Auth.ID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
