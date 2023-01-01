package test_webhook

import (
	"net/http"

	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = api.CompileValidationErrors(WebhookAuthPost{})

type WebhookAuthPost struct {
	Votes int `json:"votes" validate:"required"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/users/{uid}/bots/{bid}/test-webhook",
		OpId:        "test_webhook",
		Summary:     "Test Webhook",
		Description: "Sends a test webhook to allow testing your vote system using the credentials you have set.",
		Req:         WebhookAuthPost{},
		Resp:        types.ApiError{},
		Tags:        []string{api.CurrentTag},
		AuthType:    []types.TargetType{types.TargetTypeUser},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	// Check if the user is owner of the bot
	botIdParam := chi.URLParam(r, "bid")

	// Resolve id
	var botId string

	err := state.Pool.QueryRow(d.Context, "SELECT bot_id FROM bots WHERE "+constants.ResolveBotSQL, botIdParam).Scan(&botId)

	if err != nil {
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Validate that they actually own this bot
	isOwner, err := utils.IsBotOwner(d.Context, d.Auth.ID, botId)

	if err != nil {
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Owner find error: " + err.Error(), Error: true},
		}
	}

	if !isOwner {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "You do not own the bot you are trying to manage", Error: true},
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

	webhPayload := types.WebhookPost{
		UserID: d.Auth.ID,
		BotID:  botId,
		Votes:  payload.Votes,
		Test:   true,
	}

	err = webhooks.Send(webhPayload)

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
}
