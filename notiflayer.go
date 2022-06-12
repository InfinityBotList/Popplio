package main

import (
	"io/ioutil"
	"os"
	"popplio/types"
	"popplio/utils"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
)

var notifChannel = make(chan types.Notification)

func init() {
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

				var votes []int64

				col = mongoDb.Collection("votes")

				findOptions := options.Find()

				findOptions.SetSort(bson.M{"date": -1})

				cur, err := col.Find(ctx, bson.M{"botID": reminder.BotID, "userID": reminder.UserID}, findOptions)

				if err == nil || err == mongo.ErrNoDocuments {
					for cur.Next(ctx) {
						var vote struct {
							Date int64 `bson:"date"`
						}

						err := cur.Decode(&vote)

						if err != nil {
							log.Error(err)
							continue
						}

						votes = append(votes, vote.Date)
					}

					cur.Close(ctx)

				} else {
					log.Error(err)
					continue
				}

				voteParsed := types.UserVote{
					VoteTime: utils.GetVoteTime(),
				}

				voteParsed.Timestamps = votes

				// In most cases, will be one but not always
				if len(votes) > 0 {
					if time.Now().UnixMilli() < votes[0] {
						log.Error("detected illegal vote time", votes[0])
						votes[0] = time.Now().UnixMilli()
					}

					if time.Now().UnixMilli()-votes[0] < int64(utils.GetVoteTime())*60*60*1000 {
						voteParsed.HasVoted = true
						voteParsed.LastVoteTime = votes[0]
					}
				}

				if voteParsed.LastVoteTime == 0 && len(votes) > 0 {
					voteParsed.LastVoteTime = votes[0]
				}

				if !voteParsed.HasVoted {
					res, err := mongoDb.Collection("silverpelt").UpdateMany(ctx, bson.M{"userID": reminder.UserID, "botID": reminder.BotID}, bson.M{"$set": bson.M{"lastAcked": time.Now().Unix()}})

					if err != nil {
						log.Error("Error updating reminder: %s", err)
						return
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
}
