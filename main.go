package main

import (
	"context"
	"crypto/sha512"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"os"
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
	ua "github.com/mileusna/useragent"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	mongoUrl   = "mongodb://127.0.0.1:27017/infinity" // Is already public in 10 other places so
	docsSite   = "https://docs.botlist.site"
	mainSite   = "https://infinitybotlist.com"
	statusPage = "https://status.botlist.site"
	apiBot     = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
	pgConn     = "postgresql://127.0.0.1:5432/backups?user=root&password=iblpublic"

	notFound         = "{\"message\":\"Slow down, bucko! We couldn't find this resource *anywhere*!\",\"error\":true}"
	notFoundPage     = "{\"message\":\"Slow down, bucko! You got the path wrong or something but this endpoint doesn't exist!\",\"error\":true}"
	badRequest       = "{\"message\":\"Slow down, bucko! You're doing something illegal!!!\",\"error\":true}"
	badRequestStats  = "{\"message\":\"Slow down, bucko! You're not posting stats correctly. Hint: try posting stats as integers and not as strings?\",\"error\":true}"
	unauthorized     = "{\"message\":\"Slow down, bucko! You're not authorized to do this or did you forget a API token somewhere?\",\"error\":true}"
	internalError    = "{\"message\":\"Slow down, bucko! Something went wrong on our end!\",\"error\":true}"
	methodNotAllowed = "{\"message\":\"Slow down, bucko! That method is not allowed for this endpoint!!!\",\"error\":true}"
	notApproved      = "{\"message\":\"Woah there, your bot needs to be approved. Calling the police right now over this infraction!\",\"error\":true}"
	voteBanned       = "{\"message\":\"Slow down, bucko! Either you or this bot is banned from voting right now!\",\"error\":true}"
	success          = "{\"message\":\"Success!\",\"error\":false}"
	testNotif        = "{\"message\":\"Test notification!\", \"title\":\"Test notification!\",\"icon\":\"https://i.imgur.com/GRo0Zug.png\",\"error\":false}"
	backTick         = "`"
)

// Represents a moderated bucket typically used in 'combined' endpoints like Get/Create Votes which are just branches off a common function
// This is also the concept used in so-called global ratelimits
type moderatedBucket struct {
	BucketName string

	// Internally set, dont change
	Global bool

	// Whether or not to keep original rl
	ChangeRL bool

	Requests int
	Time     time.Duration
}

var (
	redisCache *redis.Client
	mongoDb    *mongo.Database
	pool       *pgxpool.Pool
	ctx        context.Context
	pgCtx      context.Context
	r          *mux.Router

	// This is used when we need to moderate whether or not to ratelimit a request (such as on a combined endpoint like gvotes)
	bucketModerators map[string]func(r *http.Request) moderatedBucket = make(map[string]func(r *http.Request) moderatedBucket)

	// Default global ratelimit handler
	defaultGlobalBucket = moderatedBucket{BucketName: "global", Requests: 2000, Time: 1 * time.Hour}
)

func init() {
	godotenv.Load()
}

