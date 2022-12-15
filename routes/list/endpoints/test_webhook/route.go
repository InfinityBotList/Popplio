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

	if utils.IsNone(payload.URL) && utils.IsNone(payload.URL2) {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	payload.Test = true // Always true

	var err1 error

	if !utils.IsNone(payload.URL) {
		err1 = webhooks.Send(payload)
	}

	var err2 error

	if !utils.IsNone(payload.URL2) {
		payload.URL = payload.URL2 // Test second enpdoint if it's not empty
		err2 = webhooks.Send(payload)
	}

	var errD = types.ApiError{}

	if err1 != nil {
		state.Logger.Error(err1)

		errD.Message = err1.Error()
		errD.Error = true
	}

	if err2 != nil {
		state.Logger.Error(err2)

		errD.Message += "|" + err2.Error()
		errD.Error = true
	}

	return api.HttpResponse{
		Status: http.StatusBadRequest,
		Json:   errD,
	}
}
