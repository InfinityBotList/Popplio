package test_webhook

import (
	"net/http"
	"reflect"
	"time"

	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/events"
	"popplio/webhooks/teamhooks"

	"github.com/infinitybotlist/eureka/uapi/ratelimit"

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
		state.Logger.Error(err)
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

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

	if err != nil {
		state.Logger.Error(err)
		return uapi.HttpResponse{
			Status:  http.StatusBadRequest,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !perms.Has(targetType, teams.PermissionTestWebhooks) {
		return uapi.HttpResponse{
			Status:  http.StatusForbidden,
			Headers: limit.Headers(),
			Json:    types.ApiError{Message: "You do not have permission to test webhooks on this entity"},
		}
	}

	var w events.WebhookEvent

	for _, evt := range events.Registry {
		if string(evt.Event.Event()) == eventType {
			if evt.Event.TargetType() != targetType {
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

	switch targetType {
	case "bot":
		err := bothooks.Send(bothooks.With{
			UserID: d.Auth.ID,
			BotID:  targetId,
			Data:   event,
			Metadata: &events.WebhookMetadata{
				Test: true,
			},
		})

		if err != nil {
			state.Logger.Error(err)
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: err.Error()},
			}
		}
	case "team":
		err := teamhooks.Send(teamhooks.With{
			UserID: d.Auth.ID,
			TeamID: targetId,
			Data:   event,
			Metadata: &events.WebhookMetadata{
				Test: true,
			},
		})

		if err != nil {
			state.Logger.Error(err)
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: err.Error()},
			}
		}
	default:
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Invalid target type"},
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
