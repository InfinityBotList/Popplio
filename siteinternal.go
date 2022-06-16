package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"popplio/types"
	"popplio/utils"
	"strconv"
	"strings"
	"time"

	b64 "encoding/base64"
	"encoding/hex"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgtype"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/exp/slices"
)

type InternalOauthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Disc     string `json:"discriminator"`
	TID      string `json:"-"` // Only set in taskFn
}

type KVPair struct {
	Key   string
	Value any
}

type InternalPassport struct {
	User *InternalOauthUser `json:"user"`
}

type InternalSession struct {
	Passport *InternalPassport `json:"passport"`
}

type InternalBot struct {
	ObjID   string `bson:"_id"`
	BotID   string `bson:"botID"`
	BotName string `bson:"botName"`
	Votes   int    `bson:"votes"`
	Avatar  string `bson:"-"`
}

type VoteTemplate struct {
	User InternalOauthUser
	Bot  InternalBot
}

func oauthFn(w http.ResponseWriter, r *http.Request) {
	cliId := os.Getenv("CLIENT_ID")
	redirectUrl := os.Getenv("REDIRECT_URL")
	vars := mux.Vars(r)

	// Create HMAC of current time in seconds to protect against fucked up redirects
	h := hmac.New(sha512.New, []byte(os.Getenv("CLIENT_SECRET")))

	ctime := strconv.FormatInt(time.Now().Unix(), 10)

	h.Write([]byte(ctime + "@" + vars["act"]))

	hmacData := hex.EncodeToString(h.Sum(nil))

	http.Redirect(w, r, "https://discord.com/api/oauth2/authorize?client_id="+cliId+"&scope=identify&response_type=code&redirect_uri="+redirectUrl+"&state="+ctime+"."+hmacData+"."+vars["act"], http.StatusFound)
}

func performAct(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("state")

	// Split act and hmac
	actSplit := strings.Split(act, ".")

	if len(actSplit) != 3 {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Check hmac
	h := hmac.New(sha512.New, []byte(os.Getenv("CLIENT_SECRET")))

	h.Write([]byte(actSplit[0] + "@" + actSplit[2]))

	hmacData := hex.EncodeToString(h.Sum(nil))

	if hmacData != actSplit[1] {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Check time
	ctime, err := strconv.ParseInt(actSplit[0], 10, 64)

	if err != nil {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	if time.Now().Unix()-ctime > 300 {
		http.Error(w, "Invalid state. HMAC too old", http.StatusBadRequest)
		return
	}

	// Remove out the actual action
	act = actSplit[2]

	// Check code with discords api
	data := url.Values{}

	data.Set("client_id", os.Getenv("CLIENT_ID"))
	data.Set("client_secret", os.Getenv("CLIENT_SECRET"))
	data.Set("grant_type", "authorization_code")
	data.Set("code", r.URL.Query().Get("code"))
	data.Set("redirect_uri", os.Getenv("REDIRECT_URL"))

	resp, err := http.PostForm("https://discord.com/api/oauth2/token", data)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var token struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}

	err = json.Unmarshal(body, &token)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if token.Scope != "identify" {
		http.Error(w, "Invalid scope: scope must be set to ONLY identify", http.StatusBadRequest)
		return
	}

	// Get user info
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: time.Second * 10}

	resp, err = client.Do(req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var user InternalOauthUser

	err = json.Unmarshal(body, &user)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	taskId := utils.RandString(196)

	err = redisCache.Set(ctx, taskId, "WAITING", time.Hour*8).Err()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

	if act == "dr" {
		go dataRequestTask(taskId, user.ID, remoteIp[0], false)
	} else if act == "ddr" {
		go dataRequestTask(taskId, user.ID, remoteIp[0], true)
	} else if act == "gettoken" {
		token := utils.RandString(128)

		_, err := mongoDb.Collection("users").UpdateOne(ctx, bson.M{"userID": user.ID}, bson.M{"$set": bson.M{"apiToken": token}})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte(token))
		return

	} else if strings.HasPrefix(act, "vote-") {
		voteBot := strings.Replace(act, "vote-", "", 1)

		// Find bot id from vote bot using either bot id or vanity

		var bot InternalBot

		err = mongoDb.Collection("bots").FindOne(ctx, bson.M{
			"$or": []bson.M{
				{
					"botName": voteBot,
				},
				{
					"vanity": voteBot,
				},
				{
					"botID": voteBot,
				},
			},
		}).Decode(&bot)

		if err != nil {
			log.Error(err)
			http.Error(w, "This bot could not be found", http.StatusNotFound)
			return
		}

		// Get bot avatar
		m, err := utils.GetDiscordUser(metro, redisCache, ctx, bot.BotID)

		if err != nil {
			log.Error(err)
			http.Error(w, "We couldn't fetch this bot from discord for some reason", http.StatusNotFound)
			return
		}

		bot.Avatar = m.Avatar

		t, err := template.ParseFiles("html/vote.html")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		t.Execute(w, VoteTemplate{
			Bot:  bot,
			User: user,
		})
		return
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFound))
		return
	}

	http.Redirect(w, r, "/cosmog/tasks/"+taskId+"?n="+b64.URLEncoding.EncodeToString(body), http.StatusFound)
}

