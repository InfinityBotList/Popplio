package notifications

import (
	"popplio/state"
	"popplio/utils"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func VrLoop() {
	for {
		state.Logger.Info("Running vrCheck")
		vrCheck()
		time.Sleep(10 * time.Second)
	}
}

func vrCheck() {
	rows, err := state.Pool.Query(state.Context, "SELECT user_id, bot_id FROM silverpelt WHERE NOW() - last_acked > interval '4 hours'")

	if err != nil {
		state.Logger.Error("Error finding reminders: ", err)
		return
	}

	for rows.Next() {
		var userId string
		var botId string
		err := rows.Scan(&userId, &botId)

		if err != nil {
			state.Logger.Error("Error decoding reminder:", err)
			continue
		}

		voteParsed, err := utils.GetVoteData(state.Context, userId, botId, true)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		if !voteParsed.HasVoted {
			// Loop over all user poppypaw subscriptions and push to goro
			rows, err := state.Pool.Query(state.Context, "SELECT notif_id, endpoint FROM poppypaw WHERE user_id = $1", userId)

			if err != nil {
				state.Logger.Error("Error finding subscriptions:", err)
				return
			}

			var notifs []struct {
				NotifId  string `db:"notif_id"`
				Endpoint string `db:"endpoint"`
			}

			err = pgxscan.ScanAll(&notifs, rows)

			if err != nil {
				state.Logger.Error("Error finding subscriptions:", err)
				return
			}

			botInf, err := utils.GetDiscordUser(state.Context, botId)

			if err != nil {
				state.Logger.Error("Error finding bot info:", err)
				continue
			}

			message := Message{
				Message: "You can vote for " + botInf.Username + " now!",
				Title:   "Vote for " + botInf.Username + "!",
				Icon:    botInf.Avatar,
			}

			bytes, err := json.Marshal(message)

			if err != nil {
				state.Logger.Error(err)
				continue
			}

			for _, notif := range notifs {
				if notif.NotifId == "" {
					continue
				}

				state.Logger.Infow("Sending notification", "notif_id", notif.NotifId)

				err := PushToClient(notif.NotifId, bytes)

				if err != nil {
					state.Logger.Error(err)
					continue
				}
			}

			_, err = state.Pool.Exec(state.Context, "UPDATE silverpelt SET last_acked = NOW() WHERE bot_id = $1 AND user_id = $2", botId, userId)
			if err != nil {
				state.Logger.Error("Error updating reminder: %s", err)
				continue
			}
		}
	}
}
