package add_webhook

import (
	"fmt"
	"net/http"
	"strings"

	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/utils"

	"github.com/go-playground/validator/v10"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	perms "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"

	"popplio/api/authz"

	"github.com/go-chi/chi/v5"
)

const MaximumWebhookCount = 5

var compiledMessages = uapi.CompileValidationErrors(types.CreateWebhook{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Webhook",
		Description: "Creates a new webhook for an entity. Returns 204 on success. **Requires Create Webhooks permission**",
		Req:         types.CreateWebhook{},
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

	// Perform entity specific checks
	err := authz.EntityPermissionCheck(
		d.Context,
		d.Auth,
		targetType,
		targetId,
		perms.Permission{Namespace: targetType, Perm: teams.PermissionCreateWebhooks},
	)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "Entity permission checks failed: " + err.Error()},
		}
	}

	// Read payload from body
	var payload types.CreateWebhook

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	if !(strings.HasPrefix(payload.Url, "https://")) {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Webhook URL must start with https://. Insecure HTTP webhooks are no longer supported"},
		}
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
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: fmt.Sprintf("A secret must be specified for new webhooks: %s", payload.Name),
			},
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var count int64

	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2", targetId, targetType).Scan(&count)

	if err != nil {
		state.Logger.Error("Error while checking webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count >= MaximumWebhookCount {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: fmt.Sprintf("An entity may only have a maximum of %d webhooks", MaximumWebhookCount)},
		}
	}

	_, err = tx.Exec(d.Context, "INSERT INTO webhooks (target_id, target_type, url, secret, simple_auth, name, event_whitelist) VALUES ($1, $2, $3, $4, $5, $6, $7)", targetId, targetType, payload.Url, payload.Secret, payload.SimpleAuth, payload.Name, payload.EventWhitelist)

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
