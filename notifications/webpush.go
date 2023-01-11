package notifications

import (
	"io"
	"popplio/state"

	"github.com/SherClockHolmes/webpush-go"
)

type Notification struct {
	NotifID string
	Message []byte
}

var (
	NotifChannel = make(chan Notification)
)

func webPush() {
	/* This channel is used to send a notification to the client

	   In order to so, we create a always running goroutine responsible for fanning out notifications

	   Vote reminders is a seperate goroutine
	*/
	for msg := range NotifChannel {
		var auth string
		var endpoint string
		var p256dh string

		err := state.Pool.QueryRow(state.Context, "SELECT auth, endpoint, p256dh FROM poppypaw WHERE notif_id = $1", msg.NotifID).Scan(&auth, &endpoint, &p256dh)

		if err != nil {
			state.Logger.Error("Error finding notification: %s", err)
			continue
		}

		sub := webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				Auth:   auth,
				P256dh: p256dh,
			},
		}

		// Send Notification
		resp, err := webpush.SendNotification(msg.Message, &sub, &webpush.Options{
			Subscriber:      "votereminders@infinitybots.gg",
			VAPIDPublicKey:  state.Config.Notifications.VapidPublicKey,
			VAPIDPrivateKey: state.Config.Notifications.VapidPrivateKey,
			TTL:             30,
		})
		if err != nil {
			// TODO: Handle error
			if resp.StatusCode == 410 || resp.StatusCode == 404 {
				// Delete the subscription
				state.Pool.Exec(state.Context, "DELETE FROM poppypaw WHERE notif_id = $1", msg.NotifID)
			}
			state.Logger.Error(err)
			continue
		}

		msg, _ := io.ReadAll(resp.Body)
		state.Logger.Info(resp.StatusCode, msg)
		resp.Body.Close()
	}
}
