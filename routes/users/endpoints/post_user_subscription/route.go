package post_user_subscription

import (
	"encoding/json"
	"io"
	"net/http"
	"popplio/api"
	"popplio/constants"
	"popplio/docs"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/users/{id}/sub",
		OpId:        "post_user_subscription",
		Summary:     "Create User Subscription",
		Description: "Creates a user subscription for a push notification",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:      types.UserSubscription{},
		Resp:     types.ApiError{},
		Tags:     []string{api.CurrentTag},
		AuthType: []types.TargetType{types.TargetTypeUser},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var subscription types.UserSubscription

	var id = chi.URLParam(r, "id")

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(bodyBytes, &subscription)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- api.DefaultResponse(http.StatusInternalServerError)
		return
	}

	if subscription.Auth == "" || subscription.P256dh == "" {
		d.Resp <- api.DefaultResponse(http.StatusBadRequest)
		return
	}

	// Store new subscription
	notifId := utils.RandString(512)

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

	// Fan out test notification
	notifications.NotifChannel <- types.Notification{
		NotifID: notifId,
		Message: []byte(constants.TestNotif),
	}

	d.Resp <- api.HttpResponse{
		Status: http.StatusNoContent,
	}
}
