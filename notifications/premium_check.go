package notifications

import (
	"popplio/state"
	"time"
)

func premiumCheck() {
	rows, err := state.Pool.Query(state.Context, "SELECT bot_id, start_premium_period, premium_period_length, type FROM bots WHERE premium = true")

	if err != nil {
		state.Logger.Error("Error finding bots: %s", err)
		return
	}

	for rows.Next() {
		// Check bot
		var botId string
		var startPremiumPeriod time.Time
		var premiumPeriodLength time.Duration
		var typeStr string

		err := rows.Scan(&botId, &startPremiumPeriod, &premiumPeriodLength, &typeStr)

		if err != nil {
			state.Logger.Error("Error decoding bot: %s", err)
			continue
		}

		if typeStr != "approved" && typeStr != "certified" {
			// This bot isnt approved, so we should remove premium
			state.Logger.Info("Removing premium from bot: ", botId)
			PremiumChannel <- botId
			continue
		}

		// Check start and sub period
		if time.Now().After(startPremiumPeriod.Add(premiumPeriodLength)) {
			state.Logger.Info("Removing premium from bot: ", botId)
			PremiumChannel <- botId
			continue
		}
	}

	rows.Close()
}
