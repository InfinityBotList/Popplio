package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"popplio/utils"
	"strings"
	"time"

	b64 "encoding/base64"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InternalOauthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Disc     string `json:"discriminator"`
	TID      string `json:"-"` // Only set in taskFn
}

func oauthFn(w http.ResponseWriter, r *http.Request) {
	cliId := os.Getenv("CLIENT_ID")
	redirectUrl := os.Getenv("REDIRECT_URL")

	vars := mux.Vars(r)

	http.Redirect(w, r, "https://discord.com/api/oauth2/authorize?client_id="+cliId+"&scope=identify&response_type=code&redirect_uri="+redirectUrl+"&state="+vars["act"], http.StatusFound)
}

func performAct(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("state")

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

	if act == "dr" {
		remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

		go dataRequestTask(taskId, user.ID, remoteIp[0])
	} else if act == "ddr" {
		//go dataDeleteTask(taskId, user.ID)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFound))
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

func dataRequestTask(taskId string, id string, ip string) {
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
		UniqueClicks []string         `json:"unique_clicks"`
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

	finalDump.Votes = votes

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

	finalDump.Reviews = reviews

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
					}
				}
			}
		}

		if addOwners, ok := bot["additional_owners"]; ok {
			if addOwnersSlice, ok := addOwners.([]string); ok {
				for _, owner := range addOwnersSlice {
					if owner == id {
						delete(bot, "unique_clicks")
						bots = append(bots, bot)
					}
				}
			}
		}

		if owner, ok := bot["main_owner"]; ok {
			if ownerStr, ok := owner.(string); ok {
				if ownerStr == id {
					delete(bot, "unique_clicks")
					bots = append(bots, bot)
				}
			}
		}
	}

	finalDump.Bots = bots
	finalDump.UniqueClicks = ucs

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

/*
func dataDeleteTask(taskId string, id string) {
	//
}
*/
