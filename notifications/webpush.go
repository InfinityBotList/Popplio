package notifications

import (
	"fmt"
	"io"
	"popplio/state"

	"github.com/SherClockHolmes/webpush-go"
)

type Message struct {
	Message string `json:"message"`
	Title   string `json:"title"`
	Icon    string `json:"icon"`
}

func PushToClient(notifId string, message []byte) error {
	var auth string
	var endpoint string
	var p256dh string

	err := state.Pool.QueryRow(state.Context, "SELECT auth, endpoint, p256dh FROM poppypaw WHERE notif_id = $1", notifId).Scan(&auth, &endpoint, &p256dh)

	if err != nil {
		return fmt.Errorf("error finding notification: %s", err)
	}

	sub := webpush.Subscription{
		Endpoint: endpoint,
		Keys: webpush.Keys{
			Auth:   auth,
			P256dh: p256dh,
		},
	}

	// Send Notification
	resp, err := webpush.SendNotification(message, &sub, &webpush.Options{
		Subscriber:      "votereminders@infinitybots.gg",
		VAPIDPublicKey:  state.Config.Notifications.VapidPublicKey,
		VAPIDPrivateKey: state.Config.Notifications.VapidPrivateKey,
		TTL:             30,
	})

	if err != nil {
		// TODO: Handle error
		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			// Delete the subscription
			state.Pool.Exec(state.Context, "DELETE FROM poppypaw WHERE notif_id = $1", notifId)
		}
		return err
	}

	defer resp.Body.Close()

	msg, _ := io.ReadAll(resp.Body)
	state.Logger.Info(resp.StatusCode, msg)

	return nil
}
