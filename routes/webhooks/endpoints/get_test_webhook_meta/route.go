package get_test_webhook_meta

import (
	"net/http"
	"slices"

	"popplio/types"
	"popplio/webhooks/core/events"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Test Webhook Metadata",
		Description: "Responds with the metadata of all webhooks that can currently be tested. Note that this does not require any specific permission",
		Resp:        types.GetTestWebhookMeta{},
		Params: []docs.Parameter{
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "event",
				Description: "The event that is being posted",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := r.URL.Query().Get("target_type")

	var data = types.GetTestWebhookMeta{}

	for _, evt := range events.Registry {
		evtTgtType := evt.Event.TargetTypes()
		if slices.Contains(evtTgtType, targetType) {
			data.Types = append(data.Types, types.TestWebhookType{
				Type: string(evt.Event.Event()),
				Data: evt.TestVars,
			})
		}
	}

	if len(data.Types) == 0 {
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "There are no available events for this target type"},
		}
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
