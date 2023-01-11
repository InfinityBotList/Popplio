package notifications

import (
	"popplio/state"
	"popplio/utils"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgtype"
)

var PremiumChannel = make(chan string)

func premium() {
	// Fans out premium removals
	for id := range PremiumChannel {
		state.Logger.Info("Removing premium bot: ", id)

		_, err := state.Pool.Exec(state.Context, "UPDATE bots SET premium = false, start_premium_period = $1, premium_period_length = $2 WHERE bot_id = $3", time.Now().UnixMilli(), 2592000000, id)

		if err != nil {
			state.Logger.Errorw("Error removing premium", "error", err, "bot_id", id)
			continue
		}

		// Send message
		botObj, err := utils.GetDiscordUser(id)

		if err != nil {
			state.Logger.Errorw("Error getting bot object", "error", err, "bot_id", id)
			continue
		}

		var owner pgtype.Text

		err = state.Pool.QueryRow(state.Context, "SELECT owner FROM bots WHERE bot_id = $1", id).Scan(&owner)

		if err != nil || !owner.Valid {
			state.Logger.Errorw("Error getting bot ownership info:", "error", err, "bot_id", id)
			continue
		}

		userObj, err := utils.GetDiscordUser(owner.String)

		if err != nil {
			state.Logger.Errorw("Error getting main owner info:", "error", err, "user_id", owner.String, "bot_id", id)
			continue
		}

		state.Discord.ChannelMessageSendComplex(state.Config.Channels.BotLogs, &discordgo.MessageSend{
			Content: botObj.Mention + "(" + botObj.Username + ") by " + userObj.Mention + " has been removed from the premium list as their subscription has expired [v4].",
		})

		dmChannel, err := state.Discord.UserChannelCreate(owner.String)

		if err != nil {
			state.Logger.Errorw("Error creating DM channel", "error", err, "user_id", owner.String, "bot_id", id)
			continue
		}

		state.Discord.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
			Content: "Your bot " + botObj.Mention + "(" + botObj.Username + ") has been removed from the premium list as your subscription has expired [v4].",
		})
	}
}
