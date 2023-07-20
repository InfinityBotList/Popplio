package test_webhook

import (
	"net/http"
	"time"

	"popplio/ratelimit"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/events"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(WebhookAuthPost{})

type WebhookAuthPost struct {
	Votes int                     `json:"votes"`
	User  *dovetypes.PlatformUser `json:"user" description:"The user to use for testing webhooks"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Test Webhook",
		Description: "Sends a test webhook to allow testing our vote webhook system using the credentials you have set.",
		Req:         WebhookAuthPost{},
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
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: http.StatusNotImplemented,
		Json:   types.ApiError{Message: "This endpoint is not yet implemented"},
	}

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

	// Validate that they actually own this bot
	perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, id)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has(teams.TeamPermissionTestBotWebhooks) {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "You do not have permission to test this bot's webhooks"},
		}
	}

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

	return uapi.DefaultResponse(http.StatusNoContent)
}