func taskFn(w http.ResponseWriter, r *http.Request) {
	var user InternalOauthUser

	tid := mux.Vars(r)["tid"]

	if tid == "" {
		http.Error(w, "Invalid task id", http.StatusBadRequest)
		return
	}

	userStr := r.URL.Query().Get("n")

	if userStr == "" {
		user = InternalOauthUser{
			ID:       "Unknown",
			Username: "Unknown",
			Disc:     "0000",
		}
	} else {
		body, err := b64.URLEncoding.DecodeString(userStr)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = json.Unmarshal(body, &user)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	user.TID = tid

	t, err := template.ParseFiles("html/taskpage.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.Execute(w, user)
}

func getTask(w http.ResponseWriter, r *http.Request) {
	tid := mux.Vars(r)["tid"]

	if tid == "" {
		http.Error(w, "No task id provided", http.StatusBadRequest)
		return
	}

	task, err := redisCache.Get(ctx, tid).Result()

	if err == redis.Nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(task))
}

func toString(myUUID pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", myUUID.Bytes[0:4], myUUID.Bytes[4:6], myUUID.Bytes[6:8], myUUID.Bytes[8:10], myUUID.Bytes[10:16])
}

func dataRequestTask(taskId string, id string, ip string, del bool) {
	redisCache.SetArgs(ctx, taskId, "Fetching basic user data", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	// Get user info from mongo
	col := mongoDb.Collection("users")

	var finalDump struct {
		UserInfo     map[string]any   `json:"user_info"`
		Votes        []map[string]any `json:"votes"`
		Reviews      []map[string]any `json:"reviews"`
		Bots         []map[string]any `json:"bots"`
		Sessions     []any            `json:"sessions"`
		UniqueClicks []string         `json:"unique_clicks"`
		Backups      []any            `json:"backups"`
		Poppypaw     []map[string]any `json:"poppypaw"`
	}

	var userInfo map[string]any

	err := col.FindOne(ctx, bson.M{"userID": id}).Decode(&userInfo)

	if err != nil {
		log.Error("Failed to get user info")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch user data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	if del {
		_, err := col.DeleteOne(ctx, bson.M{"userID": id})
		if err != nil {
			log.Error("Failed to delete user")
			redisCache.SetArgs(ctx, taskId, "Failed to delete user: "+err.Error(), redis.SetArgs{
				KeepTTL: true,
			})
			return
		}
	}

	finalDump.UserInfo = userInfo

	// Get all votes with this user
	redisCache.SetArgs(ctx, taskId, "Fetching vote data on this user", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	col = mongoDb.Collection("votes")

	var votes []map[string]any

	cur, err := col.Find(ctx, bson.M{"userID": id})

	if err != nil {
		log.Error("Failed to get votes")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch vote data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	err = cur.All(ctx, &votes)

	if err != nil {
		log.Error("Failed to decode vote")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch vote data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	if del {
		_, err := col.DeleteMany(ctx, bson.M{"userID": id})
		if err != nil {
			log.Error("Failed to delete vote")
			redisCache.SetArgs(ctx, taskId, "Failed to delete vote: "+err.Error(), redis.SetArgs{
				KeepTTL: true,
			})
			return
		}
	}

	finalDump.Votes = votes

	// Poppypaw (Vote reminders)
	redisCache.SetArgs(ctx, taskId, "Fetching poppypaw data on this user", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	col = mongoDb.Collection("poppypaw")

	var poppypaw []map[string]any

	cur, err = col.Find(ctx, bson.M{"userID": id})

	if err != nil {
		log.Error("Failed to get votes")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch poppypaw data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	err = cur.All(ctx, &poppypaw)

	if err != nil {
		log.Error("Failed to decode vote")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch poppypaw data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	if del {
		_, err := col.DeleteMany(ctx, bson.M{"userID": id})
		if err != nil {
			log.Error("Failed to delete poppypaw")
			redisCache.SetArgs(ctx, taskId, "Failed to delete poppypaw: "+err.Error(), redis.SetArgs{
				KeepTTL: true,
			})
			return
		}
	}

	finalDump.Poppypaw = poppypaw

	// Reviews
	redisCache.SetArgs(ctx, taskId, "Fetching review data on this user", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	col = mongoDb.Collection("reviews")

	var reviews []map[string]any

	cur, err = col.Find(ctx, bson.M{"author": id})

	if err != nil {
		log.Error("Failed to get review")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch review data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	err = cur.All(ctx, &reviews)

	if err != nil {
		log.Error("Failed to decode review")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch review data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	if del {
		_, err := col.DeleteMany(ctx, bson.M{"author": id})
		if err != nil {
			log.Error("Failed to delete review")
			redisCache.SetArgs(ctx, taskId, "Failed to delete review: "+err.Error(), redis.SetArgs{
				KeepTTL: true,
			})
			return
		}
	}

	finalDump.Reviews = reviews

	redisCache.SetArgs(ctx, taskId, "Fetching bot data on this user", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	col = mongoDb.Collection("bots")

	var bots []map[string]any

	cur, err = col.Find(ctx, bson.M{})

	if err != nil {
		log.Error("Failed to get bots")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch bot data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	defer cur.Close(ctx)

	ucs := []string{}

	for cur.Next(ctx) {
		var bot map[string]any

		err = cur.Decode(&bot)

		if err != nil {
			log.Error("Failed to decode bot")
			redisCache.SetArgs(ctx, taskId, "Failed to fetch bot data: "+err.Error(), redis.SetArgs{
				KeepTTL: true,
			})
			return
		}

		if unique_clicks, ok := bot["unique_clicks"]; ok {
			ucsWithoutIp := []string{}
			if uc, ok := unique_clicks.(primitive.A); ok {
				for _, click := range uc {
					ucStr, ok := click.(string)

					if !ok {
						log.Error("Failed to convert click to string")
						continue
					}

					ipList := strings.Split(strings.ReplaceAll(ucStr, " ", ""), ",")

					if ipList[0] == ip {
						botID, ok := bot["botID"].(string)

						if !ok {
							continue
						}

						ucs = append(ucs, botID)
					} else {
						ucsWithoutIp = append(ucsWithoutIp, ucStr)
					}
				}
			}

			if del && len(ucsWithoutIp) > 0 {
				_, err = col.UpdateOne(ctx, bson.M{"botID": bot["botID"]}, bson.M{"$set": bson.M{"unique_clicks": ucsWithoutIp}})
				if err != nil {
					log.Error("Failed to update bot clicks")
					redisCache.SetArgs(ctx, taskId, "Failed to delete bot unique clicks: "+err.Error(), redis.SetArgs{
						KeepTTL: true,
					})
					return
				}
			}

		}

		if addOwners, ok := bot["additional_owners"]; ok {
			if addOwnersSlice, ok := addOwners.(primitive.A); ok {
				for _, owner := range addOwnersSlice {
					if ownerStr, ok := owner.(string); ok {
						if ownerStr == id {
							delete(bot, "unique_clicks")
							bots = append(bots, bot)
						}
					}
				}
			}
		}

		if owner, ok := bot["main_owner"]; ok {
			if ownerStr, ok := owner.(string); ok {
				if ownerStr == id {
					delete(bot, "unique_clicks")
					bots = append(bots, bot)

					if del {
						if del {
							_, err := col.DeleteOne(ctx, bson.M{"botID": bot["botID"]})
							if err != nil {
								log.Error("Failed to delete bot")
								redisCache.SetArgs(ctx, taskId, "Failed to delete bot: "+err.Error(), redis.SetArgs{
									KeepTTL: true,
								})
								return
							}
						}
					}
				}
			}
		}
	}

	finalDump.Bots = bots
	finalDump.UniqueClicks = ucs

	redisCache.SetArgs(ctx, taskId, "Fetching postgres backups on this user", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	rows, err := pool.Query(pgCtx, "SELECT col, data, ts, id FROM backups")

	if err != nil {
		log.Error("Failed to get backups")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	defer rows.Close()

	var backups []any

	var foundBackup bool

	for rows.Next() {
		var col pgtype.Text
		var data pgtype.JSONB
		var ts pgtype.Timestamptz
		var uid pgtype.UUID

		err = rows.Scan(&col, &data, &ts, &uid)

		if err != nil {
			log.Error("Failed to scan backup")
			redisCache.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error()+". Ignoring", redis.SetArgs{
				KeepTTL: true,
			})
			continue
		}

		var dataPacket []KVPair

		err = json.Unmarshal([]byte(data.Bytes), &dataPacket)

		if err != nil {
			log.Error("Failed to decode backup")
			redisCache.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error()+". Ignoring", redis.SetArgs{
				KeepTTL: true,
			})
			continue
		}

		var backupDat = make(map[string]any)

		for _, kvpair := range dataPacket {
			if kvpair.Key == "userID" || kvpair.Key == "author" || kvpair.Key == "main_owner" {
				val, ok := kvpair.Value.(string)
				if !ok {
					continue
				}

				if val == id {
					foundBackup = true
					break
				}
			}
		}

		if foundBackup {
			backupDat["col"] = col.String
			backupDat["data"] = dataPacket
			backupDat["ts"] = ts.Time
			backupDat["id"] = toString(uid)
			backups = append(backups, backupDat)

			if del {
				_, err := pool.Exec(pgCtx, "DELETE FROM backups WHERE id=$1", toString(uid))
				if err != nil {
					log.Error("Failed to delete backup")
					redisCache.SetArgs(ctx, taskId, "Failed to delete backup: "+err.Error(), redis.SetArgs{
						KeepTTL: true,
					})
					return
				}
			}
		}

		foundBackup = false
	}

	finalDump.Backups = backups

	// Handle sessions
	redisCache.SetArgs(ctx, taskId, "Fetching sessions of this user", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	col = mongoDb.Collection("sessions")

	cur, err = col.Find(ctx, bson.M{})

	if err != nil {
		log.Error("Failed to get sessions")
		redisCache.SetArgs(ctx, taskId, "Failed to fetch session data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	defer cur.Close(ctx)

	var sessions []any

	// May need to be rewritten
	for cur.Next(ctx) {

		var sessionD InternalSession

		var sessionMap map[string]any

		var session string

		err = cur.Decode(&sessionMap)

		if err != nil {
			log.Error("Failed to decode session")
			redisCache.SetArgs(ctx, taskId, "Failed to fetch session data", redis.SetArgs{
				KeepTTL: true,
			})
			return
		}

		session, ok := sessionMap["session"].(string)

		if !ok {
			log.Error("Failed to convert session to string")
			redisCache.SetArgs(ctx, taskId, "Failed to fetch session data: could not convert to string. Ignoring", redis.SetArgs{
				KeepTTL: true,
			})
			continue
		}

		err = json.Unmarshal([]byte(session), &sessionD)

		if err != nil {
			log.Error("Failed to decode session")
			redisCache.SetArgs(ctx, taskId, "Failed to fetch a session or two: Ignoring as it is likely bad data", redis.SetArgs{
				KeepTTL: true,
			})
			continue
		}

		if sessionD.Passport == nil || sessionD.Passport.User == nil {
			continue
		}

		if sessionD.Passport.User.ID == id {
			sessions = append(sessions, sessionMap)

			if del {
				_, err := col.DeleteOne(ctx, bson.M{"_id": sessionMap["_id"]})
				if err != nil {
					log.Error("Failed to delete session")
					redisCache.SetArgs(ctx, taskId, "Failed to delete session: "+err.Error(), redis.SetArgs{
						KeepTTL: true,
					})
					return
				}
			}
		}
	}

	finalDump.Sessions = sessions

	bytes, err := json.Marshal(finalDump)

	if err != nil {
		log.Error("Failed to encode data")
		redisCache.SetArgs(ctx, taskId, "Failed to encode data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	redisCache.SetArgs(ctx, taskId, string(bytes), redis.SetArgs{
		KeepTTL: false,
	})
}

func isDiscord(url string) bool {
	validPrefixes := []string{
		"https://discordapp.com/api/webhooks/",
		"https://discord.com/api/webhooks/",
		"https://canary.discord.com/api/webhooks/",
		"https://ptb.discord.com/api/webhooks/",
	}

	return slices.Contains(validPrefixes, url)
}

// Sends a webhook
func sendWebhook(webhook types.WebhookPost) error {
	url, token := webhook.URL, webhook.Token

	isDiscordIntegration := isDiscord(url)

	if !webhook.Test && (utils.IsNone(&url) || utils.IsNone(&token)) {
		// Fetch URL from mongoDB
		col := mongoDb.Collection("bots")

		var bot struct {
			Discord    string `bson:"webhook"`
			CustomURL  string `bson:"webURL"`
			CustomAuth string `bson:"webAuth"`
			APIToken   string `bson:"token"`
			HMACAuth   bool   `bson:"webHmacAuth,omitempty"`
		}

		err := col.FindOne(ctx, bson.M{"botID": webhook.BotID}).Decode(&bot)

		if err != nil {
			log.Error("Failed to fetch webhook")
			return err
		}

		// Check custom auth viability
		if utils.IsNone(&bot.CustomAuth) {
			// We set the token to the a random string in DB in this case
			token = utils.RandString(256)

			_, err := col.UpdateOne(ctx, bson.M{"botID": webhook.BotID}, bson.M{"$set": bson.M{"webAuth": token}})

			if err != mongo.ErrNoDocuments && err != nil {
				log.Error("Failed to update webhook: ", err.Error())
				return err
			}

			bot.CustomAuth = token
		}

		webhook.HMACAuth = bot.HMACAuth
		webhook.Token = bot.CustomAuth

		log.Info("Using hmac: ", webhook.HMACAuth)

		// For each url, make a new sendWebhook
		if !utils.IsNone(&bot.CustomURL) {
			webhook.URL = bot.CustomURL
			err := sendWebhook(webhook)
			log.Error("Custom URL send error", err)
		}

		if !utils.IsNone(&bot.Discord) {
			webhook.URL = bot.Discord
			err := sendWebhook(webhook)
			log.Error("Discord send error", err)
		}
	}

	if utils.IsNone(&url) {
		log.Warning("Refusing to continue as no webhook")
		return nil
	}

	if isDiscordIntegration && !isDiscord(url) {
		return errors.New("webhook is not a discord webhook")
	}

	if isDiscordIntegration {
		parts := strings.Split(url, "/")
		if len(parts) < 7 {
			log.WithFields(log.Fields{
				"url": url,
			}).Warning("Invalid webhook URL")
			return errors.New("invalid discord webhook URL. Could not parse")
		}

		webhookId := parts[5]
		webhookToken := parts[6]
		userObj, err := utils.GetDiscordUser(metro, redisCache, ctx, webhook.UserID)

		if err != nil {
			userObj = &types.DiscordUser{
				ID:            "510065483693817867",
				Username:      "Toxic Dev (test webhook)",
				Avatar:        "https://cdn.discordapp.com/avatars/510065483693817867/a_96c9cea3c656deac48f1d8fdfdae5007.gif?size=1024",
				Discriminator: "0000",
			}
		}

		log.WithFields(log.Fields{
			"user":      webhook.UserID,
			"webhookId": webhookId,
			"token":     webhookToken,
		}).Warning("Got here in parsing webhook for discord")

		botObj, err := utils.GetDiscordUser(metro, redisCache, ctx, webhook.BotID)
		if err != nil {
			log.WithFields(log.Fields{
				"user": webhook.BotID,
			}).Warning(err)
			return err
		}
		userWithDisc := userObj.Username + "#" + userObj.Discriminator // Create the user object

		var embeds []*discordgo.MessageEmbed = []*discordgo.MessageEmbed{
			{
				Title: "Congrats! " + botObj.Username + " got a new vote!!!",
				Description: "**" + userWithDisc + "** just voted for **" + botObj.Username + "**!\n\n" +
					"**" + botObj.Username + "** now has **" + strconv.Itoa(webhook.Votes) + "** votes!",
				Color: 0x00ff00,
				URL:   "https://botlist.site/bots/" + webhook.BotID,
			},
		}

		_, err = metro.WebhookExecute(webhookId, webhookToken, true, &discordgo.WebhookParams{
			Embeds:    embeds,
			Username:  userObj.Username,
			AvatarURL: userObj.Avatar,
		})

		if err != nil {
			log.WithFields(log.Fields{
				"webhook": webhookId,
			}).Warning("Failed to execute webhook", err)
			return err
		}
	} else {
		tries := 0

		for tries < 3 {
			// Create response body
			body := types.WebhookData{
				Votes:        webhook.Votes,
				UserID:       webhook.UserID,
				BotID:        webhook.BotID,
				UserIDLegacy: webhook.UserID,
				BotIDLegacy:  webhook.BotID,
				Test:         webhook.Test,
				Time:         time.Now().Unix(),
			}

			data, err := json.Marshal(body)

			if err != nil {
				log.Error("Failed to encode data")
				return err
			}

			if webhook.HMACAuth {
				// Generate HMAC token using token and request body
				h := hmac.New(sha512.New, []byte(token))
				h.Write(data)
				token = hex.EncodeToString(h.Sum(nil))
			}

			// Create request
			responseBody := bytes.NewBuffer(data)
			req, err := http.NewRequest("POST", url, responseBody)

			if err != nil {
				log.Error("Failed to create request")
				return err
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "Popplio/v5.0")
			req.Header.Set("Authorization", token)

			// Send request
			client := &http.Client{Timeout: time.Second * 5}
			resp, err := client.Do(req)

			if err != nil {
				log.Error("Failed to send request")
				return err
			}

			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				log.Info("Retrying webhook again. Got status code of ", resp.StatusCode)
				tries++
				continue
			}

			break
		}
	}

	return nil
}
