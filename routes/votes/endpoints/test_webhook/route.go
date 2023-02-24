package test_webhook

import (
	"net/http"

	"popplio/api"
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
	isOwner, err := utils.IsBotOwner(d.Context, d.Auth.ID, id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
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
		BotID:  id,
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
