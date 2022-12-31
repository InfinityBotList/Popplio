package test_webhook

import (
	"io"
	"net/http"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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
	defer r.Body.Close()

	var payload types.WebhookPost

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = json.Unmarshal(bodyBytes, &payload)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if utils.IsNone(payload.URL) {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	payload.Test = true // Always true

	if !utils.IsNone(payload.URL) {
		err = webhooks.Send(payload)

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
