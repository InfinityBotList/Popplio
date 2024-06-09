package notifications

import (
	"encoding/json"
	"fmt"
	"popplio/state"
	"popplio/types"

	"github.com/SherClockHolmes/webpush-go"
	"go.uber.org/zap"
)

func PushNotification(userId string, notif types.Alert) error {
	err := state.Validator.Struct(notif)

	if err != nil {
		return fmt.Errorf("invalid notification: %s", err)
	}

	if len(notif.AlertData) == 0 {
		notif.AlertData = map[string]any{}
	}

	if notif.NoSave {
		_, err = state.Pool.Exec(
			state.Context,
			"INSERT INTO alerts (user_id, type, url, message, title, icon, alert_data, priority) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			userId,
			notif.Type,
			notif.URL,
			notif.Message,
			notif.Title,
			notif.Icon,
			notif.AlertData,
			notif.Priority,
		)

		if err != nil {
			state.Logger.Error("Error inserting alert", zap.Error(err), zap.String("user_id", userId), zap.Any("alert", notif))
			return err
		}
	}

	bytes, err := json.Marshal(notif)

	if err != nil {
		return err
	}

	notifIds, err := state.Pool.Query(state.Context, "SELECT notif_id, auth, endpoint, p256dh FROM user_notifications WHERE user_id = $1", userId)

	if err != nil {
		return err
	}

	defer notifIds.Close()

	for notifIds.Next() {
		var notifId string
		var auth string
		var endpoint string
		var p256dh string

		err = notifIds.Scan(&notifId, &auth, &endpoint, &p256dh)

		if err != nil {
			return fmt.Errorf("error finding notification: %s", err)
		}

		if notifId == "" {
			continue
		}

		state.Logger.Info("Sending notification", zap.String("notif_id", notifId), zap.String("endpoint", endpoint))

		sub := webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				Auth:   auth,
				P256dh: p256dh,
			},
		}

		resp, err := webpush.SendNotification(bytes, &sub, &webpush.Options{
			Subscriber:      "notifications@infinitybots.gg",
			VAPIDPublicKey:  state.Config.Notifications.VapidPublicKey,
			VAPIDPrivateKey: state.Config.Notifications.VapidPrivateKey,
			TTL:             30,
		})

		if err != nil {
			if resp.StatusCode == 410 || resp.StatusCode == 404 {
				// Delete the subscription
				state.Pool.Exec(state.Context, "DELETE FROM user_notifications WHERE notif_id = $1", notifId)
			}
			return err
		}
	}

	return nil
}
