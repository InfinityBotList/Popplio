package notifications

import (
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"time"

	"github.com/infinitybotlist/dovewing"
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
			botInf, err := dovewing.GetDiscordUser(state.Context, botId)

			if err != nil {
				state.Logger.Error("Error finding bot info:", err)
				continue
			}

			message := types.Notification{
				Type:    types.NotificationTypeInfo,
				Message: "You can vote for " + botInf.Username + " now!",
				Title:   "Vote for " + botInf.Username + "!",
				Icon:    botInf.Avatar,
			}

			err = PushNotification(userId, message)

			if err != nil {
				state.Logger.Error(err)
				continue
			}

			_, err = state.Pool.Exec(state.Context, "UPDATE silverpelt SET last_acked = NOW() WHERE bot_id = $1 AND user_id = $2", botId, userId)
			if err != nil {
				state.Logger.Error("Error updating reminder: %s", err)
				continue
			}
		}
	}
}
