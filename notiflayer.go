package main

import (
	"os"
	"popplio/types"
	"popplio/utils"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

var premiumChannel = make(chan string)
var messageNotifyChannel = make(chan types.DiscordLog)

func init() {
	/* This channel is used to fan out premium removals */
	go func() {
		for id := range premiumChannel {
			log.WithFields(log.Fields{
				"id": id,
			}).Warning("Removing premium bot: ", id)

			mongoDb.Collection("bots").UpdateOne(ctx, bson.M{"botID": id}, bson.M{"$set": bson.M{"premium": false, "start_period": time.Now().UnixMilli(), "sub_period": 2592000000}})

			// Send message
			botObj, err := utils.GetDiscordUser(metro, redisCache, ctx, id)

			if err != nil {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error getting bot object: ", err)
				continue
			}

			var botInf struct {
				MainOwner string `bson:"main_owner"`
			}

			err = mongoDb.Collection("bots").FindOne(ctx, bson.M{"botID": id}).Decode(&botInf)

			if err != nil {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error getting bot ownership info: ", err)
				continue
			}

			userObj, err := utils.GetDiscordUser(metro, redisCache, ctx, botInf.MainOwner)

			if err != nil {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error getting main owner info: ", err)
				continue
			}

			metro.ChannelMessageSendComplex(os.Getenv("CHANNEL_ID"), &discordgo.MessageSend{
				Content: botObj.Mention + "(" + botObj.Username + ") by " + userObj.Mention + " has been removed from the premium list as their subscription has expired.",
			})

			dmChannel, err := metro.UserChannelCreate(botInf.MainOwner)

			if err != nil {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error creating DM channel: ", err)
				continue
			}

			metro.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
				Content: "Your bot " + botObj.Mention + "(" + botObj.Username + ") has been removed from the premium list as their subscription has expired.",
			})
		}
	}()

	// Premium check goroutine
	go func() {
		d := 10 * time.Second
		for x := range time.Tick(d) {
			log.Info("Tick at ", x, ", checking premiums")
			cur, err := mongoDb.Collection("bots").Find(ctx, bson.M{"premium": true})

			if err != nil {
				log.Error("Error finding bots: %s", err)
				continue
			}

			for cur.Next(ctx) {
				// Check bot
				var bot struct {
					ID          string `bson:"botID"`
					StartPeriod int    `bson:"start_period,omitempty"`
					SubPeriod   int    `bson:"sub_period,omitempty"`
					Type        string `bson:"type,omitempty"`
				}

				err := cur.Decode(&bot)

				if err != nil {
					log.Error("Error decoding bot: %s", err)
					continue
				}

				if bot.Type != "approved" {
					// This bot isnt approved, so we should remove premium
					log.Info("Removing premium from bot: ", bot.ID)
					premiumChannel <- bot.ID
				}

				// Check start and sub period
				if int64(bot.SubPeriod)-(time.Now().UnixMilli()-int64(bot.StartPeriod)) < 0 {
					// This bot isnt premium, so we should remove premium
					log.Info("Removing premium from bot: ", bot.ID)
					premiumChannel <- bot.ID
				}
			}

			cur.Close(ctx)
		}
	}()

	// Message sending notification goroutine
	go func() {
		for msg := range messageNotifyChannel {
			if msg.WebhookID != "" && msg.WebhookToken != "" && msg.WebhookData != nil {
				log.Info("Sending message to webhook: ", msg.WebhookID)
				_, err := metro.WebhookExecute(msg.WebhookID, msg.WebhookToken, false, msg.WebhookData)

				if err != nil {
					log.Error("Error sending message to webhook: ", err)
					continue
				}
			}

			if msg.Message == nil {
				continue
			}

			log.Info("Sending message to: ", msg.ChannelID)

			// Send message to channel
			_, err := metro.ChannelMessageSendComplex(msg.ChannelID, msg.Message)

			if err != nil {
				log.Error("Error sending message: ", err)
				continue
			}
		}
	}()
}
