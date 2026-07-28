package create_user_notifications

import (
	"net/http"

	"popplio/notifications"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/crypto"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create User Notification",
		Description: "Creates a new subscription for a push notification. Returns 204 on success",
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

	hresp, ok := uapi.MarshalReq(r, &subscription)

	if !ok {
		return hresp
	}

	var id = chi.URLParam(r, "id")

	if subscription.Auth == "" || subscription.P256dh == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	// Store new subscription
	notifId := crypto.RandString(64)

	ua := r.UserAgent()

	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	tx.Exec(d.Context, "DELETE FROM user_notifications WHERE user_id = $1 AND endpoint = $2", id, subscription.Endpoint)

	tx.Exec(
		d.Context,
		"INSERT INTO user_notifications (user_id, notif_id, auth, p256dh, endpoint, ua) VALUES ($1, $2, $3, $4, $5, $6)",
		id,
		notifId,
		subscription.Auth,
		subscription.P256dh,
		subscription.Endpoint,
		ua,
	)

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error while committing transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Fan out notification
	err = notifications.PushNotification(id, types.Alert{
		Type:    types.AlertTypeSuccess,
		Title:   "New Subscription",
		Message: "This is an automated message to let you know that you have successfully subscribed to push notifications!",
	})

	if err != nil {
		state.Logger.Error("Error while sending push notification", zap.Error(err), zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
