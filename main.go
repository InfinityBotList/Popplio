package main

import (
	"context"
	"crypto/sha512"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"popplio/docs"
	"popplio/types"
	"popplio/utils"

	integrase "github.com/MetroReviews/metro-integrase/lib"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	mongoUrl          = "mongodb://127.0.0.1:27017/infinity" // Is already public in 10 other places so
	docsSite          = "https://docs.botlist.site"
	mainSite          = "https://infinitybotlist.com"
	statusPage        = "https://status.botlist.site"
	apiBot            = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
	pgConn            = "postgresql://127.0.0.1:5432/backups?user=root&password=iblpublic"
	voteTime   uint16 = 12 // 12 hours per vote

	notFound     = "{\"message\":\"Slow down, bucko! We couldn't find this resource *anywhere*!\"}"
	notFoundPage = "{\"message\":\"Slow down, bucko! You got the path wrong or something but this endpoint doesn't exist!\"}"
	badRequest   = "{\"message\":\"Slow down, bucko! You're doing something illegal!!!\"}"
	notApproved  = "{\"message\":\"Woah there, your bot needs to be approved. Calling the police right now over this infraction!\"}"
	backTick     = "`"
)

var (
	redisCache *redis.Client
	mongoDb    *mongo.Database
	pool       *pgxpool.Pool
	ctx        context.Context
	pgCtx      context.Context
	r          *mux.Router
)

func init() {
	godotenv.Load()
}

func rateLimitWrap(reqs int, t time.Duration, bucket string, fn http.HandlerFunc) http.HandlerFunc {
	reqStr := strconv.Itoa(reqs)
	timeStr := strconv.FormatFloat(t.Seconds(), 'g', -1, 64)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Ratelimit-Bucket", bucket)
		w.Header().Set("Ratelimit-Bucket-Reqs-Allowed-Count", reqStr)
		w.Header().Set("Ratelimit-Bucket-Reqs-Allowed-Second", timeStr)

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

		rlKey := "rl:" + id + "-" + bucket

		v := redisCache.Get(r.Context(), rlKey).Val()

		if v == "" {
			v = "0"

			err := redisCache.Set(ctx, rlKey, "0", t).Err()

			if err != nil {
				log.Error(err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("{\"error\":\"Something broke!\"}"))
				return
			}
		}

		err := redisCache.Incr(ctx, rlKey).Err()

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
			retryAfter := redisCache.TTL(ctx, rlKey).Val()
			w.Header().Set("Retry-After", strconv.FormatFloat(retryAfter.Seconds(), 'g', -1, 64))

			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("{\"error\":\"You're being rate limited!\"}"))

			return
		}

		w.Header().Set("Ratelimit-Req-Made", strconv.Itoa(vInt))

		fn(w, r)
	}
}

type Hello struct {
	Message string `json:"message"`
	Docs    string `json:"docs"`
	OurSite string `json:"our_site"`
	Status  string `json:"status"`
}

