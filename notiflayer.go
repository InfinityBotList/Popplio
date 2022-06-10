package main

import (
	"io/ioutil"
	"os"
	"popplio/types"

	webpush "github.com/SherClockHolmes/webpush-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

var notifChannel = make(chan types.Notification)

// This channel is used to send a notification to the client
func init() {
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
}
