package patch_webhook

import (
	"net/http"
	"strings"

	"popplio/state"
	"popplio/teams"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Entity Webhook",
		Description: "Updates a entities webhook. Returns 204 on success",
		Req:         types.PatchWebhook{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the tntity",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	uid := chi.URLParam(r, "uid")
	targetId := chi.URLParam(r, "target_id")
	targetType := r.URL.Query().Get("target_type")

	if uid == "" || targetId == "" || targetType == "" {
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
			Json:   types.ApiError{Message: "Target type not implemented"},
		}
	}

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !perms.Has(targetType, teams.PermissionEditWebhooks) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to update this entities webhook settings"},
		}
	}

	// Read payload from body
	var payload types.PatchWebhook

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	if payload.Clear {
		_, err = tx.Exec(d.Context, "DELETE FROM webhooks WHERE target_id = $1 AND target_type = $2", targetId, targetType)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		return uapi.DefaultResponse(http.StatusNoContent)
	}

	if payload.WebhookURL == "" || payload.WebhookSecret == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both webhook_url and webhook_secret must be specified"},
		}
	}

	if !(strings.HasPrefix(payload.WebhookURL, "http://") || strings.HasPrefix(payload.WebhookURL, "https://")) {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Webhook URL must start with http:// or https://"},
		}
	}

	// Check that a webhooks row exists
	var count int64
	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2", targetId, targetType).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		_, err = tx.Exec(d.Context, "INSERT INTO webhooks (target_id, target_type, url, secret) VALUES ($1, $2, $3, $4)", targetId, targetType, payload.WebhookURL, payload.WebhookSecret)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	} else {
		// Update webhooks
		_, err = tx.Exec(d.Context, "UPDATE webhooks SET url = $1, secret = $2, broken = false WHERE target_id = $3 AND target_type = $4", payload.WebhookURL, payload.WebhookSecret, targetId, targetType)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
