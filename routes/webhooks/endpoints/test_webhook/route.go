package test_webhook

import (
	"net/http"
	"strings"
	"time"

	"popplio/routes/webhooks/assets"
	"popplio/state"
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

	if !strings.HasPrefix(eventType, strings.ToUpper(targetType)+"_") {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This event is not valid for this target type",
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

	resp, ok := assets.CheckWebhookPermissions(
		d.Context,
		targetId,
		targetType,
		d.Auth.ID,
		assets.OpTestWebhooks,
	)

	if !ok {
		resp.Headers = limit.Headers()
		return resp
	}

	var hresp uapi.HttpResponse

	switch events.WebhookType(eventType) {
	case events.WebhookTypeBotEditReview:
		hresp, ok = handle[events.WebhookBotEditReviewData](d, r, limit)
	case events.WebhookTypeBotNewReview:
		hresp, ok = handle[events.WebhookBotNewReviewData](d, r, limit)
	case events.WebhookTypeBotVote:
		hresp, ok = handle[events.WebhookBotVoteData](d, r, limit)
	case events.WebhookTypeTeamEdit:
		hresp, ok = handle[events.WebhookTeamEditData](d, r, limit)
	default:
		return uapi.HttpResponse{
			Status:  http.StatusNotImplemented,
			Headers: limit.Headers(),
			Json: types.ApiError{
				Message: "This event is not implemented yet",
			},
		}
	}

	if !ok {
		return hresp
	}

	return uapi.DefaultResponse(http.StatusNoContent)
	/*
		var payload WebhookAuthPost

		resp, ok := uapi.MarshalReq(r, &payload)

		if !ok {
			return resp
		}

		// Validate the payload
		err = state.Validator.Struct(payload)

		if err != nil {
			errors := err.(validator.ValidationErrors)
			return uapi.ValidatorErrorResponse(compiledMessages, errors)
		}

		err = bothooks.Send(bothooks.With[events.WebhookBotVoteData]{
			Data: events.WebhookBotVoteData{
				Votes: payload.Votes,
			},
			UserID: d.Auth.ID,
			BotID:  id,
			Metadata: &events.WebhookMetadata{
				Test: true,
			},
		})

		if err != nil {
			state.Logger.Error(err)
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Message: err.Error(),
				},
			}
		}

		return uapi.DefaultResponse(http.StatusNoContent)*/
}

func handle[T events.WebhookEvent](
	d uapi.RouteData,
	r *http.Request,
	limit ratelimit.Limit,
) (uapi.HttpResponse, bool) {
	targetType := r.URL.Query().Get("target_type")
	targetId := chi.URLParam(r, "target_id")
	eventType := r.URL.Query().Get("event")

	var event T

	if eventType != string(event.Event()) {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Internal error: eventType != event.Event()",
			},
		}, false
	}

	hresp, ok := uapi.MarshalReqWithHeaders(r, &event, limit.Headers())

	if !ok {
		return hresp, ok
	}

	switch targetType {
	case "bot":
		err := bothooks.Send(bothooks.With[T]{
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
			}, false
		}
	case "team":
		err := teamhooks.Send(teamhooks.With[T]{
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
			}, false
		}
	}

	return hresp, true
}
