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
	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
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

const ddrStr = `
SELECT 
  tbl.relname AS table_name,
  col.attname AS column_name,
  referenced_tbl.relname AS foreign_table_name,
  referenced_field.attname AS foreign_column_name
FROM pg_constraint c
    INNER JOIN pg_namespace AS sh ON sh.oid = c.connamespace
    INNER JOIN (SELECT oid, unnest(conkey) as conkey FROM pg_constraint) con ON c.oid = con.oid
    INNER JOIN pg_class tbl ON tbl.oid = c.conrelid
    INNER JOIN pg_attribute col ON (col.attrelid = tbl.oid AND col.attnum = con.conkey)
    INNER JOIN pg_class referenced_tbl ON c.confrelid = referenced_tbl.oid
    INNER JOIN pg_namespace AS referenced_sh ON referenced_sh.oid = referenced_tbl.relnamespace
    INNER JOIN (SELECT oid, unnest(confkey) as confkey FROM pg_constraint) conf ON c.oid = conf.oid
    INNER JOIN pg_attribute referenced_field ON (referenced_field.attrelid = c.confrelid AND referenced_field.attnum = conf.confkey)
WHERE c.contype = 'f'`

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

		_, err := pool.Exec(ctx, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, user.ID)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte(token))
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

	var keys []*struct {
		ForeignTable      string `db:"foreign_table_name"`
		TableName         string `db:"table_name"`
		ColumnName        string `db:"column_name"`
		ForeignColumnName string `db:"foreign_column_name"`
	}

	data, err := pool.Query(ctx, ddrStr)

	if err != nil {
		log.Error(err)

		redisCache.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})

		return
	}

	if err := pgxscan.ScanAll(&keys, data); err != nil {
		log.Error(err)

		redisCache.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
	}

	finalDump := make(map[string]any)

	for _, key := range keys {
		if key.ForeignTable == "users" {
			sqlStmt := "SELECT * FROM " + key.TableName + " WHERE " + key.ColumnName + "= $1"

			data, err := pool.Query(ctx, sqlStmt, id)

			if err != nil {
				log.Error(err)
			}

			var rows []map[string]any

			if err := pgxscan.ScanAll(&rows, data); err != nil {
				log.Error(err)

				redisCache.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
					KeepTTL: true,
				})

				return
			}

			if del {
				sqlStmt = "DELETE FROM " + key.TableName + " WHERE " + key.ColumnName + "= $1"

				_, err := pool.Exec(ctx, sqlStmt, id)

				if err != nil {
					log.Error(err)

					redisCache.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
						KeepTTL: true,
					})

					return
				}
			}

			finalDump[key.TableName] = rows
		}
	}

	redisCache.SetArgs(ctx, taskId, "Fetching postgres backups on this user", redis.SetArgs{
		KeepTTL: true,
	})

	rows, err := backupPool.Query(ctx, "SELECT col, data, ts, id FROM backups")

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
				_, err := backupPool.Exec(ctx, "DELETE FROM backups WHERE id=$1", toString(uid))
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

	finalDump["backups"] = backups

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

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}

	return false
}

// Sends a webhook
func sendWebhook(webhook types.WebhookPost) error {
	url, token := webhook.URL, webhook.Token

	isDiscordIntegration := isDiscord(url)

	if !webhook.Test && (utils.IsNone(url) || utils.IsNone(token)) {
		// Fetch URL from postgres

		var bot struct {
			Discord    pgtype.Text `db:"webhook"`
			CustomURL  pgtype.Text `db:"custom_webhook"`
			CustomAuth pgtype.Text `db:"web_auth"`
			APIToken   pgtype.Text `db:"token"`
			HMACAuth   pgtype.Bool `db:"hmac"`
		}

		err := pgxscan.Get(ctx, pool, &bot, "SELECT webhook, custom_webhook, web_auth, token, hmac FROM bots WHERE bot_id = $1", webhook.BotID)

		if err != nil {
			log.Error("Failed to fetch webhook: ", err.Error())
			return err
		}

		// Check custom auth viability
		if bot.CustomAuth.Status != pgtype.Present || utils.IsNone(bot.CustomAuth.String) {
			if bot.APIToken.String != "" {
				token = bot.APIToken.String
			} else {
				// We set the token to the a random string in DB in this case
				token = utils.RandString(256)

				_, err := pool.Exec(ctx, "UPDATE bots SET web_auth = $1 WHERE bot_id = $2", token, webhook.BotID)

				if err != pgx.ErrNoRows && err != nil {
					log.Error("Failed to update webhook: ", err.Error())
					return err
				}
			}

			bot.CustomAuth = pgtype.Text{String: token, Status: pgtype.Present}
		}

		webhook.HMACAuth = bot.HMACAuth.Bool
		webhook.Token = bot.CustomAuth.String

		log.Info("Using hmac: ", webhook.HMACAuth)

		// For each url, make a new sendWebhook
		if !utils.IsNone(bot.CustomURL.String) {
			webhook.URL = bot.CustomURL.String
			err := sendWebhook(webhook)
			log.Error("Custom URL send error", err)
		}

		if !utils.IsNone(bot.Discord.String) {
			webhook.URL = bot.Discord.String
			err := sendWebhook(webhook)
			log.Error("Discord send error", err)
		}
	}

	if utils.IsNone(url) {
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
