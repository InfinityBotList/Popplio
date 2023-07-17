package get_test_webhook_meta

import (
	"net/http"

	"popplio/routes/webhooks/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Test Webhook Metadata",
		Description: "Responds with the metadata of all webhooks that can currently be tested.",
		Resp:        types.GetTestWebhookMeta{},
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
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The entity type to return logs for. Must be `bot` or `team` (other entity types coming soon)",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := r.URL.Query().Get("target_type")
	targetId := chi.URLParam(r, "target_id")

	resp, ok := assets.CheckWebhookPermissions(
		d.Context,
		targetId,
		targetType,
		d.Auth.ID,
		assets.OpTestWebhooks,
	)

	if !ok {
		return resp
	}

	var data *types.GetTestWebhookMeta

	switch targetType {
	case "bot":
		// Check if webhooks v2
		var webhooksV2 bool

		err := state.Pool.QueryRow(d.Context, "SELECT webhooks_v2 FROM bots WHERE bot_id = $1", targetId).Scan(&webhooksV2)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		data = assets.GetTestMeta(targetId, targetType, webhooksV2)
	default:
		data = assets.GetTestMeta(targetId, targetType, false)
	}

	if data == nil {
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "This endpoint is not yet implemented for this target id/type combo"},
		}
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
