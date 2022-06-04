package main

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"popplio/types"
	"popplio/utils"

	integrase "github.com/MetroReviews/metro-integrase/lib"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoUrl   = "mongodb://127.0.0.1:27017/infinity" // Is already public in 10 other places so
	docsSite   = "https://docs.botlist.site"
	mainSite   = "https://infinitybotlist.com"
	statusPage = "https://status.botlist.site"
	apiBot     = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
)

var (
	redisCache *redis.Client
	mongoDb    *mongo.Database
	ctx        context.Context
)

func rateLimitWrap(reqs int, t time.Duration, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get ratelimit from redis
		var id string

		auth := r.Header.Get("Authorization")

		if auth != "" {
			// Check if the user is a bot
			botCol := mongoDb.Collection("bots")

			var bot struct {
				BotID string `bson:"botID"`
			}

			options := options.FindOne().SetProjection(bson.M{"botID": 1})

			err := botCol.FindOne(ctx, bson.M{"token": auth}, options).Decode(&bot)

			if err != nil {
				// Bot does not exist, return
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("{\"error\":\"Invalid API token\"}"))
				return
			}

			id = bot.BotID
		} else {
			remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

			// For user privacy, hash the remote ip
			hasher := sha512.New()
			hasher.Write([]byte(remoteIp[0]))
			id = fmt.Sprintf("%x", hasher.Sum(nil))
		}

		v := redisCache.Get(r.Context(), "rl:"+id).Val()

		if v == "" {
			v = "0"

			err := redisCache.Set(ctx, "rl:"+id, "0", t).Err()

			if err != nil {
				log.Error(err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("{\"error\":\"Something broke!\"}"))
				return
			}
		}

		err := redisCache.Incr(ctx, "rl:"+id).Err()

		if err != nil {
			log.Error(err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\":\"Something broke!\"}"))
			return
		}

		vInt, err := strconv.Atoi(v)

		if err != nil {
			log.Error(err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\":\"Something broke!\"}"))
			return
		}

		if vInt > reqs {
			w.Header().Set("Content-Type", "application/json")
			retryAfter := redisCache.TTL(ctx, "rl:"+id).Val()
			w.Header().Set("Retry-After", strconv.FormatFloat(retryAfter.Seconds(), 'g', -1, 64))

			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("{\"error\":\"You're being rate limited!\"}"))

			return
		}

		w.Header().Set("Ratelimit-Req-Made", strconv.Itoa(vInt))

		fn(w, r)
	}
}

func main() {
	r := mux.NewRouter()

	// Init redisCache
	redisCache = redis.NewClient(&redis.Options{})

	// Create base payloads before startup
	// Index
	helloWorldB := map[string]string{
		"message": "Hello world from IBL API v5!",
		"docs":    docsSite,
		"ourSite": mainSite,
		"status":  statusPage,
	}

	helloWorld, err := json.Marshal(helloWorldB)

	if err != nil {
		panic(err)
	}

	// Not Found
	notFoundB := map[string]string{
		"message": "Slow down, bucko! We couldn't find this resource *anywhere*!",
	}

	notFound, err := json.Marshal(notFoundB)

	if err != nil {
		panic(err)
	}

	// Bad request
	badRequestB := map[string]string{
		"message": "Slow down, bucko! You're doing something illegal!!!",
	}

	badRequest, err := json.Marshal(badRequestB)

	if err != nil {
		panic(err)
	}

	// Not approved
	notApprovedB := map[string]string{
		"message": "Woah there, your bot needs to be approved. Calling the police right now!",
	}

	notApproved, err := json.Marshal(notApprovedB)

	if err != nil {
		panic(err)
	}

	ctx = context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUrl))

	if err != nil {
		panic(err)
	}

	fmt.Println("Connected to mongoDB?")

	mongoDb = client.Database("infinity")

	colNames, err := mongoDb.ListCollectionNames(ctx, bson.D{})

	fmt.Println("Collections:", colNames)

	if err != nil {
		panic(err)
	}
	godotenv.Load()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(helloWorld))
	})

	statsFn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" || r.Method == "DELETE" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(badRequest))
			return
		}

		if r.Body == nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		// Check token
		col := mongoDb.Collection("bots")

		var bot types.Bot

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(badRequest))
			return
		} else {
			err := col.FindOne(ctx, bson.M{"token": r.Header.Get("Authorization")}).Decode(&bot)

			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(badRequest))
				return
			}
		}

		if bot.Type != "approved" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(notApproved))
			return
		}

		defer r.Body.Close()

		var payload types.BotStats

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte([]byte("{\"error\":\"Something broke!\"}")))
			return
		}

		err = json.Unmarshal(bodyBytes, &payload)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		servers, shards, users := payload.GetStats()

		_, err = col.UpdateOne(ctx, bson.M{"token": r.Header.Get("Authorization")}, bson.M{"$set": bson.M{"servers": servers, "shards": shards, "users": users}})

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte([]byte("{\"error\":\"Something broke!\"}")))
			return
		}

		w.Write([]byte("{\"error\":null}"))
	}

	r.HandleFunc("/bots/stats", rateLimitWrap(4, 1*time.Minute, statsFn))

	// Note that only token matters for this endpoint at this time
	// TODO: Handle bot id as well
	r.HandleFunc("/bots/{id}/stats", rateLimitWrap(4, 1*time.Minute, statsFn))

	r.HandleFunc("/bots/{id}", rateLimitWrap(4, 1*time.Minute, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(badRequest))
			return
		}

		vars := mux.Vars(r)

		name := vars["id"]

		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		botCol := mongoDb.Collection("bots")

		var bot types.Bot

		var err error

		if r.URL.Query().Get("resolve") == "1" || r.URL.Query().Get("resolve") == "true" {
			err = botCol.FindOne(ctx, bson.M{
				"$or": []bson.M{
					{
						"botName": name,
					},
					{
						"vanity": name,
					},
					{
						"botID": name,
					},
				},
			}).Decode(&bot)
		} else {
			err = botCol.FindOne(ctx, bson.M{"botID": name}).Decode(&bot)
		}

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(notFound))
			return
		}

		bot = *utils.ParseBot(&bot)

		/* Removing or modifying fields directly in API is very dangerous as scrapers will
		 * just ignore owner checks anyways or cross-reference via another list. Also we
		 * want to respect the permissions of the owner if they're the one giving permission,
		 * blocking IPs is a better idea to this
		 */

		bytes, err := json.Marshal(bot)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte([]byte("{\"error\":\"Something broke!\"}")))
			return
		}

		w.Write(bytes)
	}))

	r.HandleFunc("/fates/bots/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(badRequest))
			return
		}

		vars := mux.Vars(r)

		botId := vars["id"]

		if botId == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		if r.Header.Get("Authorization") == "" || r.Header.Get("Authorization") != os.Getenv("FATES_TOKEN") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(badRequest))
			return
		}

		botCol := mongoDb.Collection("bots")

		var bot types.Bot

		err := botCol.FindOne(ctx, bson.M{"botID": botId}).Decode(&bot)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(notFound))
			return
		}

		bot = *utils.ParseBot(&bot)

		bytes, err := json.Marshal(bot)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte([]byte("{\"error\":\"Something broke!\"}")))
			return
		}

		w.Write(bytes)
	})

	adp := DummyAdapter{}

	integrase.StartServer(adp, r)
}