func main() {
	// Add the base tags
	docs.AddTag("System", "These API endpoints are core basic system APIs")
	docs.AddTag("Bots", "These API endpoints are related to bots on IBL")
	docs.AddTag("Votes", "These API endpoints are related to user votes on IBL")
	docs.AddTag("Variants", "These API endpoints are variants of other APIs or that do similar/same things as other API")

	ctx = context.Background()

	r = mux.NewRouter()

	// Init redisCache
	redisCache = redis.NewClient(&redis.Options{})

	pgCtx = context.Background()

	var err error

	pool, err = pgxpool.Connect(pgCtx, pgConn)

	if err != nil {
		panic(err)
	}

	// Create base payloads before startup
	// Index
	var helloWorldB Hello

	helloWorldB.Message = "Hello world from IBL API v5!"
	helloWorldB.Docs = docsSite
	helloWorldB.OurSite = mainSite
	helloWorldB.Status = statusPage

	helloWorld, err := json.Marshal(helloWorldB)

	if err != nil {
		panic(err)
	}

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

	metro, err = discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))

	if err != nil {
		panic(err)
	}

	docs.AddDocs("GET", "/", "ping", "Ping Server", "Pings the server", []docs.Paramater{}, []string{"System"}, nil, helloWorldB)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(helloWorld))
	})

	docs.AddDocs("GET", "/openapi", "openapi", "Get OpenAPI", "Gets the OpenAPI spec", []docs.Paramater{}, []string{"System"}, nil, map[string]any{})
	r.HandleFunc("/openapi", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		openapi, err := json.Marshal(docs.GetSchema())

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.Write([]byte(openapi))
	})

	r.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("html/docs.html")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		t.Execute(w, nil)
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

		var bot struct {
			BotID string `bson:"botID"`
			Type  string `bson:"type"`
		}

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(badRequest))
			return
		} else {
			options := options.FindOne().SetProjection(bson.M{"botID": 1, "type": 1})

			err := col.FindOne(ctx, bson.M{"token": r.Header.Get("Authorization")}, options).Decode(&bot)

			if err != nil {
				log.Error(err)
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
			if r.URL.Query().Get("count") != "" {
				payload = types.BotStats{}
			} else {
				log.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(badRequest))
				return
			}
		}

		if r.URL.Query().Get("count") != "" {
			count, err := strconv.ParseUint(r.URL.Query().Get("count"), 10, 32)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(badRequest))
				return
			}

			countPtr := uint32(count)

			payload.Count = &countPtr
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

	docs.AddDocs("POST", "/bots/stats", "post_stats", "Post New Stats", `
This endpoint can be used to post the stats of a bot.

The variation`+backTick+`/bots/{bot_id}/stats`+backTick+` can be used to post the stats of a bot. **Note that only the token is checked, not the bot ID at this time**

**Example:**

`+backTick+backTick+backTick+`py
import requests

req = requests.post(f"{API_URL}/bots/stats", json={"servers": 4000, "shards": 2}, headers={"Authorization": "{TOKEN}"})

print(req.json())
`+backTick+backTick+backTick+`

`, []docs.Paramater{}, []string{"Bots"}, types.BotStats{}, docs.ApiError{})

	docs.AddDocs("POST", "/bots/{id}/stats", "post_stats_variant2", "Post New Stats", `
This endpoint can be used to post the stats of a bot.

The variation`+backTick+`/bots/{bot_id}/stats`+backTick+` can be used to post the stats of a bot. **Note that only the token is checked, not the bot ID at this time**

**Example:**

`+backTick+backTick+backTick+`py
import requests

req = requests.post(f"{API_URL}/bots/stats", json={"servers": 4000, "shards": 2}, headers={"Authorization": "{TOKEN}"})

print(req.json())
`+backTick+backTick+backTick+`

`, []docs.Paramater{
		{
			Name:     "id",
			In:       "path",
			Required: true,
			Schema:   docs.IdSchema,
		},
	}, []string{"Variants"}, types.BotStats{}, docs.ApiError{})

	r.HandleFunc("/bots/stats", rateLimitWrap(4, 1*time.Minute, "stats", statsFn))

	// Note that only token matters for this endpoint at this time
	// TODO: Handle bot id as well
	r.HandleFunc("/bots/{id}/stats", rateLimitWrap(4, 1*time.Minute, "stats", statsFn))

	docs.AddDocs("GET", "/users/{uid}/bots/{bid}/votes", "get_user_votes", "Get User Votes", "Gets the users votes. **Requires authentication**", []docs.Paramater{
		{
			Name:     "uid",
			In:       "path",
			Required: true,
			Schema:   docs.IdSchema,
		},
		{
			Name:     "bid",
			In:       "path",
			Required: true,
			Schema:   docs.IdSchema,
		},
	}, []string{"Votes"}, nil, types.UserVote{
		Timestamps: []uint64{},
		VoteTime:   12,
		HasVoted:   true,
	})
	r.HandleFunc("/users/{uid}/bots/{bid}/votes", rateLimitWrap(3, 5*time.Minute, "gvotes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(badRequest))
			return
		}

		vars := mux.Vars(r)

		var bot struct {
			Type string `bson:"type"`
		}

		col := mongoDb.Collection("bots")

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(badRequest))
			return
		} else {
			options := options.FindOne().SetProjection(bson.M{"botID": 1, "type": 1})

			err := col.FindOne(ctx, bson.M{"token": r.Header.Get("Authorization"), "botID": vars["bid"]}, options).Decode(&bot)

			if err != nil {
				log.Error(err)
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

		var votes []uint64

		col = mongoDb.Collection("votes")

		cur, err := col.Find(ctx, bson.M{"botID": vars["bid"], "userID": vars["uid"]})

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(notFound))
			return
		}

		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var vote struct {
				Date uint64 `bson:"date"`
			}

			err := cur.Decode(&vote)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte([]byte("{\"error\":\"Something broke!\"}")))
				return
			}

			votes = append(votes, vote.Date)
		}

		voteParsed := types.UserVote{
			VoteTime: voteTime,
		}

		sort.Slice(votes, func(i, j int) bool { return votes[i] < votes[j] })

		voteParsed.Timestamps = votes

		if len(votes) > 0 {
			unixTs := time.Now().Unix()
			if uint64(unixTs)-votes[len(votes)-1] < uint64(voteTime*60*60) {
				voteParsed.HasVoted = true
			}
		}

		bytes, err := json.Marshal(voteParsed)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(badRequest))
			return
		}

		w.Write(bytes)
	}))

	r.HandleFunc("/voteinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(badRequest))
			return
		}

		// is it weekend yet?
		curTime := time.Now()

		var isWeekend bool

		switch curTime.Weekday() {
		// Friday is also a double vote
		case time.Friday:
			isWeekend = true
		case time.Saturday:
			isWeekend = true
		case time.Sunday:
			isWeekend = true
		}

		var payload = types.VoteInfo{
			Weekend: isWeekend,
		}

		b, err := json.Marshal(payload)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(badRequest))
			return
		}

		w.Write(b)
	})

	docs.AddDocs("GET", "/bots/{id}", "get_bot", "Get Bot", "Gets a bot by id or name, set ``resolve`` to true to also handle bot names."+`

- `+backTick+backTick+`external_source`+backTick+backTick+` shows the source of where a bot came from (Metro Reviews etc etc.). If this is set to `+backTick+backTick+`metro`+backTick+backTick+`, then `+backTick+backTick+`list_source`+backTick+backTick+` will be set to the metro list ID where it came from`+`
	`, []docs.Paramater{
		{
			Name:     "id",
			In:       "path",
			Required: true,
			Schema:   docs.IdSchema,
		},
		{
			Name:     "resolve",
			In:       "query",
			Required: true,
			Schema:   docs.BoolSchema,
		},
	}, []string{"Bots"}, nil, types.Bot{})

	getBotsFn := func(w http.ResponseWriter, r *http.Request) {
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

		// Check cache, this is how we can avoid hefty ratelimits
		cache := redisCache.Get(ctx, "bc-"+name).Val()
		if cache != "" {
			w.Header().Add("X-Popplio-Cached", "true")
			w.Write([]byte(cache))
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

		utils.ParseBot(&bot)

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

		redisCache.Set(ctx, "bc-"+name, string(bytes), time.Minute*3)

		w.Write(bytes)
	}

	r.HandleFunc("/users/{id}", rateLimitWrap(10, 3*time.Minute, "guser", func(w http.ResponseWriter, r *http.Request) {
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

		// Check cache, this is how we can avoid hefty ratelimits
		cache := redisCache.Get(ctx, "uc-"+name).Val()
		if cache != "" {
			w.Header().Add("X-Popplio-Cached", "true")
			w.Write([]byte(cache))
			return
		}

		userCol := mongoDb.Collection("users")

		var user types.User

		var err error

		if r.URL.Query().Get("resolve") == "1" || r.URL.Query().Get("resolve") == "true" {
			err = userCol.FindOne(ctx, bson.M{
				"$or": []bson.M{
					{
						"nickname": name,
					},
					{
						"userID": name,
					},
				},
			}).Decode(&user)
		} else {
			err = userCol.FindOne(ctx, bson.M{"userID": name}).Decode(&user)
		}

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(notFound))
			return
		}

		utils.ParseUser(&user)

		/* Removing or modifying fields directly in API is very dangerous as scrapers will
		 * just ignore owner checks anyways or cross-reference via another list. Also we
		 * want to respect the permissions of the owner if they're the one giving permission,
		 * blocking IPs is a better idea to this
		 */

		bytes, err := json.Marshal(user)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte([]byte("{\"error\":\"Something broke!\"}")))
			return
		}

		redisCache.Set(ctx, "uc-"+name, string(bytes), time.Minute*3)

		w.Write(bytes)
	}))

	r.HandleFunc("/bots/{id}", getBotsFn)

	r.HandleFunc("/bots/{id}/reviews", rateLimitWrap(10, 1*time.Minute, "greview", func(w http.ResponseWriter, r *http.Request) {
		col := mongoDb.Collection("reviews")

		vars := mux.Vars(r)

		name := vars["id"]

		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		var reviews []types.Review = []types.Review{}

		cur, err := col.Find(ctx, bson.M{"botID": name})

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(badRequest))
			return
		}

		for cur.Next(ctx) {
			var review types.Review

			err := cur.Decode(&review)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(badRequest))
				return
			}

			reviews = append(reviews, review)
		}

		bytes, err := json.Marshal(reviews)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(badRequest))
			return
		}

		w.Write(bytes)
	}))

	r.HandleFunc("/login/{act}", oauthFn)
	r.HandleFunc("/cosmog", performAct)
	r.HandleFunc("/cosmog/tasks/{tid}.arceus", getTask)
	r.HandleFunc("/cosmog/tasks/{tid}", taskFn)

	adp := DummyAdapter{}

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFoundPage))
	})

	integrase.StartServer(adp, r)
}
