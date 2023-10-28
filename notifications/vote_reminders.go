package notifications

import (
	"fmt"
	"popplio/config"
	"popplio/state"
	"popplio/types"
	"popplio/votes"
	"time"

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
		err := vrCheck()

		if err != nil {
			state.Logger.Error(err)
			time.Sleep(5 * time.Minute)
			continue
		}

		time.Sleep(10 * time.Second)
	}
}

func vrCheck() error {
	rows, err := state.Pool.Query(state.Context, "SELECT user_id, target_id, target_type FROM user_reminders WHERE NOW() - last_acked > interval '4 hours'")

	if err != nil {
		return fmt.Errorf("error finding reminders: %w", err)
	}

	for rows.Next() {
		var userId string
		var targetId string
		var targetType string
		err := rows.Scan(&userId, &targetId, &targetType)

		if err != nil {
			state.Logger.Error("Error decoding reminder:", err)
			continue
		}

		vi, err := votes.EntityVoteCheck(state.Context, userId, targetId, targetType)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		if !vi.HasVoted {
			entityInfo, err := votes.GetEntityInfo(state.Context, targetId, targetType)

			if err != nil {
				state.Logger.Error("Error finding bot info:", err, targetId, targetType)
				continue
			}

			message := types.Alert{
				Type:    types.AlertTypeInfo,
				URL:     pgtype.Text{String: entityInfo.VoteURL, Valid: true},
				Message: "You can vote for the " + targetType + " " + entityInfo.Name + " now!",
				Title:   "Vote for " + entityInfo.Name + "!",
				Icon:    entityInfo.Avatar,
			}

			err = PushNotification(userId, message)

			if err != nil {
				state.Logger.Error(err)
				continue
			}

			_, err = state.Pool.Exec(state.Context, "UPDATE user_reminders SET last_acked = NOW() WHERE user_id = $1 AND target_id = $2 AND target_type = $3", userId, targetId, targetType)
			if err != nil {
				state.Logger.Error("Error updating reminder: %s", err)
				continue
			}

		}
	}

	return nil
}
