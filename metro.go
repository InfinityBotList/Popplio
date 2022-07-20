package main

import (
	"errors"
	"os"
	"popplio/utils"
	"regexp"
	"strings"
	"time"

	popltypes "popplio/types"

	"github.com/MetroReviews/metro-integrase/types"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

var regex *regexp.Regexp
var metro *discordgo.Session

func init() {
	var err error
	regex, err = regexp.Compile("[^a-zA-Z0-9]")

	if err != nil {
		panic(err)
	}
}

func addBot(bot *types.Bot) (pgconn.CommandTag, error) {
	prefix := bot.Prefix

	if prefix == "" {
		prefix = "/"
	}

	invite := bot.Invite

	if invite == "" {
		invite = "https://discord.com/oauth2/authorize?client_id=" + bot.BotID + "&permissions=0&scope=bot%20applications.commands"
	}

	_, err := pool.Exec(ctx, "DELETE FROM bots WHERE bot_id = $1", bot.BotID)

	if err != nil {
		log.Error(err)
	}

	return pool.Exec(
		ctx,
		`INSERT INTO bots (bot_id, name, vanity, approval_note, date, prefix, website, github, donate, nsfw, library, 
			cross_add, list_source, external_source, short, long, tags, invite, owner, additional_owners,
			web_auth, custom_webhook, webhook, token, type) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, 
				$18, $19, $20, $21, $22, $23, $24, $25
			)`,
		bot.BotID,
		bot.Username,
		strings.ToLower(regex.ReplaceAllString(bot.Username, "")),
		"Metro-approved",
		time.Now(),
		prefix,
		bot.Website,
		bot.Github,
		bot.Donate,
		bot.NSFW,
		bot.Library,
		bot.CrossAdd,
		bot.ListSource,
		"metro_reviews",
		bot.Description,
		bot.LongDescription,
		bot.Tags,
		invite,
		bot.Owner,
		bot.ExtraOwners,
		"",
		"",
		"",
		utils.RandString(128),
		"pending",
	)
}

// Returns empty string if bot doesn't exist
func getBotType(id string) string {
	var botType pgtype.Text

	err := pool.QueryRow(ctx, `SELECT type FROM bots WHERE bot_id = $1`, id).Scan(&botType)

	if err != nil {
		log.Error(err)
	}

	return botType.String
}

// Dummy adapter backend
type DummyAdapter struct {
}

func (adp DummyAdapter) GetConfig() types.ListConfig {
	return types.ListConfig{
		SecretKey:   os.Getenv("SECRET_KEY"),
		ListID:      os.Getenv("LIST_ID"),
		RequestLogs: true,
		StartupLogs: true,
		BindAddr:    ":8081",
		DomainName:  "",
	}
}

func (adp DummyAdapter) ClaimBot(bot *types.Bot) error {
	log.Info("Called ClaimBot")
	if bot == nil {
		return errors.New("bot is nil")
	}

	_, err := addBot(bot)

	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `UPDATE bots SET claimed = true, claimed_by = $1 WHERE bot_id = $2`, bot.Reviewer, bot.BotID)

	if err != nil {
		return err
	}

	return nil
}

func (adp DummyAdapter) UnclaimBot(bot *types.Bot) error {
	log.Info("Called UnclaimBot")
	if bot == nil {
		return errors.New("bot is nil")
	}

	_, err := addBot(bot)

	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `UPDATE bots SET claimed = false, claimed_by = NULL WHERE bot_id = $1`, bot.BotID)

	if err != nil {
		return err
	}

	return nil
}

func (adp DummyAdapter) ApproveBot(bot *types.Bot) error {
	log.Info("Called ApproveBot")
	if bot == nil {
		return errors.New("bot is nil")
	}

	// Check if bot already exists on DB
	botType := getBotType(bot.BotID)

	if botType == "" {
		if !bot.CrossAdd && bot.ListSource != os.Getenv("LIST_ID") {
			return errors.New("bot is not from the correct source")
		}

		res, err := addBot(bot)

		if err != nil {
			return err
		}

		log.Info("Added bot: ", res.RowsAffected())

	} else {
		if botType != "pending" {
			return errors.New("bot 'type' is not pending")
		}
	}

	res, err := pool.Exec(ctx, `UPDATE bots SET type = 'approved' WHERE bot_id = $1`, bot.BotID)

	if err != nil {
		return err
	}

	log.Info("Updated ", res.RowsAffected(), " bots")

	messageNotifyChannel <- popltypes.DiscordLog{
		ChannelID: os.Getenv("CHANNEL_ID"),
		Message: &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "**__Bot Approved:__**",
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://cdn.discordapp.com/attachments/815094858439065640/972734471369527356/FD34E31D-BFBC-4B96-AEDB-0ECB16F49314.png",
					},
					Color: 0x00FF00,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Bot:",
							Value:  "<@" + bot.BotID + ">",
							Inline: true,
						},
						{
							Name:   "Owner:",
							Value:  "<@" + bot.Owner + ">",
							Inline: true,
						},
						{
							Name:   "Moderator:",
							Value:  "<@" + bot.Reviewer + ">",
							Inline: true,
						},
						{
							Name:  "Feedback:",
							Value: bot.Reason,
						},
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text: "© Copyright 2021 - 2022 - Metro Reviewer",
					},
					Timestamp: time.Now().Format(time.RFC3339),
				},
			},
		},
	}

	return nil
}

func (adp DummyAdapter) DenyBot(bot *types.Bot) error {
	log.Info("Called DenyBot")
	if bot == nil {
		return errors.New("bot is nil")
	}

	// Check if bot already exists on DB
	botType := getBotType(bot.BotID)

	if botType == "" {
		if !bot.CrossAdd && bot.ListSource != os.Getenv("LIST_ID") {
			return errors.New("bot is not from the correct source")
		}

		res, err := addBot(bot)

		if err != nil {
			return err
		}

		log.Info("Added bot: ", res.RowsAffected())

	} else {
		if botType != "pending" {
			return errors.New("bot 'type' is not pending")
		}
	}

	res, err := pool.Exec(ctx, `UPDATE bots SET type = 'denied' WHERE bot_id = $1`, bot.BotID)

	if err != nil {
		return err
	}

	log.Info("Updated ", res.RowsAffected(), " bots")

	messageNotifyChannel <- popltypes.DiscordLog{
		ChannelID: os.Getenv("CHANNEL_ID"),
		Message: &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "**__Bot Denied:__**",
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://cdn.discordapp.com/attachments/815094858439065640/972734471369527356/FD34E31D-BFBC-4B96-AEDB-0ECB16F49314.png",
					},
					Color: 0xFF0000,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Bot:",
							Value:  "<@" + bot.BotID + ">",
							Inline: true,
						},
						{
							Name:   "Owner:",
							Value:  "<@" + bot.Owner + ">",
							Inline: true,
						},
						{
							Name:   "Moderator:",
							Value:  "<@" + bot.Reviewer + ">",
							Inline: true,
						},
						{
							Name:  "Reason:",
							Value: bot.Reason,
						},
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text: "© Copyright 2021 - 2022 - Metro Reviewer",
					},
					Timestamp: time.Now().Format(time.RFC3339),
				},
			},
		},
	}

	return nil
}

func (adp DummyAdapter) DataDelete(id string) error {
	return nil
}

func (adp DummyAdapter) DataRequest(id string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id": id,
	}, nil
}
