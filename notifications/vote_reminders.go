package notifications

import (
	"popplio/state"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Message struct {
	Message string `json:"message"`
	Title   string `json:"title"`
	Icon    string `json:"icon"`
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

			botInf, err := utils.GetDiscordUser(botId)

			if err != nil {
				state.Logger.Error("Error finding bot info:", err)
				return
			}

			message := Message{
				Message: "You can vote for " + botInf.Username + " now!",
				Title:   "Vote for " + botInf.Username + "!",
				Icon:    botInf.Avatar,
			}

			bytes, err := json.Marshal(message)

			if err != nil {
				state.Logger.Error(err)
				return
			}

			for _, notif := range notifs {
				if notif.NotifId == "" {
					continue
				}

				state.Logger.Info("NotifID: ", notif.NotifId)

				NotifChannel <- Notification{
					NotifID: notif.NotifId,
					Message: bytes,
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
