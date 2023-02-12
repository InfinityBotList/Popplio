package notifications

import (
	"popplio/state"
	"popplio/utils"
	"time"

	"github.com/bwmarrin/discordgo"
)

func premiumCheck() {
	rows, err := state.Pool.Query(
		state.Context,
		`SELECT bot_id, owner, start_premium_period, premium_period_length, type FROM bots 
		WHERE (
			premium = true
			AND (
				(type != 'approved' AND type != 'certified')
				OR (start_premium_period + premium_period_length) < NOW()
			)
		)`,
	)

	if err != nil {
		state.Logger.Error("[PremiumCheck] Error finding bots: %s", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		// Check bot
		var botId string
		var owner string
		var startPremiumPeriod time.Time
		var premiumPeriodLength time.Duration
		var typeStr string

		err := rows.Scan(&botId, &owner, &startPremiumPeriod, &premiumPeriodLength, &typeStr)

		if err != nil {
			state.Logger.Error("[PremiumCheck] Error decoding bot: %s", err)
			continue
		}

		state.Logger.Infow("[PremiumCheck] Removing premium bot", "bot_id", botId, "owner", owner, "start_premium_period", startPremiumPeriod, "premium_period_length", premiumPeriodLength, "type", typeStr)

		_, err = state.Pool.Exec(state.Context, "UPDATE bots SET premium = false WHERE bot_id = $1", botId)

		if err != nil {
			state.Logger.Errorw("[PremiumCheck] Error removing premium", "error", err, "bot_id", botId)
			continue
		}

		// Send message
		botObj, err := utils.GetDiscordUser(state.Context, botId)

		if err != nil {
			state.Logger.Errorw("[PremiumCheck] Error getting bot object", "error", err, "bot_id", botId)
			continue
		}

		userObj, err := utils.GetDiscordUser(state.Context, owner)

		if err != nil {
			state.Logger.Errorw("[PremiumCheck] Error getting main owner info:", "error", err, "user_id", owner, "bot_id", botId)
			continue
		}

		var msg string
		if typeStr != "approved" && typeStr != "certified" {
			msg = botObj.Mention + "(" + botObj.Username + ") by " + userObj.Mention + " has been removed from the premium list as their bot is neither approved or denied [v4]."
		} else {
			msg = botObj.Mention + "(" + botObj.Username + ") by " + userObj.Mention + " has been removed from the premium list as their subscription has expired [v4]."
		}

		_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
			Content: msg,
		})

		if err != nil {
			state.Logger.Errorw("[PremiumCheck] Error sending message", "error", err, "msg", msg)
			continue
		}
	}
}
