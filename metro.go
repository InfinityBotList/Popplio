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
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

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

func addBot(bot *types.Bot) (*mongo.InsertOneResult, error) {
	col := mongoDb.Collection("bots")

	prefix := bot.Prefix

	if prefix == "" {
		prefix = "/"
	}

	invite := bot.Invite

	if invite == "" {
		invite = "https://discord.com/oauth2/authorize?client_id=" + bot.BotID + "&permissions=0&scope=bot%20applications.commands"
	}

	// Check if bot exists in DB
	count, err := col.CountDocuments(ctx, bson.M{"botID": bot.BotID})

	if err != nil {
		return nil, err
	}

	if count > 0 {
		col.DeleteOne(ctx, bson.M{"botID": bot.BotID})
	}

	res, err := col.InsertOne(ctx, bson.M{
		"botID":             bot.BotID,
		"botName":           bot.Username,
		"vanity":            strings.ToLower(regex.ReplaceAllString(bot.Username, "")),
		"note":              "Metro-approved",
		"date":              time.Now().UnixMilli(),
		"prefix":            prefix,
		"website":           bot.Website,
		"github":            bot.Github,
		"donate":            bot.Donate,
		"nsfw":              bot.NSFW,
		"library":           bot.Library,
		"crossAdd":          bot.CrossAdd,
		"listSource":        bot.ListSource,
		"external_source":   "metro_reviews",
		"short":             bot.Description,
		"long":              bot.LongDescription,
		"tags":              strings.Join(bot.Tags, ","),
		"invite":            invite,
		"main_owner":        bot.Owner,
		"additional_owners": bot.ExtraOwners,
		"webAuth":           "",
		"webURL":            "",
		"webhook":           "",
		"token":             utils.RandString(128),
		"type":              "pending",
	})

	return res, err
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
		BindAddr:    ":8080",
		DomainName:  "https://api.infinitybotlist.com",
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

	col := mongoDb.Collection("bots")

	_, err = col.UpdateOne(ctx, bson.M{"botID": bot.BotID}, bson.M{"$set": bson.M{
		"claimed":   true,
		"claimedBY": bot.Reviewer,
	}})

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

	col := mongoDb.Collection("bots")

	_, err = col.UpdateOne(ctx, bson.M{"botID": bot.BotID}, bson.M{"$set": bson.M{
		"claimed":   false,
		"claimedBY": "",
	}})

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
	col := mongoDb.Collection("bots")

	mongoBot := col.FindOne(ctx, bson.M{"botID": bot.BotID})

	if mongoBot.Err() == mongo.ErrNoDocuments {
		if !bot.CrossAdd && bot.ListSource != os.Getenv("LIST_ID") {
			return errors.New("bot is not from the correct source")
		}

		res, err := addBot(bot)

		if err != nil {
			return err
		}

		log.Info("Added bot: ", res.InsertedID)

	} else if mongoBot.Err() != nil {
		log.Error(mongoBot.Err())
		return mongoBot.Err()
	} else {
		var mongoType struct {
			Type string `bson:"type"`
		}

		mongoBot.Decode(&mongoType)

		if mongoType.Type != "pending" {
			return errors.New("bot 'type' is not pending")
		}
	}

	res, err := col.UpdateOne(ctx, bson.M{"botID": bot.BotID}, bson.M{"$set": bson.M{"type": "approved"}})

	if err != nil {
		return err
	}

	log.Info("Updated ", res.MatchedCount, " bots")

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
	col := mongoDb.Collection("bots")

	mongoBot := col.FindOne(ctx, bson.M{"botID": bot.BotID})

	if mongoBot.Err() == mongo.ErrNoDocuments {
		if !bot.CrossAdd && bot.ListSource != os.Getenv("LIST_ID") {
			return errors.New("bot is not from the correct source")
		}

		res, err := addBot(bot)

		if err != nil {
			return err
		}

		log.Info("Added bot: ", res.InsertedID)

	} else if mongoBot.Err() != nil {
		log.Error(mongoBot.Err())
		return mongoBot.Err()
	} else {
		var mongoType struct {
			Type string `bson:"type"`
		}

		mongoBot.Decode(&mongoType)

		if mongoType.Type != "pending" {
			return errors.New("bot 'type' is not pending")
		}
	}

	res, err := col.UpdateOne(ctx, bson.M{"botID": bot.BotID}, bson.M{"$set": bson.M{"type": "denied"}})

	if err != nil {
		return err
	}

	log.Info("Updated ", res.MatchedCount, " bots")

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
