package main

import (
	"io/ioutil"
	"os"
	"popplio/types"
	"popplio/utils"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/exp/slices"
)

var notifChannel = make(chan types.Notification)
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

	/* This channel is used to send a notification to the client

	   In order to so, we create a always running goroutine responsible for fanning out notifications

	   Vote reminders is a seperate goroutine
	*/
	go func() {
		for msg := range notifChannel {
			col := mongoDb.Collection("poppypaw")

			// Push out notification
			var notifInfo struct {
				Auth     string `bson:"auth"`
				Endpoint string `bson:"endpoint"`
				P256dh   string `bson:"p256dh"`
			}

			err := col.FindOne(ctx, bson.M{"notifId": msg.NotifID}).Decode(&notifInfo)

			if err != nil {
				log.Error("Error finding notification: %s", err)
				continue
			}

			sub := webpush.Subscription{
				Endpoint: notifInfo.Endpoint,
				Keys: webpush.Keys{
					Auth:   notifInfo.Auth,
					P256dh: notifInfo.P256dh,
				},
			}

			// Send Notification
			resp, err := webpush.SendNotification(msg.Message, &sub, &webpush.Options{
				Subscriber:      "votereminders@infinitybots.gg",
				VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
				VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
				TTL:             30,
			})
			if err != nil {
				// TODO: Handle error
				log.Error(err)
				continue
			}

			msg, _ := ioutil.ReadAll(resp.Body)
			log.Info(resp.StatusCode, msg)
			resp.Body.Close()
		}
	}()

	go func() {
		d := 5 * time.Second
		for x := range time.Tick(d) {
			log.Info("Tick at ", x, ", checking reminders")

			col := mongoDb.Collection("silverpelt")

			// Get all reminders

			cur, err := col.Find(ctx, bson.M{})

			if err != nil {
				log.Error("Error finding reminders:ß", err)
				continue
			}

			for cur.Next(ctx) {
				var reminder types.Reminder
				err := cur.Decode(&reminder)

				if err != nil {
					log.Error("Error decoding reminder:", err)
					continue
				}

				// Check for duplicates
				count, err := col.CountDocuments(ctx, bson.M{"userID": reminder.UserID, "botID": reminder.BotID})

				if err != nil {
					log.Error("Error counting reminders:", err)
				} else {
					if count > 1 {
						log.Warning("Reminder has duplicates, deleting one of them")
						_, err := col.DeleteOne(ctx, bson.M{"userID": reminder.UserID, "botID": reminder.BotID})

						if err != nil {
							log.Error("Error deleting reminder:", err)
						}
					}
				}

				// Check if reminder is acked
				if time.Now().Unix()-reminder.LastAcked < 4*60*60 {
					log.WithFields(log.Fields{
						"userId":    reminder.UserID,
						"botId":     reminder.BotID,
						"lastAcked": reminder.LastAcked,
					}).Warning("Reminder has been acked, skipping")
					continue
				}

				voteParsed, err := utils.GetVoteData(ctx, mongoDb, reminder.UserID, reminder.BotID)

				if err != nil {
					log.Error(err)
					continue
				}

				if !voteParsed.HasVoted {
					res, err := mongoDb.Collection("silverpelt").UpdateMany(ctx, bson.M{"userID": reminder.UserID, "botID": reminder.BotID}, bson.M{"$set": bson.M{"lastAcked": time.Now().Unix()}})

					if err != nil {
						log.Error("Error updating reminder: %s", err)
						continue
					}

					log.Info("Updated reminder: ", res.ModifiedCount)

					// Loop over all user poppypaw subscriptions and push to goro
					go func(id string, bId string) {
						col := mongoDb.Collection("poppypaw")

						cur, err := col.Find(ctx, bson.M{"id": id})

						if err != nil {
							log.Error("Error finding subscriptions: %s", err)
							return
						}

						botInf, err := utils.GetDiscordUser(metro, redisCache, ctx, bId)

						if err != nil {
							log.Error("Error finding bot info: %s", err)
							return
						}

						message := types.Message{
							Message: "You can vote for " + botInf.Username + " now!",
							Title:   "Vote for " + botInf.Username + "!",
							Icon:    botInf.Avatar,
						}

						bytes, err := json.Marshal(message)

						if err != nil {
							log.Error(err)
							return
						}

						defer cur.Close(ctx)

						doneIds := []string{}
						doneNotifs := []string{}

						for cur.Next(ctx) {
							var sub struct {
								NotifID  string `bson:"notifId"`
								Endpoint string `bson:"endpoint"`
							}

							err := cur.Decode(&sub)

							if err != nil {
								log.Error(err)
								continue
							}

							log.Info("NotifID: ", sub.NotifID)

							if sub.NotifID == "" {
								continue
							}

							if slices.Contains(doneIds, sub.Endpoint) || slices.Contains(doneNotifs, sub.NotifID) {
								log.Info("Already sent notification to: ", sub.Endpoint)
								continue
							}

							doneIds = append(doneIds, sub.Endpoint)
							doneNotifs = append(doneNotifs, sub.NotifID)

							notifChannel <- types.Notification{
								NotifID: sub.NotifID,
								Message: bytes,
							}
						}
					}(reminder.UserID, reminder.BotID)
				}
			}
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
