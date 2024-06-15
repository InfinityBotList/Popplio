package test_webhook

import (
	"net/http"
	"reflect"
	"slices"
	"time"

	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/core/events"

	"github.com/infinitybotlist/eureka/ratelimit"
	"go.uber.org/zap"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Test Webhook",
		Description: "Sends a test webhook.",
		Req:         map[string]any{},
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
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))
	targetId := chi.URLParam(r, "target_id")
	eventType := r.URL.Query().Get("event")

	if eventType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "event must be specified",
			},
		}
	}

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 3,
		Bucket:      "test_webhook",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error("Error while ratelimiting", zap.Error(err), zap.String("bucket", "test_webhook"))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	var w events.WebhookEvent

	for _, evt := range events.Registry {
		if string(evt.Event.Event()) == eventType {
			tgtTypes := evt.Event.TargetTypes()
			if !slices.Contains(tgtTypes, targetType) {
				return uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json: types.ApiError{
						Message: "This event is not valid for this target type",
					},
				}
			}

			w = evt.Event
		}
	}

	if w == nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This event does not exist",
			},
		}
	}

	event := reflect.New(reflect.TypeOf(w)).Interface().(events.WebhookEvent)

	// JSON serialize the event from request body
	hresp, ok := uapi.MarshalReqWithHeaders(r, event, limit.Headers())

	if !ok {
		return hresp
	}

	state.Logger.Info("Sending test webhook", zap.String("userID", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType), zap.String("eventType", eventType))

	err = drivers.Send(drivers.With{
		UserID:     d.Auth.ID,
		TargetID:   targetId,
		TargetType: targetType,
		Data:       event,
		Metadata: &events.WebhookMetadata{
			Test: true,
		},
	})

	if err != nil {
		state.Logger.Error("Error while sending webhook", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType), zap.String("eventType", eventType))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
