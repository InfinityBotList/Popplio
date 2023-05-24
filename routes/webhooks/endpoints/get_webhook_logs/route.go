package get_webhook_logs

import (
	"net/http"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const perPage = 50

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhook Logs",
		Description: "Gets logs of a specific entity. The entity type is determined by the auth type used. Paginated to 50 at a time **Requires authentication**",
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
	return uapi.DefaultResponse(http.StatusNotImplemented)
}