func bucketHandle(bucket moderatedBucket, id string, w http.ResponseWriter, r *http.Request) bool {

	rlKey := "rl:" + id + "-" + bucket.BucketName

	v := redisCache.Get(r.Context(), rlKey).Val()

	if v == "" {
		v = "0"

		err := redisCache.Set(ctx, rlKey, "0", bucket.Time).Err()

		if err != nil {
			log.Error(err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return false
		}
	}

	err := redisCache.Incr(ctx, rlKey).Err()

	if err != nil {
		log.Error(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(internalError))
		return false
	}

	vInt, err := strconv.Atoi(v)

	if err != nil {
		log.Error(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(internalError))
		return false
	}

	if vInt < 0 {
		redisCache.Expire(ctx, rlKey, 1*time.Second)
		vInt = 0
	}

	if vInt > bucket.Requests {
		w.Header().Set("Content-Type", "application/json")
		retryAfter := redisCache.TTL(ctx, rlKey).Val()

		if bucket.Global {
			w.Header().Set("X-Global-Ratelimit", "true")
		}

		w.Header().Set("Retry-After", strconv.FormatFloat(retryAfter.Seconds(), 'g', -1, 64))

		w.WriteHeader(http.StatusTooManyRequests)

		// Set ratelimit to expire in more time if not global
		if !bucket.Global {
			redisCache.Expire(ctx, rlKey, retryAfter+2*time.Second)
		}

		w.Write([]byte("{\"message\":\"You're being rate limited!\",\"error\":true}"))

		return false
	}

	if bucket.Global {
		w.Header().Set("X-Ratelimit-Global-Req-Made", strconv.Itoa(vInt))
	} else {
		w.Header().Set("X-Ratelimit-Req-Made", strconv.Itoa(vInt))
	}
	return true
}

func rateLimitWrap(reqs int, t time.Duration, bucket string, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if moderated buckets are needed, if so use them
		var reqBucket = moderatedBucket{}
		var globalBucket = defaultGlobalBucket

		if modBucket, ok := bucketModerators[bucket]; ok {
			log.Info("Found modBucket")
			modBucketData := modBucket(r)
			if modBucketData.ChangeRL {
				reqBucket = modBucketData
			} else {
				reqBucket.Requests = reqs
				reqBucket.Time = t
				reqBucket.BucketName = modBucketData.BucketName
			}
		} else {
			reqBucket.Requests = reqs
			reqBucket.Time = t
			reqBucket.BucketName = bucket
		}

		if modBucket, ok := bucketModerators["global"]; ok {
			log.Info("Found globalBucket")
			modBucketData := modBucket(r)
			if modBucketData.ChangeRL {
				globalBucket = modBucketData
			} else {
				globalBucket.Requests = reqs
				globalBucket.Time = t
				globalBucket.BucketName = modBucketData.BucketName
			}
		}

		globalBucket.Global = true // Just in case

		if strings.HasSuffix(r.Header.Get("Origin"), "infinitybots.gg") || strings.HasPrefix(r.Header.Get("Origin"), "localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")

		w.Header().Set("X-Ratelimit-Bucket", reqBucket.BucketName)
		w.Header().Set("X-Ratelimit-Bucket-Global", globalBucket.BucketName)

		w.Header().Set("X-Ratelimit-Bucket-Global-Reqs-Allowed-Count", strconv.Itoa(globalBucket.Requests))
		w.Header().Set("X-Ratelimit-Bucket-Reqs-Allowed-Count", strconv.Itoa(reqBucket.Requests))

		w.Header().Set("X-Ratelimit-Bucket-Global-Reqs-Allowed-Second", strconv.FormatFloat(globalBucket.Time.Seconds(), 'g', -1, 64))
		w.Header().Set("X-Ratelimit-Bucket-Reqs-Allowed-Second", strconv.FormatFloat(reqBucket.Time.Seconds(), 'g', -1, 64))

		if r.Method == "OPTIONS" {
			w.Write([]byte(""))
			return
		}

		// Get ratelimit from redis
		var id string

		auth := r.Header.Get("Authorization")

		if auth != "" {
			if strings.HasPrefix(auth, "User ") {
				rlId := strings.TrimPrefix(auth, "User ")

				if rlId == "" {
					// Bot does not exist, return
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("{\"error\":\"Invalid API token\"}"))
					return
				}

				userCol := mongoDb.Collection("users")

				var user struct {
					UserID string `bson:"userID"`
				}

				options := options.FindOne().SetProjection(bson.M{"userID": 1})

				err := userCol.FindOne(ctx, bson.M{"apiToken": strings.Replace(rlId, "User ", "", 1)}, options).Decode(&user)

				if err != nil {
					// Bot does not exist, return
					log.Error(err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("{\"error\":\"Invalid API token\"}"))
					return
				}

				id = user.UserID
			} else {

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
			}
		} else {
			remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

			// For user privacy, hash the remote ip
			hasher := sha512.New()
			hasher.Write([]byte(remoteIp[0]))
			id = fmt.Sprintf("%x", hasher.Sum(nil))
		}

		if ok := bucketHandle(globalBucket, id, w, r); !ok {
			return
		}

		if ok := bucketHandle(reqBucket, id, w, r); !ok {
			return
		}

		w.Header().Set("Content-Type", "application/json")

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
	docs.AddTag("Users", "These API endpoints are related to users on IBL")
	docs.AddTag("Votes", "These API endpoints are related to user votes on IBL")
	docs.AddTag("Variants", "These API endpoints are variants of other APIs or that do similar/same things as other API")

	docs.AddSecuritySchema("User", "Requires a user token. Usually must be prefixed with `User `")
	docs.AddSecuritySchema("Bot", "Requires a bot token. Cannot be prefixed")
	docs.AddSecuritySchema("None", "No authentication required however some APIs may not return all data")

	ctx = context.Background()

	r = mux.NewRouter()

	// Init redisCache
	rOptions, err := redis.ParseURL("redis://localhost:6379")

	if err != nil {
		panic(err)
	}

	redisCache = redis.NewClient(rOptions)

	pgCtx = context.Background()

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

	metro.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentGuildPresences | discordgo.IntentsGuildMembers

	err = metro.Open()
	if err != nil {
		panic(err)
	}

	docs.AddDocs("GET", "/", "ping", "Ping Server", "Pings the server", []docs.Paramater{}, []string{"System"}, nil, helloWorldB, []string{})
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(helloWorld))
	})

	docs.AddDocs("GET", "/announcements", "announcements", "Get Announcements", "Gets the announcements. User authentication is optional and using it will show user targetted announcements", []docs.Paramater{}, []string{"System"}, nil, types.Announcement{}, []string{"User"})
	r.HandleFunc("/announcements", rateLimitWrap(30, 1*time.Minute, "gannounce", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		col := mongoDb.Collection("announcements")

		var announcements []types.Announcement

		cur, err := col.Find(ctx, bson.M{})

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		// Auth header check
		auth := r.Header.Get("Authorization")

		var target types.UserID

		if auth != "" {
			err := mongoDb.Collection("users").FindOne(ctx, bson.M{"apiToken": strings.Replace(auth, "User ", "", 1)}).Decode(&target)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(unauthorized))
				return
			}
		} else {
			target = types.UserID{}
		}

		for cur.Next(ctx) {
			var announcement types.Announcement

			err := cur.Decode(&announcement)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				continue
			}

			if announcement.Status == "private" {
				// Staff only
				continue
			}

			if announcement.Targetted {
				// Check auth header
				if target.UserID != announcement.Target {
					continue
				}
			}

			announcements = append(announcements, announcement)
		}

		bytes, err := json.Marshal(announcements)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		w.Write(bytes)
	}))

	r.HandleFunc("/_duser/{id}", func(w http.ResponseWriter, r *http.Request) {
		var id = mux.Vars(r)["id"]

		user, err := utils.GetDiscordUser(metro, redisCache, ctx, id)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		bytes, err := json.Marshal(user)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.Write(bytes)
	})

	docs.AddDocs("GET", "/openapi", "openapi", "Get OpenAPI", "Gets the OpenAPI spec", []docs.Paramater{}, []string{"System"}, nil, map[string]any{}, []string{})
	r.HandleFunc("/openapi", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

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
			w.Write([]byte(methodNotAllowed))
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
		}

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(unauthorized))
			return
		} else {
			options := options.FindOne().SetProjection(bson.M{"botID": 1})

			err := col.FindOne(ctx, bson.M{"token": r.Header.Get("Authorization")}, options).Decode(&bot)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(unauthorized))
				return
			}
		}

		defer r.Body.Close()

		var payload types.BotStats

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		err = json.Unmarshal(bodyBytes, &payload)

		if err != nil {
			if r.URL.Query().Get("count") != "" {
				payload = types.BotStats{}
			} else {
				log.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(badRequestStats))
				return
			}
		}

		if r.URL.Query().Get("count") != "" {
			count, err := strconv.ParseUint(r.URL.Query().Get("count"), 10, 32)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(badRequestStats))
				return
			}

			var countAny any = count

			payload.Count = &countAny
		}

		servers, shards, users := payload.GetStats()

		if servers > 0 {
			_, err = col.UpdateOne(ctx, bson.M{"token": r.Header.Get("Authorization")}, bson.M{"$set": bson.M{"servers": servers}})

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}
		}

		if shards > 0 {
			_, err = col.UpdateOne(ctx, bson.M{"token": r.Header.Get("Authorization")}, bson.M{"$set": bson.M{"shards": shards}})

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}
		}

		if users > 0 {
			_, err = col.UpdateOne(ctx, bson.M{"token": r.Header.Get("Authorization")}, bson.M{"$set": bson.M{"users": users}})

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}
		}

		w.Write([]byte(success))
	}

	docs.AddDocs("GET", "/bots/all", "get_all_bots", "Get All Bots", "Gets all bots on the list", []docs.Paramater{}, []string{"System"}, nil, types.AllBots{}, []string{})
	r.Handle("/bots/all", rateLimitWrap(5, 2*time.Second, "allbots", func(w http.ResponseWriter, r *http.Request) {
		const perPage = 10

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		page := r.URL.Query().Get("page")

		if page == "" {
			page = "1"
		}

		pageNum, err := strconv.ParseUint(page, 10, 32)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		limit := perPage
		offset := (pageNum - 1) * perPage

		options := options.Find().SetSort(bson.M{"created": -1}).SetLimit(int64(limit)).SetSkip(int64(offset))

		col := mongoDb.Collection("bots")

		var bots []types.Bot

		cur, err := col.Find(ctx, bson.M{}, options)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var bot types.Bot

			err := cur.Decode(&bot)

			if err != nil {
				log.Error(err)
				continue
			}

			bots = append(bots, bot)
		}

		if err := cur.Err(); err != nil {
			log.Error(err)
		}

		var previous strings.Builder

		// More optimized string concat
		previous.WriteString(os.Getenv("SITE_URL"))
		previous.WriteString("/bots/all?page=")
		previous.WriteString(strconv.FormatUint(pageNum-1, 10))

		if pageNum-1 < 1 || pageNum == 0 {
			previous.Reset()
		}

		count, err := col.CountDocuments(ctx, bson.M{})

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		var next strings.Builder

		next.WriteString(os.Getenv("SITE_URL"))
		next.WriteString("/bots/all?page=")
		next.WriteString(strconv.FormatUint(pageNum+1, 10))

		if float64(pageNum+1) > math.Ceil(float64(count)/perPage) {
			next.Reset()
		}

		data := types.AllBots{
			Count:    count,
			Results:  bots,
			PerPage:  perPage,
			Previous: previous.String(),
			Next:     next.String(),
		}

		bytes, err := json.Marshal(data)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		w.Write(bytes)

	}))

	docs.AddDocs("POST", "/bots/stats", "post_stats", "Post New Stats", `
This endpoint can be used to post the stats of a bot.

The variation`+backTick+`/bots/{bot_id}/stats`+backTick+` can be used to post the stats of a bot. **Note that only the token is checked, not the bot ID at this time**

**Example:**

`+backTick+backTick+backTick+`py
import requests

req = requests.post(f"{API_URL}/bots/stats", json={"servers": 4000, "shards": 2}, headers={"Authorization": "{TOKEN}"})

print(req.json())
`+backTick+backTick+backTick+`

`, []docs.Paramater{}, []string{"Bots"}, types.BotStatsTyped{}, types.ApiError{}, []string{"Bot"})

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
	}, []string{"Variants"}, types.BotStatsTyped{}, types.ApiError{}, []string{"Bot"})

	r.HandleFunc("/bots/stats", rateLimitWrap(10, 1*time.Minute, "stats", statsFn))

	// Note that only token matters for this endpoint at this time
	// TODO: Handle bot id as well
	r.HandleFunc("/bots/{id}/stats", rateLimitWrap(10, 1*time.Minute, "stats", statsFn))

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
		Timestamps: []int64{},
		VoteTime:   12,
		HasVoted:   true,
	}, []string{"User", "Bot"})
	r.HandleFunc("/users/{uid}/bots/{bid}/votes", rateLimitWrap(5, 1*time.Minute, "gvotes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "GET" && r.Method != "PUT" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		vars := mux.Vars(r)

		var bot struct {
			BotID      string `bson:"botID"`
			Type       string `bson:"type"`
			VoteBanned bool   `bson:"vote_banned,omitempty"`
		}

		col := mongoDb.Collection("bots")

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(unauthorized))
			return
		} else {
			if strings.HasPrefix(r.Header.Get("Authorization"), "User ") {
				userCol := mongoDb.Collection("users")

				var user struct {
					VoteBanned bool `bson:"vote_banned,omitempty"`
				}

				err := userCol.FindOne(ctx, bson.M{"userID": vars["uid"], "apiToken": strings.Replace(r.Header.Get("Authorization"), "User ", "", 1)}).Decode(&user)

				if err == mongo.ErrNoDocuments {
					log.Error(err)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(unauthorized))
					return
				} else if err != nil {
					log.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(internalError))
					return
				}

				if user.VoteBanned && r.Method == "PUT" {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(voteBanned))
					return
				}

				options := options.FindOne().SetProjection(bson.M{"botID": 1, "type": 1})

				err = col.FindOne(
					ctx,
					bson.M{
						"$or": []bson.M{
							{
								"botName": vars["bid"],
							},
							{
								"vanity": vars["bid"],
							},
							{
								"botID": vars["bid"],
							},
						},
					},
					options,
				).Decode(&bot)

				vars["bid"] = bot.BotID

				if err != nil {
					log.Error(err)
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(notFound))
					return
				}

			} else {
				options := options.FindOne().SetProjection(bson.M{"botID": 1, "type": 1})

				err = col.FindOne(
					ctx,
					bson.M{
						"$or": []bson.M{
							{
								"botName": vars["bid"],
							},
							{
								"vanity": vars["bid"],
							},
							{
								"botID": vars["bid"],
							},
						},
					},
					options,
				).Decode(&bot)

				vars["bid"] = bot.BotID

				if err != nil {
					log.Error(err)
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(notFound))
					return
				}

				err := col.FindOne(ctx, bson.M{"token": r.Header.Get("Authorization"), "botID": vars["bid"]}, options).Decode(&bot)

				if err != nil {
					log.Error(err)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(unauthorized))
					return
				}
			}
		}

		if bot.Type != "approved" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(notApproved))
			return
		}

		if bot.VoteBanned {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(voteBanned))
			return
		}

		voteParsed, err := utils.GetVoteData(ctx, mongoDb, vars["uid"], vars["bid"])

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		if r.Method == "GET" {
			bytes, err := json.Marshal(voteParsed)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(badRequest))
				return
			}

			w.Write(bytes)
		} else if r.Method == "PUT" {
			if voteParsed.HasVoted {
				timeElapsed := time.Now().UnixMilli() - voteParsed.LastVoteTime
				log.Info(timeElapsed)

				timeToWait := int64(utils.GetVoteTime())*60*60*1000 - timeElapsed

				timeToWaitTime := (time.Duration(timeToWait) * time.Millisecond)

				hours := timeToWaitTime / time.Hour
				mins := (timeToWaitTime - (hours * time.Hour)) / time.Minute
				secs := (timeToWaitTime - (hours*time.Hour + mins*time.Minute)) / time.Second

				timeStr := fmt.Sprintf("%02d hours, %02d minutes. %02d seconds", hours, mins, secs)

				var alreadyVotedMsg = types.ApiError{
					Message: "Please wait " + timeStr + " before voting again",
					Error:   true,
				}

				bytes, err := json.Marshal(alreadyVotedMsg)

				if err != nil {
					log.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(internalError))
					return
				}

				w.WriteHeader(http.StatusBadRequest)
				w.Write(bytes)
				return
			}

			// Record new vote
			r, err := mongoDb.Collection("votes").InsertOne(ctx, bson.M{"botID": vars["bid"], "userID": vars["uid"], "date": time.Now().UnixMilli()})

			if err != nil {
				// Revert vote
				_, err := mongoDb.Collection("votes").DeleteOne(ctx, bson.M{"_id": r.InsertedID})
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			var oldVotes struct {
				Votes int `bson:"votes"`
			}

			err = mongoDb.Collection("bots").FindOne(ctx, bson.M{"botID": vars["bid"]}, options.FindOne().SetProjection(bson.M{"votes": 1})).Decode(&oldVotes)

			if err != nil {
				// Revert vote
				_, err := mongoDb.Collection("votes").DeleteOne(ctx, bson.M{"_id": r.InsertedID})

				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			var incr int = 1

			if utils.GetDoubleVote() {
				oldVotes.Votes += 2
				incr = 2
			} else {
				oldVotes.Votes++
			}

			_, err = mongoDb.Collection("bots").UpdateOne(ctx, bson.M{"botID": vars["bid"]}, bson.M{"$inc": bson.M{"votes": incr}})

			if err != nil {
				// Revert vote
				_, err := mongoDb.Collection("votes").DeleteOne(ctx, bson.M{"_id": r.InsertedID})

				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			userObj, err := utils.GetDiscordUser(metro, redisCache, ctx, vars["uid"])

			if err != nil {
				// Revert vote
				_, err := mongoDb.Collection("votes").DeleteOne(ctx, bson.M{"_id": r.InsertedID})

				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			botObj, err := utils.GetDiscordUser(metro, redisCache, ctx, vars["bid"])

			if err != nil {
				// Revert vote
				_, err := mongoDb.Collection("votes").DeleteOne(ctx, bson.M{"_id": r.InsertedID})

				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			messageNotifyChannel <- types.DiscordLog{
				WebhookID:    os.Getenv("VOTE_LOG_WEBHOOK_ID"),
				WebhookToken: os.Getenv("VOTE_LOG_WEBHOOK_TOKEN"),
				WebhookData: &discordgo.WebhookParams{
					AvatarURL: botObj.Avatar,
					Embeds: []*discordgo.MessageEmbed{
						{
							URL: "https://botlist.site/" + vars["bid"],
							Thumbnail: &discordgo.MessageEmbedThumbnail{
								URL: botObj.Avatar,
							},
							Title:       "🎉 Vote Count Updated!",
							Description: ":heart:" + userObj.Username + "#" + userObj.Discriminator + " has voted for " + botObj.Username,
							Color:       0x8A6BFD,
							Fields: []*discordgo.MessageEmbedField{
								{
									Name:   "Vote Count:",
									Value:  strconv.Itoa(oldVotes.Votes),
									Inline: true,
								},
								{
									Name:   "User ID:",
									Value:  userObj.ID,
									Inline: true,
								},
								{
									Name:   "Vote Page",
									Value:  "[View " + botObj.Username + "](https://botlist.site/" + vars["bid"] + ")",
									Inline: true,
								},
								{
									Name:   "Vote Page",
									Value:  "[Vote for " + botObj.Username + "](https://botlist.site/" + vars["bid"] + "/vote)",
									Inline: true,
								},
							},
						},
					},
				},
			}

			// Send webhook in a goroutine refunding the vote if it failed
			go func() {
				err = sendWebhook(types.WebhookPost{
					BotID:  vars["bid"],
					UserID: vars["uid"],
					Votes:  oldVotes.Votes,
				})

				if err != nil {
					mongoDb.Collection("notifications").InsertOne(ctx, bson.M{
						"userID":  vars["uid"],
						"url":     "https://infinitybots.gg/bots/" + vars["bid"],
						"message": "Whoa there! We've failed to notify this bot about this vote. The error was: " + err.Error() + ".",
						"type":    "error",
					})
				} else {
					mongoDb.Collection("notifications").InsertOne(ctx, bson.M{
						"userID":  vars["uid"],
						"url":     "https://infinitybots.gg/bots/" + vars["bid"],
						"message": "Successfully voted for bot with ID of " + vars["bid"],
						"type":    "info",
					})
				}
			}()

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(success))
		}
	}))

	// For compatibility with old API
	r.HandleFunc("/votes/{bot_id}/{user_id}", rateLimitWrap(10, 1*time.Minute, "deprecated-gvotes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		vars := mux.Vars(r)

		var bot struct {
			BotID      string `bson:"botID"`
			Type       string `bson:"type,omitempty"`
			VoteBanned bool   `bson:"vote_banned,omitempty"`
		}

		col := mongoDb.Collection("bots")

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(unauthorized))
			return
		} else {
			options := options.FindOne().SetProjection(bson.M{"botID": 1, "type": 1, "vote_banned": 1})

			err := col.FindOne(ctx, bson.M{"token": r.Header.Get("Authorization"), "botID": vars["bot_id"]}, options).Decode(&bot)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(unauthorized))
				return
			}

			vars["bot_id"] = bot.BotID
		}

		if bot.Type != "approved" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(notApproved))
			return
		}

		if bot.VoteBanned {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(voteBanned))
			return
		}

		voteParsed, err := utils.GetVoteData(ctx, mongoDb, vars["user_id"], vars["bot_id"])

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		var compatData = types.UserVoteCompat{
			HasVoted: voteParsed.HasVoted,
		}

		bytes, err := json.Marshal(compatData)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(badRequest))
			return
		}

		w.Write(bytes)
	}))

	docs.AddDocs("GET", "/voteinfo", "voteinfo", "Get Vote Info", "Returns basic voting info such as if its a weekend double vote", []docs.Paramater{}, []string{"Votes"}, nil, types.VoteInfo{Weekend: true}, []string{})
	r.HandleFunc("/voteinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		var payload = types.VoteInfo{
			Weekend: utils.GetDoubleVote(),
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
	}, []string{"Bots"}, nil, types.Bot{}, []string{})

	getBotsFn := rateLimitWrap(10, 1*time.Minute, "gbot", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "POST" {
			statsFn(w, r)
			return
		}

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
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
			w.Write([]byte(internalError))
			return
		}

		redisCache.Set(ctx, "bc-"+name, string(bytes), time.Minute*3)

		w.Write(bytes)
	})

	docs.AddDocs("GET", "/users/{id}", "get_user", "Get User", "Gets a user by id or name, set ``resolve`` to true to also handle user names.",
		[]docs.Paramater{
			{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   docs.IdSchema,
			},
		}, []string{"Users"}, nil, types.User{}, []string{})

	r.HandleFunc("/users/{id}", rateLimitWrap(10, 3*time.Minute, "guser", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
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
			w.Write([]byte(internalError))
			return
		}

		redisCache.Set(ctx, "uc-"+name, string(bytes), time.Minute*3)

		w.Write(bytes)
	}))

	r.HandleFunc("/bots/{id}", getBotsFn)
	r.HandleFunc("/bot/{id}", getBotsFn)

	docs.AddDocs("GET", "/bots/{id}/reviews", "get_bot_reviews", "Get Bot Reviews", "Gets the reviews of a bot by its ID (names are not resolved by this endpoint)",
		[]docs.Paramater{
			{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   docs.IdSchema,
			},
		}, []string{"Bots"}, nil, []types.Review{}, []string{})
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

	docs.AddDocs("POST", "/webhook-test", "webhook_test", "Test Webhook", "Sends a test webhook to allow testing your vote system. **All fields are mandatory for test bot**",
		[]docs.Paramater{}, []string{"System"}, types.WebhookPost{}, types.ApiError{}, []string{})

	r.HandleFunc("/webhook-test", rateLimitWrap(7, 3*time.Minute, "webtest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		defer r.Body.Close()

		var payload types.WebhookPost

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		err = json.Unmarshal(bodyBytes, &payload)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		if utils.IsNone(&payload.URL) && utils.IsNone(&payload.URL2) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		payload.Test = true // Always true

		var err1 error

		if !utils.IsNone(&payload.URL) {
			err1 = sendWebhook(payload)
		}

		var err2 error

		if !utils.IsNone(&payload.URL2) {
			payload.URL = payload.URL2 // Test second enpdoint if it's not empty
			err2 = sendWebhook(payload)
		}

		var errD = types.ApiError{}

		if err1 != nil {
			log.Error(err)

			errD.Message = err.Error()
			errD.Error = true
		}

		if err2 != nil {
			log.Error(err2)

			errD.Message += err2.Error()
			errD.Error = true
		}

		bytes, err := json.Marshal(errD)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write(bytes)
	}))

	// Internal APIs
	r.HandleFunc("/_protozoa/profile/{id}", rateLimitWrap(7, 1*time.Minute, "profile_update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		id := mux.Vars(r)["id"]

		// Fetch auth from mongodb
		col := mongoDb.Collection("users")

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(unauthorized))
			return
		} else {
			options := options.FindOne().SetProjection(bson.M{"userID": 1})

			err := col.FindOne(ctx, bson.M{"apiToken": strings.Replace(r.Header.Get("Authorization"), "User ", "", 1), "userID": id}, options).Err()

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(unauthorized))
				return
			}
		}

		// Fetch profile update from body
		var profile types.ProfileUpdate

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		err = json.Unmarshal(bodyBytes, &profile)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		if profile.About != "" {
			// Update about
			mongoDb.Collection("users").UpdateOne(ctx, bson.M{"userID": id}, bson.M{"$set": bson.M{"about": profile.About}})
		}
	}))

	r.HandleFunc("/_protozoa/notifications/info", rateLimitWrap(10, 1*time.Minute, "notif_info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		data := map[string]any{
			"public_key": os.Getenv("VAPID_PUBLIC_KEY"),
		}

		bytes, err := json.Marshal(data)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		w.Write(bytes)
	}))

	r.HandleFunc("/_protozoa/notifications/{id}", rateLimitWrap(40, 1*time.Minute, "get_notifs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "DELETE" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		id := mux.Vars(r)["id"]

		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		// Fetch auth from mongodb
		col := mongoDb.Collection("users")

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(unauthorized))
			return
		} else {
			options := options.FindOne().SetProjection(bson.M{"userID": 1})

			err := col.FindOne(ctx, bson.M{"apiToken": strings.Replace(r.Header.Get("Authorization"), "User ", "", 1), "userID": id}, options).Err()

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(unauthorized))
				return
			}
		}

		if r.Method == "GET" {
			var subscription []struct {
				Endpoint    string                 `bson:"endpoint" json:"endpoint"`
				NotifID     string                 `bson:"notifId" json:"notif_id"`
				CreatedAt   time.Time              `bson:"createdAt" json:"created_at"`
				UA          string                 `bson:"ua" json:"-"`
				BrowserInfo types.NotifBrowserInfo `bson:"-" json:"browser_info"`
			}

			cur, err := mongoDb.Collection("poppypaw").Find(ctx, bson.M{"id": id})

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			err = cur.All(ctx, &subscription)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			if len(subscription) == 0 {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(notFound))
				return
			}

			for i, val := range subscription {
				// Parse UA
				uaD := ua.Parse(val.UA)
				fmt.Println("Parsing UA:", val.UA, uaD)

				subscription[i].BrowserInfo = types.NotifBrowserInfo{
					OS:         uaD.OS,
					Browser:    uaD.Name,
					BrowserVer: uaD.Version,
					Mobile:     uaD.Mobile,
				}
			}

			bytes, err := json.Marshal(subscription)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			w.Write(bytes)
		} else {
			// Delete the notif
			if r.URL.Query().Get("notif_id") == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(badRequest))
				return
			}
			_, err := mongoDb.Collection("poppypaw").DeleteOne(ctx, bson.M{"notifId": r.URL.Query().Get("notif_id")})

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			w.WriteHeader(http.StatusOK)
		}
	}))

	r.HandleFunc("/_protozoa/notifications/{id}/sub", rateLimitWrap(10, 1*time.Minute, "notif_info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		var subscription struct {
			Auth     string `json:"auth"`
			P256dh   string `json:"p256dh"`
			Endpoint string `json:"endpoint"`
		}

		vars := mux.Vars(r)

		id := vars["id"]

		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		defer r.Body.Close()

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		err = json.Unmarshal(bodyBytes, &subscription)

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(internalError))
			return
		}

		if subscription.Auth == "" || subscription.P256dh == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		// Fetch auth from mongodb
		col := mongoDb.Collection("users")

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(unauthorized))
			return
		} else {
			options := options.FindOne().SetProjection(bson.M{"userID": 1})

			err := col.FindOne(ctx, bson.M{"apiToken": strings.Replace(r.Header.Get("Authorization"), "User ", "", 1), "userID": id}, options).Err()

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(unauthorized))
				return
			}
		}

		// Store new subscription

		notifId := utils.RandString(512)

		col = mongoDb.Collection("poppypaw")

		ua := r.UserAgent()

		if ua == "" {
			ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"
		}

		col.DeleteMany(ctx, bson.M{"id": id, "endpoint": subscription.Endpoint})

		col.InsertOne(ctx, bson.M{
			"id":        id,
			"notifId":   notifId,
			"auth":      subscription.Auth,
			"p256dh":    subscription.P256dh,
			"endpoint":  subscription.Endpoint,
			"createdAt": time.Now(),
			"ua":        r.UserAgent(),
		})

		// Fan out test notification
		notifChannel <- types.Notification{
			NotifID: notifId,
			Message: []byte(testNotif),
		}

		w.Write([]byte(success))
	}))

	r.HandleFunc("/_protozoa/reminders/{id}", rateLimitWrap(40, 1*time.Minute, "greminder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" && r.Method != "GET" && r.Method != "DELETE" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(methodNotAllowed))
			return
		}

		vars := mux.Vars(r)

		id := vars["id"]

		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(badRequest))
			return
		}

		// Fetch auth from mongodb
		col := mongoDb.Collection("users")

		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(unauthorized))
			return
		} else {
			options := options.FindOne().SetProjection(bson.M{"userID": 1})

			err := col.FindOne(ctx, bson.M{"apiToken": strings.Replace(r.Header.Get("Authorization"), "User ", "", 1), "userID": id}, options).Err()

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(unauthorized))
				return
			}
		}

		if r.Method == "GET" {
			// Fetch reminder from mongodb
			col = mongoDb.Collection("silverpelt")

			var reminder []types.Reminder

			cur, err := col.Find(ctx, bson.M{"userID": id})

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			defer cur.Close(ctx)

			for cur.Next(ctx) {
				var r types.Reminder

				err := cur.Decode(&r)

				if err != nil {
					log.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(internalError))
					return
				}

				// Try resolving the bot from discord API
				var resolvedBot types.ResolvedReminderBot
				bot, err := utils.GetDiscordUser(metro, redisCache, ctx, r.BotID)

				if err != nil {
					resolvedBot = types.ResolvedReminderBot{
						Name:   "Unknown",
						Avatar: "https://cdn.discordapp.com/embed/avatars/0.png",
					}
				} else {
					resolvedBot = types.ResolvedReminderBot{
						Name:   bot.Username,
						Avatar: bot.Avatar,
					}
				}

				r.ResolvedBot = resolvedBot

				reminder = append(reminder, r)
			}

			bytes, err := json.Marshal(reminder)

			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(internalError))
				return
			}

			w.Write(bytes)
		} else if r.Method == "PUT" {
			// Add subscription to collection
			bid := r.URL.Query().Get("bot_id")

			var bot struct {
				ID string `bson:"botID"`
			}

			options := options.FindOne().SetProjection(bson.M{"botID": 1})

			err = mongoDb.Collection("bots").FindOne(
				ctx,
				bson.M{
					"$or": []bson.M{
						{
							"botName": bid,
						},
						{
							"vanity": bid,
						},
						{
							"botID": bid,
						},
					},
				},
				options,
			).Decode(&bot)

			if err != nil {
				log.Error("Error adding reminder: ", err)
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(notFound))
				return
			}

			if bot.ID == "" {
				log.Error("No bot id found?")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(badRequest))
				return
			}

			err := mongoDb.Collection("silverpelt").FindOne(ctx, bson.M{"userID": id, "botID": bot.ID}).Err()

			// If reminder already exists, delete them all first to protect against db spam
			if err == nil {
				mongoDb.Collection("silverpelt").DeleteMany(ctx, bson.M{"userID": id, "botID": bot.ID})
			}

			mongoDb.Collection("silverpelt").InsertOne(ctx, bson.M{
				"userID":    id,
				"botID":     bot.ID,
				"createdAt": time.Now().Unix(),
				"lastAcked": 0,
			})

			w.Write([]byte(success))
		} else {
			botId := r.URL.Query().Get("bot_id")

			if botId == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(badRequest))
				return
			}

			// Delete reminder from mongodb
			mongoDb.Collection("silverpelt").DeleteMany(ctx, bson.M{"userID": id, "botID": botId})

			w.WriteHeader(http.StatusOK)
		}
	}))

	adp := DummyAdapter{}

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFoundPage))
	})

	createBucketMods()

	integrase.StartServer(adp, r)
}
