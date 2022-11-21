package test_webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/list/webhook-test",
		OpId:        "webhook_test",
		Summary:     "Test Webhook",
		Description: "Sends a test webhook to allow testing your vote system. **All fields are mandatory for this endpoint**",
		Req:         types.WebhookPost{},
		Resp:        types.ApiError{},
		Tags:        []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	defer r.Body.Close()

	var payload types.WebhookPost

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(bodyBytes, &payload)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if utils.IsNone(payload.URL) && utils.IsNone(payload.URL2) {
		d.Resp <- api.DefaultResponse(http.StatusBadRequest)
		return
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

	d.Resp <- api.HttpResponse{
		Status: http.StatusBadRequest,
		Json:   errD,
	}

}
