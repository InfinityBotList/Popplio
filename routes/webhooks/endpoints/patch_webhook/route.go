package patch_webhook

import (
	"fmt"
	"net/http"
	"strings"

	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/webhooks/core/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Entity Webhook",
		Description: "Updates the webhooks of an entity. Note that only provided webhooks are edited. Returns 204 on success",
		Req:         []types.PatchWebhook{},
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
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("userID", d.Auth.ID))
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
	var payload []types.PatchWebhook

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for _, v := range payload {
		// Special case of clear
		if v.Delete {
			if v.WebhookID == "" {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Webhook ID must be specified to delete a webhook"},
				}
			}

			var count int64

			err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3", targetId, targetType, v.WebhookID).Scan(&count)

			if err != nil {
				state.Logger.Error("Error while checking webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			if count == 0 {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: fmt.Sprintf("Webhook ID %s does not exist", v.WebhookID)},
				}
			}

			_, err = tx.Exec(d.Context, "DELETE FROM webhook_logs WHERE target_id = $1 AND target_type = $2 AND webhook_id = $3", targetId, targetType, v.WebhookID)

			if err != nil {
				state.Logger.Error("Error while deleting webhook logs", zap.Error(err), zap.String("userID", d.Auth.ID))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			_, err = tx.Exec(d.Context, "DELETE FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3", targetId, targetType, v.WebhookID)

			if err != nil {
				state.Logger.Error("Error while deleting webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			continue
		}

		if v.Name == "" {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Name must be specified"},
			}
		}

		if v.WebhookURL == "" {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: fmt.Sprintf("A secret must be specified: %s", v.Name),
				},
			}
		}

		if !(strings.HasPrefix(v.WebhookURL, "https://")) {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Webhook URL must start with https://. Insecure HTTP webhooks are no longer supported"},
			}
		}

		if len(v.EventWhitelist) == 0 {
			v.EventWhitelist = []string{}
		}

		if v.WebhookID != "" {
			var count int64

			err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3", targetId, targetType, v.WebhookID).Scan(&count)

			if err != nil {
				state.Logger.Error("Error while checking webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			if count == 0 {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: fmt.Sprintf("Webhook ID %s does not exist", v.WebhookID)},
				}
			}

			_, err = tx.Exec(d.Context, "UPDATE webhooks SET url = $4, broken = false, simple_auth = $5, name = $6, event_whitelist = $7 WHERE target_id = $1 AND target_type = $2 AND id = $3", targetId, targetType, v.WebhookID, v.WebhookURL, v.SimpleAuth, v.Name, v.EventWhitelist)

			if err != nil {
				state.Logger.Error("Error while updating webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			if v.WebhookSecret != "" {
				_, err = tx.Exec(d.Context, "UPDATE webhooks SET secret = $1 WHERE target_id = $2 AND target_type = $3 AND id = $4", v.WebhookSecret, targetId, targetType, v.WebhookID)

				if err != nil {
					state.Logger.Error("Error while updating webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
					return uapi.DefaultResponse(http.StatusInternalServerError)
				}
			}
		} else {
			if v.WebhookSecret == "" {
				if prefix, err := utils.GetDiscordWebhookInfo(v.WebhookURL); prefix != "" && err == nil {
					v.WebhookSecret = "discordWebhook"
				}
			}

			if v.WebhookSecret == "" {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json: types.ApiError{
						Message: fmt.Sprintf("A secret must be specified for new webhooks: %s", v.Name),
					},
				}
			}

			var count int64

			err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2", targetId, targetType).Scan(&count)

			if err != nil {
				state.Logger.Error("Error while checking webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}

			if count >= 5 {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "You can only have 5 webhooks per entity at this time"},
				}
			}

			_, err = tx.Exec(d.Context, "INSERT INTO webhooks (target_id, target_type, url, secret, simple_auth, name, event_whitelist) VALUES ($1, $2, $3, $4, $5, $6, $7)", targetId, targetType, v.WebhookURL, v.WebhookSecret, v.SimpleAuth, v.Name, v.EventWhitelist)

			if err != nil {
				state.Logger.Error("Error while inserting webhook", zap.Error(err), zap.String("userID", d.Auth.ID))
				return uapi.DefaultResponse(http.StatusInternalServerError)
			}
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error while committing transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
