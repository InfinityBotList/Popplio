package test_webhook

import (
	"math/rand"
	"net/http"
	"time"

	"popplio/api"
	"popplio/ratelimit"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks/bothooks"
	legacyhooks "popplio/webhooks/bothooks/legacy"
	"popplio/webhooks/events"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = api.CompileValidationErrors(WebhookAuthPost{})

type WebhookAuthPost struct {
	Votes int `json:"votes" validate:"required"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Test Webhook",
		Description: "Sends a test webhook to allow testing your vote system using the credentials you have set.",
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	name := chi.URLParam(r, "bid")

	// Resolve bot ID
	id, err := utils.ResolveBot(state.Context, name)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if id == "" {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Validate that they actually own this bot
	perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if !perms.Has(teams.TeamPermissionTestBotWebhooks) {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "You do not have permission to test this bot's webhooks", Error: true},
		}
	}

	var payload WebhookAuthPost

	resp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return resp
	}

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	var webhooksV2 bool

	err = state.Pool.QueryRow(d.Context, "SELECT webhooks_v2 FROM bots WHERE bot_id = $1", id).Scan(&webhooksV2)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 3,
		Bucket:      "test_webhook",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	if webhooksV2 {
		err = bothooks.CreateHook{
			Type: events.WebhookTypeBotVote,
			Data: events.WebhookBotVoteData{
				Votes: payload.Votes,
				Test:  true,
			},
		}.With(bothooks.With{
			UserID: d.Auth.ID,
			BotID:  id,
		}).Send()

		if err != nil {
			state.Logger.Error(err)
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Error:   true,
					Message: err.Error(),
				},
			}
		}

		return api.DefaultResponse(http.StatusNoContent)
	} else {
		if rand.Float64() < 0.1 {
			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Error:   true,
					Message: "webhooks v1 is deprecated and so this endpoint will error sometimes to ensure visibility",
				},
			}
		}

		err = legacyhooks.SendLegacy(legacyhooks.WebhookPostLegacy{
			UserID: d.Auth.ID,
			BotID:  id,
			Votes:  payload.Votes,
			Test:   true,
		})

		if err != nil {
			state.Logger.Error(err)

			return api.HttpResponse{
				Status: http.StatusBadRequest,
				Json: types.ApiError{
					Error:   true,
					Message: err.Error(),
				},
			}
		}
	}

	return api.DefaultResponse(http.StatusNoContent)
}
