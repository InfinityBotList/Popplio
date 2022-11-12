package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"popplio/constants"
	"popplio/state"
	"popplio/utils"
	"strconv"
	"strings"
	"time"

	b64 "encoding/base64"
	"encoding/hex"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgtype"
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

	// Create HMAC of current time in seconds to protect against fucked up redirects
	h := hmac.New(sha512.New, []byte(os.Getenv("CLIENT_SECRET")))

	ctime := strconv.FormatInt(time.Now().Unix(), 10)

	var act = chi.URLParam(r, "act")

	h.Write([]byte(ctime + "@" + act))

	hmacData := hex.EncodeToString(h.Sum(nil))

	http.Redirect(w, r, "https://discord.com/api/oauth2/authorize?client_id="+cliId+"&scope=identify&response_type=code&redirect_uri="+redirectUrl+"&state="+ctime+"."+hmacData+"."+act, http.StatusFound)
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

	body, err := io.ReadAll(resp.Body)

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

	log.Info(token)

	if !strings.Contains(token.Scope, "identify") {
		http.Error(w, "Invalid scope: scope contain identify, is currently "+token.Scope, http.StatusBadRequest)
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

	err = state.Redis.Set(ctx, taskId, "WAITING", time.Hour*8).Err()

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

		_, err := state.Pool.Exec(ctx, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, user.ID)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte(token))
		return

	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(constants.NotFound))
		return
	}

	http.Redirect(w, r, "/cosmog/tasks/"+taskId+"?n="+b64.URLEncoding.EncodeToString(body), http.StatusFound)
}

func taskFn(w http.ResponseWriter, r *http.Request) {
	var user InternalOauthUser

	tid := chi.URLParam(r, "tid")

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
	tid := chi.URLParam(r, "tid")

	if tid == "" {
		http.Error(w, "No task id provided", http.StatusBadRequest)
		return
	}

	task, err := state.Redis.Get(ctx, tid).Result()

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
	state.Redis.SetArgs(ctx, taskId, "Fetching basic user data", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	var keys []*struct {
		ForeignTable      string `db:"foreign_table_name"`
		TableName         string `db:"table_name"`
		ColumnName        string `db:"column_name"`
		ForeignColumnName string `db:"foreign_column_name"`
	}

	data, err := state.Pool.Query(ctx, ddrStr)

	if err != nil {
		log.Error(err)

		state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})

		return
	}

	if err := pgxscan.ScanAll(&keys, data); err != nil {
		log.Error(err)

		state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})

		return
	}

	finalDump := make(map[string]any)

	for _, key := range keys {
		if key.ForeignTable == "users" {
			sqlStmt := "SELECT * FROM " + key.TableName + " WHERE " + key.ColumnName + "= $1"

			data, err := state.Pool.Query(ctx, sqlStmt, id)

			if err != nil {
				log.Error(err)
			}

			var rows []map[string]any

			if err := pgxscan.ScanAll(&rows, data); err != nil {
				log.Error(err)

				state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
					KeepTTL: true,
				})

				return
			}

			if del {
				sqlStmt = "DELETE FROM " + key.TableName + " WHERE " + key.ColumnName + "= $1"

				_, err := state.Pool.Exec(ctx, sqlStmt, id)

				if err != nil {
					log.Error(err)

					state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
						KeepTTL: true,
					})

					return
				}
			}

			finalDump[key.TableName] = rows
		}
	}

	state.Redis.SetArgs(ctx, taskId, "Fetching postgres backups on this user", redis.SetArgs{
		KeepTTL: true,
	})

	rows, err := state.BackupsPool.Query(ctx, "SELECT col, data, ts, id FROM backups")

	if err != nil {
		log.Error("Failed to get backups")
		state.Redis.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error(), redis.SetArgs{
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
			state.Redis.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error()+". Ignoring", redis.SetArgs{
				KeepTTL: true,
			})
			continue
		}

		var dataPacket []KVPair

		err = json.Unmarshal([]byte(data.Bytes), &dataPacket)

		if err != nil {
			log.Error("Failed to decode backup")
			state.Redis.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error()+". Ignoring", redis.SetArgs{
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
				_, err := state.BackupsPool.Exec(ctx, "DELETE FROM backups WHERE id=$1", toString(uid))
				if err != nil {
					log.Error("Failed to delete backup")
					state.Redis.SetArgs(ctx, taskId, "Failed to delete backup: "+err.Error(), redis.SetArgs{
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
		state.Redis.SetArgs(ctx, taskId, "Failed to encode data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	state.Redis.SetArgs(ctx, taskId, string(bytes), redis.SetArgs{
		KeepTTL: false,
	})
}
