package test_webhook

import (
	"net/http"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/ratelimit"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = api.CompileValidationErrors(types.WebhookPost{})

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/list/webhook-test",
		OpId:        "test_webhook",
		Summary:     "Test Webhook",
		Description: "Sends a test webhook to allow testing your vote system. **All fields are mandatory for this endpoint**",
		Req:         types.WebhookPost{},
		Resp:        types.ApiError{},
		Tags:        []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      3 * time.Minute,
		MaxRequests: 10,
		Bucket:      "webh",
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

	var payload types.WebhookPost

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

	if utils.IsNone(payload.URL) {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	payload.Test = true // Always true

	if !utils.IsNone(payload.URL) {
		err := webhooks.Send(payload)

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
