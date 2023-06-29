package notifications

import (
	"popplio/config"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func VrLoop() {
	if config.CurrentEnv != config.CurrentEnvProd {
		state.Logger.Info("Skipping vrCheck due to non-prod environment")
		return
	}

	for {
		//state.Logger.Debug("Running vrCheck")
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
			botInf, err := dovewing.GetUser(state.Context, botId, state.DovewingPlatformDiscord)

			if err != nil {
				state.Logger.Error("Error finding bot info:", err)
				continue
			}

			message := types.Alert{
				Type:    types.AlertTypeInfo,
				URL:     pgtype.Text{String: "https://infinitybotlist.com/bot/" + botId + "/vote", Valid: true},
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
