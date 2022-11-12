package notifications

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/bwmarrin/discordgo"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgtype"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

var (
	NotifChannel         = make(chan types.Notification)
	PremiumChannel       = make(chan string)
	MessageNotifyChannel = make(chan types.DiscordLog)

	silverpeltColsArr = utils.GetCols(types.Reminder{})
	silverpeltCols = strings.Join(silverpeltColsArr, ",")
)

func init() {
	/* This channel is used to fan out premium removals */
	go func() {
		for id := range PremiumChannel {
			log.WithFields(log.Fields{
				"id": id,
			}).Warning("Removing premium bot: ", id)

			_, err := state.Pool.Exec(state.Context, "UPDATE bots SET premium = false, start_premium_period = $1, premium_period_length = $2 WHERE bot_id = $3", time.Now().UnixMilli(), 2592000000, id)

			if err != nil {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error setting premium: ", err)
				continue
			}

			// Send message
			botObj, err := utils.GetDiscordUser(id)

			if err != nil {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error getting bot object: ", err)
				continue
			}

			var owner pgtype.Text

			err = state.Pool.QueryRow(state.Context, "SELECT owner FROM bots WHERE bot_id = $1", id).Scan(&owner)

			if err != nil || owner.Status != pgtype.Present {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error getting bot ownership info: ", err)
				continue
			}

			userObj, err := utils.GetDiscordUser(owner.String)

			if err != nil {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error getting main owner info: ", err)
				continue
			}

			state.Discord.ChannelMessageSendComplex(os.Getenv("CHANNEL_ID"), &discordgo.MessageSend{
				Content: botObj.Mention + "(" + botObj.Username + ") by " + userObj.Mention + " has been removed from the premium list as their subscription has expired.",
			})

			dmChannel, err := state.Discord.UserChannelCreate(owner.String)

			if err != nil {
				log.WithFields(log.Fields{
					"id": id,
				}).Error("Error creating DM channel: ", err)
				continue
			}

			state.Discord.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
				Content: "Your bot " + botObj.Mention + "(" + botObj.Username + ") has been removed from the premium list as their subscription has expired.",
			})
		}
	}()

	/* This channel is used to send a notification to the client

	   In order to so, we create a always running goroutine responsible for fanning out notifications

	   Vote reminders is a seperate goroutine
	*/

	go func() {
		for msg := range NotifChannel {
			var auth string
			var endpoint string
			var p256dh string

			err := state.Pool.QueryRow(state.Context, "SELECT auth, endpoint, p256dh FROM poppypaw WHERE notif_id = $1", msg.NotifID).Scan(&auth, &endpoint, &p256dh)

			if err != nil {
				log.Error("Error finding notification: %s", err)
				continue
			}

			sub := webpush.Subscription{
				Endpoint: endpoint,
				Keys: webpush.Keys{
					Auth:   auth,
					P256dh: p256dh,
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
				if resp.StatusCode == 410 {
					// Delete the subscription
					state.Pool.Exec(state.Context, "DELETE FROM poppypaw WHERE notif_id = $1", msg.NotifID)
				}
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
		timer := time.NewTimer(d)
		for x := range timer.C {
			if state.Migration {
				return
			}

			log.Info("Tick at ", x, ", checking reminders")

			rows, err := state.Pool.Query(state.Context, "SELECT "+silverpeltCols+" FROM silverpelt WHERE NOW() - last_acked > interval '4 hours'")

			if err != nil {
				log.Error("Error finding reminders: ", err)
				continue
			}

			for rows.Next() {
				var reminder types.Reminder
				err := pgxscan.ScanRow(&reminder, rows)

				if err != nil {
					log.Error("Error decoding reminder:", err)
					continue
				}

				// Check for duplicates
				var count int64

				err = state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM silverpelt WHERE bot_id = $1 AND user_id = $2", reminder.BotID, reminder.UserID).Scan(&count)

				if err != nil {
					log.Error("Error counting reminders:", err)
				} else {
					if count > 1 {
						log.Warning("Reminder has duplicates, deleting one of them")
						_, err = state.Pool.Exec(state.Context, "DELETE FROM silverpelt WHERE bot_id = $1 AND user_id = $2", reminder.BotID, reminder.UserID)

						if err != nil {
							log.Error("Error deleting reminder:", err)
						}

						_, err = state.Pool.Exec(state.Context, "INSERT INTO silverpelt (bot_id, user_id) VALUES ($1, $2)", reminder.BotID, reminder.UserID)

						if err != nil {
							log.Error("Error readding reminder:", err)
						}
					}
				}

				voteParsed, err := utils.GetVoteData(state.Context, reminder.UserID, reminder.BotID)

				if err != nil {
					log.Error(err)
					continue
				}

				if !voteParsed.HasVoted {
					res, err := state.Pool.Exec(state.Context, "UPDATE silverpelt SET last_acked = NOW() WHERE bot_id = $1 AND user_id = $2", reminder.BotID, reminder.UserID)
					if err != nil {
						log.Error("Error updating reminder: %s", err)
						continue
					}

					log.Info("Updated reminders: ", res.RowsAffected())

					// Loop over all user poppypaw subscriptions and push to goro
					go func(id string, bId string) {
						rows, err := state.Pool.Query(state.Context, "SELECT notif_id, endpoint FROM poppypaw WHERE id = $1", id)

						if err != nil {
							log.Error("Error finding subscriptions:", err)
							return
						}

						var notifs []struct {
							NotifId  string `db:"notif_id"`
							Endpoint string `db:"endpoint"`
						}

						err = pgxscan.ScanAll(&notifs, rows)

						if err != nil {
							log.Error("Error finding subscriptions:", err)
							return
						}

						botInf, err := utils.GetDiscordUser(bId)

						if err != nil {
							log.Error("Error finding bot info:", err)
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

						doneIds := []string{}
						doneNotifs := []string{}

						for _, notif := range notifs {
							log.Info("NotifID: ", notif.NotifId)

							if notif.NotifId == "" {
								continue
							}

							if slices.Contains(doneIds, notif.Endpoint) || slices.Contains(doneNotifs, notif.Endpoint) {
								log.Info("Already sent notification to: ", notif.Endpoint)
								continue
							}

							doneIds = append(doneIds, notif.Endpoint)
							doneNotifs = append(doneNotifs, notif.NotifId)

							NotifChannel <- types.Notification{
								NotifID: notif.NotifId,
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
		timer := time.NewTimer(d)
		for x := range timer.C {
			if state.Migration {
				return
			}

			log.Info("Tick at ", x, ", checking premiums")

			rows, err := state.Pool.Query(state.Context, "SELECT bot_id, start_premium_period, premium_period_length, type FROM bots WHERE premium = true")

			if err != nil {
				log.Error("Error finding bots: %s", err)
				continue
			}

			for rows.Next() {
				// Check bot
				var botId string
				var startPremiumPeriod int64
				var premiumPeriodLength int64
				var typeStr string

				err := rows.Scan(&botId, &startPremiumPeriod, &premiumPeriodLength, &typeStr)

				if err != nil {
					log.Error("Error decoding bot: %s", err)
					continue
				}

				if typeStr != "approved" {
					// This bot isnt approved, so we should remove premium
					log.Info("Removing premium from bot: ", botId)
					PremiumChannel <- botId
				}

				// Check start and sub period
				if int64(premiumPeriodLength)-(time.Now().UnixMilli()-int64(startPremiumPeriod)) < 0 {
					// This bot isnt premium, so we should remove premium
					log.Info("Removing premium from bot: ", botId)
					PremiumChannel <- botId
				}
			}

			rows.Close()
		}
	}()

	// Message sending notification goroutine
	go func() {
		for msg := range MessageNotifyChannel {
			if msg.Message == nil {
				continue
			}

			log.Info("Sending message to: ", msg.ChannelID)

			// Send message to channel
			_, err := state.Discord.ChannelMessageSendComplex(msg.ChannelID, msg.Message)

			if err != nil {
				log.Error("Error sending message: ", err)
				continue
			}
		}
	}()
}
