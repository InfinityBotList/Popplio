package get_webhook_logs

import (
	docs "github.com/infinitybotlist/eureka/doclib"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhook Logs",
		Description: "Gets logs of a specific entity. The entity type is determined by the auth type used. Paginated to 20 at a time **Requires authentication**",
		Params: []docs.Parameter{
			{
				Name:        "target_id",
				Description: "The target ID to return webhook logs for",
				Required:    true,
				In:          "path",
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
