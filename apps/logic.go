package apps

import (
	"errors"
	"fmt"
	"popplio/api"
	"popplio/state"
	"popplio/utils"

	"github.com/bwmarrin/discordgo"
)

func extraLogicResubmit(d api.RouteData, p Position, answers map[string]string) (add bool, err error) {
	// Get the bot ID
	botID, ok := answers["id"]

	if !ok {
		return false, errors.New("bot ID not found")
	}

	// Get the bot
	var botType string
	err = state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", botID).Scan(&botType)

	if err != nil {
		return false, fmt.Errorf("error getting bot type, does the bot exist?: %w", err)
	}

	owner, err := utils.IsBotOwner(d.Context, d.Auth.ID, botID)

	if err != nil {
		return false, fmt.Errorf("error checking if user is bot owner: %w", err)
	}

	if !owner {
		return false, errors.New("you are not the owner of this bot")
	}

	if botType == "approved" || botType == "pending" || botType == "certified" {
		return false, errors.New("bot is approved, pending or certified | state=" + botType)
	}

	// Set the bot type to pending
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET type = 'pending', claimed_by = NULL WHERE bot_id = $1", botID)

	if err != nil {
		return false, fmt.Errorf("error setting bot type to pending: %w", err)
	}

	user, err := utils.GetDiscordUser(d.Context, botID)

	if err != nil {
		return false, fmt.Errorf("error getting discord user: %w", err)
	}

	// Send an embed to the bot logs channel
	_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.BotLogs, &discordgo.MessageSend{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Bot Resubmitted!",
				URL:         state.Config.Sites.Frontend + "/bots/" + botID,
				Description: "User <@" + d.Auth.ID + "> has resubmitted their bot",
				Color:       0x00ff00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Bot ID",
						Value: botID,
					},
					{
						Name:  "Bot Name",
						Value: user.Username + "#" + user.Discriminator + " (" + user.ID + ")",
					},
					{
						Name:   "Reason",
						Value:  answers["reason"],
						Inline: true,
					},
				},
			},
		},
	})

	if err != nil {
		return false, fmt.Errorf("error sending embed to bot logs channel: %w", err)
	}

	// We don't want to actually create an application
	return false, nil
}

func extraLogicCert(d api.RouteData, p Position, answers map[string]string) (add bool, err error) {
	// Get the bot ID
	botID, ok := answers["id"]

	if !ok {
		return false, errors.New("bot ID not found")
	}

	// Get the bot
	var botType string
	err = state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", botID).Scan(&botType)

	if err != nil {
		return false, fmt.Errorf("error getting bot type, does the bot exist?: %w", err)
	}

	owner, err := utils.IsBotOwner(d.Context, d.Auth.ID, botID)

	if err != nil {
		return false, fmt.Errorf("error checking if user is bot owner: %w", err)
	}

	if !owner {
		return false, errors.New("you are not the owner of this bot")
	}

	if botType != "approved" {
		return false, errors.New("bot is not approved | state=" + botType)
	}

	// Now check server count and unique clicks
	var serverCount int64
	var uniqueClicks int64
	err = state.Pool.QueryRow(d.Context, "SELECT servers, cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", botID).Scan(&serverCount, &uniqueClicks)

	if err != nil {
		return false, fmt.Errorf("error getting server count: %w", err)
	}

	if serverCount < 100 {
		return false, errors.New("bot does not have enough servers to be certified: has " + fmt.Sprint(serverCount) + ", needs 100")
	}

	if uniqueClicks < 30 {
		return false, errors.New("bot does not have enough unique clicks to be certified: has " + fmt.Sprint(uniqueClicks) + ", needs 30")
	}

	return true, nil
}
