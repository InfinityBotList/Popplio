package test_vote_webhook

import (
	"math/rand"
	"net/http"
	"time"

	"popplio/ratelimit"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks/bothooks"
	"popplio/webhooks/bothooks_legacy"
	"popplio/webhooks/events"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(WebhookAuthPost{})

type WebhookAuthPost struct {
	Votes int `json:"votes" validate:"required"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Test Vote Webhook",
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
				Name:        "bid",
				Description: "The bots ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
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
			Json:   types.ApiError{Message: "You do not have permission to test this bot's webhooks", Error: true},
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

	var webhooksV2 bool

	err = state.Pool.QueryRow(d.Context, "SELECT webhooks_v2 FROM bots WHERE bot_id = $1", id).Scan(&webhooksV2)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
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
				Error:   true,
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	if webhooksV2 {
		err = bothooks.Send(bothooks.With[events.WebhookBotVoteData]{
			Data: events.WebhookBotVoteData{
				Votes: payload.Votes,
				Test:  true,
			},
			UserID: d.Auth.ID,
			BotID:  id,
		})

		if err != nil {
			state.Logger.Error(err)
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Error:   true,
					Message: err.Error(),
				},
			}
		}

		return uapi.DefaultResponse(http.StatusNoContent)
	} else {
		if rand.Float64() < 0.1 {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Error:   true,
					Message: "webhooks v1 is deprecated and so this endpoint will error sometimes to ensure visibility",
				},
			}
		}

		err = bothooks_legacy.SendLegacy(bothooks_legacy.WebhookPostLegacy{
			UserID: d.Auth.ID,
			BotID:  id,
			Votes:  payload.Votes,
			Test:   true,
		})

		if err != nil {
			state.Logger.Error(err)

			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Error:   true,
					Message: err.Error(),
				},
			}
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
