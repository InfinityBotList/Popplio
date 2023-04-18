package post_user_subscription

import (
	"io"
	"net/http"

	"popplio/notifications"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/crypto"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create User Subscription",
		Description: "Creates a user subscription for a push notification. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.UserSubscription{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var subscription types.UserSubscription

	var id = chi.URLParam(r, "id")

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = json.Unmarshal(bodyBytes, &subscription)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if subscription.Auth == "" || subscription.P256dh == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	// Store new subscription
	notifId := crypto.RandString(64)

	ua := r.UserAgent()

	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"
	}

	state.Pool.Exec(d.Context, "DELETE FROM poppypaw WHERE user_id = $1 AND endpoint = $2", id, subscription.Endpoint)

	state.Pool.Exec(
		d.Context,
		"INSERT INTO poppypaw (user_id, notif_id, auth, p256dh, endpoint, ua) VALUES ($1, $2, $3, $4, $5, $6)",
		id,
		notifId,
		subscription.Auth,
		subscription.P256dh,
		subscription.Endpoint,
		ua,
	)

	// Fan out notification
	err = notifications.PushNotification(id, types.Alert{
		Type:    types.AlertTypeSuccess,
		Title:   "New Subscription",
		Message: "This is an automated message to let you know that you have successfully subscribed to push notifications!",
	})

	if err != nil {
		state.Logger.Error(err)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
