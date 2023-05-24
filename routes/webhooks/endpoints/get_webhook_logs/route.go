package get_webhook_logs

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const perPage = 50

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhook Logs",
		Description: "Gets logs of a specific entity. The entity type is determined by the auth type used. Paginated to 50 at a time. **Requires authentication**",
		Resp:        types.PagedResult[types.WebhookLogEntry]{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID to return webhook logs for",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The entity type to return logs for. Must be `bot` (other entity types coming soon)",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := r.URL.Query().Get("target_type")

	switch targetType {
	case "BOT":
		// Check that they own the bot
		name := chi.URLParam(r, "bid")

		// Resolve bot ID
		id, err := utils.ResolveBot(state.Context, name)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if id == "" {
			return uapi.DefaultResponse(http.StatusNotFound)
		}

		perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, id)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if !perms.Has(teams.TeamPermissionGetBotWebhookLogs) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to get webhook logs", Error: true},
			}
		}
	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "This entity type is not supported yet", Error: true},
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusNotImplemented,
		Json:   types.ApiError{Message: "This endpoint is not implemented yet", Error: true},
	}
}
