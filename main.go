package main

import (
	"context"
	"crypto/sha512"
	"fmt"
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
	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	jsoniter "github.com/json-iterator/go"
	ua "github.com/mileusna/useragent"
	log "github.com/sirupsen/logrus"

	_ "embed"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

//go:embed html/ext.js
var extUnminified string

//go:embed html/docs.html
var docsHTML string

func init() {
	m := minify.New()
	m.AddFunc("application/javascript", js.Minify)
	m.AddFunc("text/css", css.Minify)

	strWriter := &strings.Builder{}

	strReader := strings.NewReader(extUnminified)

	if err := m.Minify("application/javascript", strWriter, strReader); err != nil {
		panic(err)
	}

	docsJs = strWriter.String()

	docsHTML = strings.Replace(docsHTML, "[JS]", docsJs, 1)
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	docsSite   = "https://spider.infinitybotlist.com/docs"
	mainSite   = "https://infinitybotlist.com"
	statusPage = "https://status.botlist.site"
	apiBot     = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
	pgConn     = "postgresql:///infinity"
	backupConn = "postgresql:///backups"

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

	// Whether or not to just bypass the ratelimit altogether
	Bypass bool
}

var (
	redisCache *redis.Client
	iblCache   *redis.Client
	pool       *pgxpool.Pool
	backupPool *pgxpool.Pool
	ctx        context.Context

	docsJs  string
	openapi []byte

	// This is used when we need to moderate whether or not to ratelimit a request (such as on a combined endpoint like gvotes)
	bucketModerators map[string]func(r *http.Request) moderatedBucket = make(map[string]func(r *http.Request) moderatedBucket)

	// Default global ratelimit handler
	defaultGlobalBucket = moderatedBucket{BucketName: "global", Requests: 500, Time: 2 * time.Minute}

	announcementCols = utils.GetCols(types.Announcement{})

	announcementColsStr = strings.Join(announcementCols, ",")

	botsCols    = utils.GetCols(types.Bot{})
	botsColsStr = strings.Join(botsCols, ",")

	packsCols       = utils.GetCols(types.BotPack{})
	packsColsString = strings.Join(packsCols, ",")

	usersCols    = utils.GetCols(types.User{})
	usersColsStr = strings.Join(usersCols, ",")

	reviewCols    = utils.GetCols(types.Review{})
	reviewColsStr = strings.Join(reviewCols, ",")

	indexBotsCol    = utils.GetCols(types.IndexBot{})
	indexBotsColStr = strings.Join(indexBotsCol, ",")

	silverpeltCols = utils.GetCols(types.Reminder{})

	silverpeltColsStr = strings.Join(silverpeltCols, ",")
)

func init() {
	godotenv.Load()
}

func apiDefaultReturn(statusCode int, w http.ResponseWriter, r *http.Request) {
	switch statusCode {
	case http.StatusUnauthorized:
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(unauthorized))
	case http.StatusNotFound:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFound))
	case http.StatusBadRequest:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(badRequest))
	case http.StatusInternalServerError:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(internalError))
	case http.StatusMethodNotAllowed:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(methodNotAllowed))
	}
}

func authCheck(token string, bot bool) *string {
	if token == "" {
		return nil
	}

	if bot {
		var id pgtype.Text
		err := pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE token = $1", strings.Replace(token, "Bot ", "", 1)).Scan(&id)

		if err != nil {
			fmt.Println(err)
			return nil
		} else {
			if id.Status == pgtype.Null {
				return nil
			}
			return &id.String
		}
	} else {
		var id pgtype.Text
		err := pool.QueryRow(ctx, "SELECT user_id FROM users WHERE api_token = $1", strings.Replace(token, "User ", "", 1)).Scan(&id)

		if err != nil {
			fmt.Println(err)
			return nil
		} else {
			if id.Status == pgtype.Null {
				return nil
			}
			return &id.String
		}
	}
}

func bucketHandle(bucket moderatedBucket, id string, w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("CF-RAY") == "" && r.Header.Get("X-Forwarded-For") == "" {
		return true // Don't ratelimit internal API calls, the internal API should itself be handling ratelimits there
	} else if bucket.Bypass {
		return true // Don't ratelimit bypass buckets
	}

	rlKey := "rl:" + id + "-" + bucket.BucketName

	v := redisCache.Get(r.Context(), rlKey).Val()

	if v == "" {
		v = "0"

		err := redisCache.Set(ctx, rlKey, "0", bucket.Time).Err()

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return false
		}
	}

	err := redisCache.Incr(ctx, rlKey).Err()

	if err != nil {
		log.Error(err)
		apiDefaultReturn(http.StatusInternalServerError, w, r)
		return false
	}

	vInt, err := strconv.Atoi(v)

	if err != nil {
		log.Error(err)
		apiDefaultReturn(http.StatusInternalServerError, w, r)
		return false
	}

	if vInt < 0 {
		redisCache.Expire(ctx, rlKey, 1*time.Second)
		vInt = 0
	}

	if vInt > bucket.Requests {
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

		w.Header().Set("X-Ratelimit-Bucket", reqBucket.BucketName)
		w.Header().Set("X-Ratelimit-Bucket-Global", globalBucket.BucketName)

		w.Header().Set("X-Ratelimit-Bucket-Global-Reqs-Allowed-Count", strconv.Itoa(globalBucket.Requests))
		w.Header().Set("X-Ratelimit-Bucket-Reqs-Allowed-Count", strconv.Itoa(reqBucket.Requests))

		w.Header().Set("X-Ratelimit-Bucket-Global-Reqs-Allowed-Second", strconv.FormatFloat(globalBucket.Time.Seconds(), 'g', -1, 64))
		w.Header().Set("X-Ratelimit-Bucket-Reqs-Allowed-Second", strconv.FormatFloat(reqBucket.Time.Seconds(), 'g', -1, 64))

		// Get ratelimit from redis
		var id string

		auth := r.Header.Get("Authorization")

		if auth != "" {
			if strings.HasPrefix(auth, "User ") {
				idCheck := authCheck(auth, false)

				if idCheck == nil {
					// Bot does not exist, return
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("{\"message\":\"Invalid API token\",\"error\":true}"))
					return
				}

				id = *idCheck
			} else {
				idCheck := authCheck(auth, true)

				if idCheck == nil {
					// Bot does not exist, return
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("{\"message\":\"Invalid API token\",\"error\":true}"))
					return
				}

				id = *idCheck
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

		fn(w, r)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.Header.Get("Origin"), "infinitybots.gg") || strings.HasPrefix(r.Header.Get("Origin"), "localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, User-Auth, Bot-Auth")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")

		if r.Method == "OPTIONS" {
			w.Write([]byte{})
			return
		}

		if r.Header.Get("User-Auth") != "" {
			if strings.HasPrefix(r.Header.Get("User-Auth"), "User ") {
				r.Header.Set("Authorization", r.Header.Get("User-Auth"))
			} else {
				r.Header.Set("Authorization", "User "+r.Header.Get("User-Auth"))
			}
		} else if r.Header.Get("Bot-Auth") != "" {
			r.Header.Set("Authorization", "Bot "+r.Header.Get("Bot-Auth"))
		}

		w.Header().Set("Content-Type", "application/json")

		next.ServeHTTP(w, r)
	})
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

	docs.AddSecuritySchema("User", "User-Auth", "Requires a user token. Usually must be prefixed with `User `. Note that both ``User-Auth`` and ``Authorization`` headers are supported")
	docs.AddSecuritySchema("Bot", "Bot-Auth", "Requires a bot token. Can be optionally prefixed. Note that both ``Bot-Auth`` and ``Authorization`` headers are supported")

	ctx = context.Background()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(corsMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(30 * time.Second))

	// Init redisCache
	rOptions, err := redis.ParseURL("redis://localhost:6379/12")

	if err != nil {
		panic(err)
	}

	redisCache = redis.NewClient(rOptions)

	rOptionsNext, err := redis.ParseURL("redis://localhost:6379/1")

	if err != nil {
		panic(err)
	}

	iblCache = redis.NewClient(rOptionsNext)

	pool, err = pgxpool.Connect(ctx, pgConn)

	if err != nil {
		panic(err)
	}

	backupPool, err = pgxpool.Connect(ctx, backupConn)

	if err != nil {
		panic(err)
	}

	// Create base payloads before startup
	// Index
	var helloWorldB Hello

	helloWorldB.Message = "Hello world from IBL API v6!"
	helloWorldB.Docs = docsSite
	helloWorldB.OurSite = mainSite
	helloWorldB.Status = statusPage

	helloWorld, err := json.Marshal(helloWorldB)

	if err != nil {
		panic(err)
	}

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

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/",
		OpId:        "ping",
		Summary:     "Ping Server",
		Description: "This is a simple ping endpoint to check if the API is online. It will return a simple JSON object with a message, docs link, our site link and status page link.",
		Tags:        []string{"System"},
		Resp:        helloWorldB,
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(helloWorld))
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/_duser/{id}/clear",
		OpId:        "clear_duser",
		Summary:     "Clear Discord User Cache",
		Description: "This endpoint will clear the cache for a specific discord user. This is useful if you the user's data has changes",
		Tags:        []string{"System"},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The ID of the user to clear the cache for",
				In:          "path",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	})
	r.Get("/_duser/{id}/clear", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		redisCache.Del(ctx, "uobj:"+id)
		w.Write([]byte(success))
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "announcements",
		OpId:        "announcements",
		Summary:     "Get Announcements",
		Description: "This endpoint will return a list of announcements. User authentication is optional and using it will show user targetted announcements.",
		Tags:        []string{"System"},
		Resp:        []types.AnnouncementList{},
		AuthType:    []string{"User"},
	})
	r.Get("/announcements", rateLimitWrap(30, 1*time.Minute, "gannounce", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(ctx, "SELECT "+announcementColsStr+" FROM announcements ORDER BY id DESC")

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		var announcements []types.Announcement

		err = pgxscan.ScanAll(&announcements, rows)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		// Auth header check
		auth := r.Header.Get("Authorization")

		var target types.UserID

		if auth != "" {
			targetId := authCheck(auth, false)

			if targetId != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}

			target = types.UserID{UserID: *targetId}
		} else {
			target = types.UserID{}
		}

		annList := []types.Announcement{}

		for _, announcement := range announcements {
			if announcement.Status == "private" {
				// Staff only
				continue
			}

			if announcement.Targetted {
				// Check auth header
				if target.UserID != announcement.Target.String {
					continue
				}
			}

			annList = append(annList, announcement)
		}

		annListObj := types.AnnouncementList{
			Announcements: annList,
		}

		bytes, err := json.Marshal(annListObj)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	}))

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/_duser/{id}",
		OpId:        "get_duser",
		Summary:     "Get Discord User",
		Description: "This endpoint will return a discord user object. This is useful for getting a user's avatar, username or discriminator etc.",
		Tags:        []string{"System"},
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The user's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.DiscordUser{},
	})
	r.Get("/_duser/{id}", func(w http.ResponseWriter, r *http.Request) {
		var id = chi.URLParam(r, "id")

		user, err := utils.GetDiscordUser(metro, redisCache, ctx, id)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		bytes, err := json.Marshal(user)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/openapi",
		OpId:        "openapi",
		Summary:     "Get OpenAPI Spec",
		Description: "This endpoint will return the OpenAPI spec for the API. This is useful for generating clients for the API.",
		Tags:        []string{"System"},
		Resp:        types.OpenAPI{},
	})
	r.Get("/openapi", func(w http.ResponseWriter, r *http.Request) {
		w.Write(openapi)
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(docsHTML))
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/bots/all",
		OpId:        "get_all_bots",
		Summary:     "Get All Bots",
		Description: "Gets all bots on the list.",
		Tags:        []string{"Bots"},
		Resp:        types.AllBots{},
	})
	r.Get("/bots/all", rateLimitWrap(5, 2*time.Second, "allbots", func(w http.ResponseWriter, r *http.Request) {
		const perPage = 10

		page := r.URL.Query().Get("page")

		if page == "" {
			page = "1"
		}

		pageNum, err := strconv.ParseUint(page, 10, 32)

		if err != nil {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		limit := perPage
		offset := (pageNum - 1) * perPage

		rows, err := pool.Query(ctx, "SELECT "+botsColsStr+" FROM bots ORDER BY date DESC LIMIT $1 OFFSET $2", limit, offset)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var bots []*types.Bot

		err = pgxscan.ScanAll(&bots, rows)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var previous strings.Builder

		// More optimized string concat
		previous.WriteString(os.Getenv("SITE_URL"))
		previous.WriteString("/bots/all?page=")
		previous.WriteString(strconv.FormatUint(pageNum-1, 10))

		if pageNum-1 < 1 || pageNum == 0 {
			previous.Reset()
		}

		var count uint64

		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots").Scan(&count)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
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
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)

	}))

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/packs/all",
		OpId:        "get_all_packs",
		Summary:     "Get All Packs",
		Description: "Gets all packs on the list.",
		Tags:        []string{"Bot Packs"},
		Resp:        types.AllPacks{},
	})
	r.Get("/packs/all", rateLimitWrap(5, 2*time.Second, "allpacks", func(w http.ResponseWriter, r *http.Request) {
		const perPage = 12

		page := r.URL.Query().Get("page")

		if page == "" {
			page = "1"
		}

		pageNum, err := strconv.ParseUint(page, 10, 32)

		if err != nil {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Check cache, this is how we can avoid hefty ratelimits
		cache := redisCache.Get(ctx, "pca-"+strconv.FormatUint(pageNum, 10)).Val()
		if cache != "" {
			w.Header().Add("X-Popplio-Cached", "true")
			w.Write([]byte(cache))
			return
		}

		limit := perPage
		offset := (pageNum - 1) * perPage

		rows, err := pool.Query(ctx, "SELECT "+packsColsString+" FROM packs ORDER BY date DESC LIMIT $1 OFFSET $2", limit, offset)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var packs []*types.BotPack

		err = pgxscan.ScanAll(&packs, rows)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		for _, pack := range packs {
			err := utils.ResolveBotPack(ctx, pool, pack, metro, redisCache)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}
		}

		var previous strings.Builder

		// More optimized string concat
		previous.WriteString(os.Getenv("SITE_URL"))
		previous.WriteString("/packs/all?page=")
		previous.WriteString(strconv.FormatUint(pageNum-1, 10))

		if pageNum-1 < 1 || pageNum == 0 {
			previous.Reset()
		}

		var count uint64

		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM packs").Scan(&count)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var next strings.Builder

		next.WriteString(os.Getenv("SITE_URL"))
		next.WriteString("/packs/all?page=")
		next.WriteString(strconv.FormatUint(pageNum+1, 10))

		if float64(pageNum+1) > math.Ceil(float64(count)/perPage) {
			next.Reset()
		}

		data := types.AllPacks{
			Count:    count,
			Results:  packs,
			PerPage:  perPage,
			Previous: previous.String(),
			Next:     next.String(),
		}

		bytes, err := json.Marshal(data)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		redisCache.Set(ctx, "pca-"+strconv.FormatUint(pageNum, 10), bytes, 2*time.Minute)

		w.Write(bytes)
	}))

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/packs/{id}",
		OpId:        "get_packs",
		Summary:     "Get Packs",
		Description: "Gets a pack on the list based on either URL or Name.",
		Tags:        []string{"Bot Packs"},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The ID of the pack.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.BotPack{},
	})
	r.Get("/packs/{id}", rateLimitWrap(10, 3*time.Minute, "gpack", func(w http.ResponseWriter, r *http.Request) {
		var id = chi.URLParam(r, "id")

		if id == "" {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		var pack types.BotPack

		row, err := pool.Query(ctx, "SELECT "+packsColsString+" FROM packs WHERE url = $1 OR name = $1", id)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		err = pgxscan.ScanOne(&pack, row)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		err = utils.ResolveBotPack(ctx, pool, &pack, metro, redisCache)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		bytes, err := json.Marshal(pack)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	}))

	docs.Route(&docs.Doc{
		Method:  "POST",
		Path:    "/bots/stats",
		OpId:    "post_stats",
		Summary: "Post Bot Stats",
		Description: `
This endpoint can be used to post the stats of a bot.

The variation` + backTick + `/bots/{bot_id}/stats` + backTick + ` can also be used to post the stats of a bot. **Note that only the token is checked, not the bot ID at this time**

**Example:**

` + backTick + backTick + backTick + `py
import requests

req = requests.post(f"{API_URL}/bots/stats", json={"servers": 4000, "shards": 2}, headers={"Authorization": "{TOKEN}"})

print(req.json())
` + backTick + backTick + backTick + "\n\n",
		Tags:     []string{"Bots"},
		Req:      types.BotStatsDocs{},
		Resp:     types.ApiError{},
		AuthType: []string{"Bot"},
	})
	statsFn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "DELETE" {
			apiDefaultReturn(http.StatusMethodNotAllowed, w, r)
			return
		}

		if r.Body == nil {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		var id *string

		// Check token
		if r.Header.Get("Authorization") == "" {
			apiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			id = authCheck(r.Header.Get("Authorization"), true)

			if id == nil {
				log.Error(err)
				apiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		defer r.Body.Close()

		var payload types.BotStats

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
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
				apiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			var countAny any = count

			payload.Count = &countAny
		}

		servers, shards, users := payload.GetStats()

		if servers > 0 {
			_, err = pool.Exec(ctx, "UPDATE bots SET servers = $1 WHERE bot_id = $2", servers, id)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}
		}

		if shards > 0 {
			_, err = pool.Exec(ctx, "UPDATE bots SET shards = $1 WHERE bot_id = $2", shards, id)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}
		}

		if users > 0 {
			_, err = pool.Exec(ctx, "UPDATE bots SET users = $1 WHERE bot_id = $2", users, id)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}
		}

		// Get name and vanity, delete from cache
		var name, vanity string

		pool.QueryRow(ctx, "SELECT name, vanity FROM bots WHERE bot_id = $1", id).Scan(&name, &vanity)

		// Delete from cache
		redisCache.Del(ctx, "bc-"+name)
		redisCache.Del(ctx, "bc-"+vanity)
		redisCache.Del(ctx, "bc-"+*id)

		// Clear ibl next cache
		iblCache.Del(ctx, *id+"_data")
		iblCache.Del(ctx, name+"_data")
		iblCache.Del(ctx, vanity+"_data")

		w.Write([]byte(success))
	}

	r.HandleFunc("/bots/stats", rateLimitWrap(10, 1*time.Minute, "stats", statsFn))

	// Intentionally not documented, variant endpoint
	r.HandleFunc("/bots/{id}/stats", rateLimitWrap(10, 1*time.Minute, "stats", statsFn))

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/list/index",
		OpId:        "get_list_index",
		Summary:     "Get List Index",
		Description: "Gets the index of the list. Note that this endpoint does not resolve the owner or the bots of a pack and will only give the `owner_id` and the `bot_ids` for performance purposes",
		Tags:        []string{"System"},
		Resp:        types.ListIndex{},
	})
	r.Get("/list/index", rateLimitWrap(5, 1*time.Minute, "glstats", func(w http.ResponseWriter, r *http.Request) {
		// Check cache, this is how we can avoid hefty ratelimits
		cache := redisCache.Get(ctx, "indexcache").Val()
		if cache != "" {
			w.Header().Add("X-Popplio-Cached", "true")
			w.Write([]byte(cache))
			return
		}

		listIndex := types.ListIndex{}

		certRow, err := pool.Query(ctx, "SELECT "+indexBotsColStr+" FROM bots WHERE certified = true AND type = 'approved' ORDER BY votes DESC LIMIT 9")
		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		certDat := []types.IndexBot{}
		err = pgxscan.ScanAll(&certDat, certRow)
		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}
		listIndex.Certified = certDat

		mostViewedRow, err := pool.Query(ctx, "SELECT "+indexBotsColStr+" FROM bots WHERE type = 'approved' ORDER BY clicks DESC LIMIT 9")
		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}
		mostViewedDat := []types.IndexBot{}
		err = pgxscan.ScanAll(&mostViewedDat, mostViewedRow)
		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}
		listIndex.MostViewed = mostViewedDat

		recentlyAddedRow, err := pool.Query(ctx, "SELECT "+indexBotsColStr+" FROM bots WHERE type = 'approved' ORDER BY date DESC LIMIT 9")
		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}
		recentlyAddedDat := []types.IndexBot{}
		err = pgxscan.ScanAll(&recentlyAddedDat, recentlyAddedRow)
		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}
		listIndex.RecentlyAdded = recentlyAddedDat

		topVotedRow, err := pool.Query(ctx, "SELECT "+indexBotsColStr+" FROM bots WHERE type = 'approved' ORDER BY votes DESC LIMIT 9")
		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}
		topVotedDat := []types.IndexBot{}
		err = pgxscan.ScanAll(&topVotedDat, topVotedRow)
		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}
		listIndex.TopVoted = topVotedDat

		rows, err := pool.Query(ctx, "SELECT "+packsColsString+" FROM packs ORDER BY date DESC")

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var packs []*types.BotPack

		err = pgxscan.ScanAll(&packs, rows)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		listIndex.Packs = packs

		bytes, err := json.Marshal(listIndex)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		redisCache.Set(ctx, "indexcache", string(bytes), 10*time.Minute)
		w.Write(bytes)
	}))

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/list/stats",
		OpId:        "get_list_stats",
		Summary:     "Get List Statistics",
		Description: "Gets the statistics of the list",
		Tags:        []string{"System"},
		Resp: types.ListStats{
			Bots: []types.ListStatsBot{},
		},
	})
	r.Get("/list/stats", rateLimitWrap(5, 1*time.Minute, "glstats", func(w http.ResponseWriter, r *http.Request) {
		listStats := types.ListStats{}

		bots, err := pool.Query(ctx, "SELECT bot_id, name, short, type, owner, additional_owners, avatar FROM bots")

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		defer bots.Close()

		for bots.Next() {
			var botId string
			var name string
			var short string
			var typeStr string
			var owner string
			var additionalOwners []string
			var avatar string

			err := bots.Scan(&botId, &name, &short, &typeStr, &owner, &additionalOwners, &avatar)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			listStats.Bots = append(listStats.Bots, types.ListStatsBot{
				BotID:              botId,
				Name:               name,
				Short:              short,
				Type:               typeStr,
				AvatarDB:           avatar,
				MainOwnerID:        owner,
				AdditionalOwnerIDS: additionalOwners,
			})
		}

		bytes, err := json.Marshal(listStats)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	}))

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{uid}/bots/{bid}/votes",
		OpId:        "get_user_votes",
		Summary:     "Get User Votes",
		Description: "Gets the users votes. **Requires authentication**",
		Tags:        []string{"Votes"},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserVote{
			Timestamps: []int64{},
			VoteTime:   12,
			HasVoted:   true,
		},
		AuthType: []string{"User", "Bot"},
	})

	docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/users/{uid}/bots/{bid}/votes",
		OpId:        "post_user_votes",
		Summary:     "Post User Votes",
		Description: "Posts a users votes. **For internal use only**",
		Tags:        []string{"Votes"},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "The bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp:     types.ApiError{},
		AuthType: []string{"User"},
	})
	// TODO: Document POST as well and seperate the two funcs
	r.HandleFunc("/users/{uid}/bots/{bid}/votes", rateLimitWrap(5, 1*time.Minute, "gvotes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "PUT" {
			apiDefaultReturn(http.StatusMethodNotAllowed, w, r)
			return
		}

		var vars = map[string]string{
			"uid": chi.URLParam(r, "uid"),
			"bid": chi.URLParam(r, "bid"),
		}

		userAuth := strings.HasPrefix(r.Header.Get("Authorization"), "User ")

		var botId pgtype.Text
		var botType pgtype.Text

		if r.Header.Get("Authorization") == "" {
			apiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			if userAuth {
				uid := authCheck(r.Header.Get("Authorization"), false)

				if uid == nil || *uid != vars["uid"] {
					apiDefaultReturn(http.StatusUnauthorized, w, r)
					return
				}

				var voteBannedState bool

				err := pool.QueryRow(ctx, "SELECT vote_banned FROM users WHERE user_id = $1", uid).Scan(&voteBannedState)

				if err != nil {
					log.Error(err)
					apiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}

				if voteBannedState && r.Method == "PUT" {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(voteBanned))
					return
				}

				var voteBannedBotsState bool

				err = pool.QueryRow(ctx, "SELECT bot_id, type, vote_banned FROM bots WHERE (bot_id = $1 OR vanity = $1 OR name = $1)", vars["bid"]).Scan(&botId, &botType, &voteBannedBotsState)

				if err != nil {
					log.Error(err)
					apiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}

				if voteBannedBotsState && r.Method == "PUT" {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(voteBanned))
					return
				}

				vars["bid"] = botId.String
			} else {
				err = pool.QueryRow(ctx, "SELECT bot_id, type FROM bots WHERE (vanity = $1 OR bot_id = $1 OR name = $1)", vars["bid"]).Scan(&botId, &botType)

				if err != nil || botId.Status != pgtype.Present || botType.Status != pgtype.Present {
					log.Error(err)
					apiDefaultReturn(http.StatusNotFound, w, r)
					return
				}

				vars["bid"] = botId.String

				id := authCheck(r.Header.Get("Authorization"), true)

				if id == nil || *id != vars["bid"] {
					apiDefaultReturn(http.StatusUnauthorized, w, r)
					return
				}
			}
		}

		if botType.String != "approved" && r.Method == "PUT" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(notApproved))
			return
		}

		if !userAuth && r.Method == "PUT" {
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		voteParsed, err := utils.GetVoteData(ctx, pool, vars["uid"], vars["bid"])

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		if r.Method == "GET" {
			bytes, err := json.Marshal(voteParsed)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
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
					apiDefaultReturn(http.StatusInternalServerError, w, r)
					return
				}

				w.WriteHeader(http.StatusBadRequest)
				w.Write(bytes)
				return
			}

			// Record new vote
			var itag pgtype.UUID
			err := pool.QueryRow(ctx, "INSERT INTO votes (user_id, bot_id) VALUES ($1, $2) RETURNING itag", vars["uid"], vars["bid"]).Scan(&itag)

			if err != nil {
				// Revert vote
				_, err := pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var oldVotes pgtype.Int4

			err = pool.QueryRow(ctx, "SELECT votes FROM bots WHERE bot_id = $1", vars["bid"]).Scan(&oldVotes)

			if err != nil {
				// Revert vote
				_, err := pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)

				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var incr = 1
			var votes = oldVotes.Int

			if utils.GetDoubleVote() {
				incr = 2
				votes += 2
			} else {
				votes++
			}

			_, err = pool.Exec(ctx, "UPDATE bots SET votes = votes + $1 WHERE bot_id = $2", incr, vars["bid"])

			if err != nil {
				// Revert vote
				_, err := pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)

				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			userObj, err := utils.GetDiscordUser(metro, redisCache, ctx, vars["uid"])

			if err != nil {
				// Revert vote
				_, err := pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)

				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			botObj, err := utils.GetDiscordUser(metro, redisCache, ctx, vars["bid"])

			if err != nil {
				// Revert vote
				_, err := pool.Exec(ctx, "DELETE FROM votes WHERE itag = $1", itag)

				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			channel := os.Getenv("VOTE_LOGS_CHANNEL")

			metro.ChannelMessageSendComplex(channel, &discordgo.MessageSend{
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
								Value:  strconv.Itoa(int(votes)),
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
			})

			// Send webhook in a goroutine refunding the vote if it failed
			go func() {
				err = sendWebhook(types.WebhookPost{
					BotID:  vars["bid"],
					UserID: vars["uid"],
					Votes:  int(votes),
				})

				if err != nil {
					pool.Exec(
						ctx,
						"INSERT INTO notifications (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
						vars["uid"],
						"https://infinitybots.gg/bots/"+vars["bid"],
						"Whoa there! We've failed to notify this bot about this vote. The error was: "+err.Error()+".",
						"error")
				} else {
					pool.Exec(
						ctx,
						"INSERT INTO notifications (user_id, url, message, type) VALUES ($1, $2, $3, $4)",
						vars["uid"],
						"https://infinitybots.gg/bots/"+vars["bid"],
						"Successfully voted for bot with ID of "+vars["bid"],
						"info",
					)
				}
			}()

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(success))
		}
	}))

	// For compatibility with old API
	r.HandleFunc("/votes/{bot_id}/{user_id}", rateLimitWrap(10, 1*time.Minute, "deprecated-gvotes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			apiDefaultReturn(http.StatusMethodNotAllowed, w, r)
			return
		}

		var botId = chi.URLParam(r, "bot_id")
		var userId = chi.URLParam(r, "user_id")

		if r.Header.Get("Authorization") == "" {
			apiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			id := authCheck(r.Header.Get("Authorization"), true)

			if id == nil || *id != botId {
				apiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}

			// To try and push users into new API, vote ban and approved check on GET is enforced on the old API
			var voteBannedState bool

			err := pool.QueryRow(ctx, "SELECT vote_banned FROM bots WHERE bot_id = $1", id).Scan(&voteBannedState)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		var botType pgtype.Text

		pool.QueryRow(ctx, "SELECT type FROM bots WHERE bot_id = $1", botId).Scan(&botType)

		if botType.String != "approved" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(notApproved))
			return
		}

		voteParsed, err := utils.GetVoteData(ctx, pool, userId, botId)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var compatData = types.UserVoteCompat{
			HasVoted: voteParsed.HasVoted,
		}

		bytes, err := json.Marshal(compatData)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	}))

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/voteinfo",
		OpId:        "get_iote_info",
		Summary:     "Get Vote Info",
		Description: "Returns basic voting info such as if its a weekend double vote.",
		Resp:        types.VoteInfo{Weekend: true},
		Tags:        []string{"Votes"},
	})
	r.Get("/voteinfo", func(w http.ResponseWriter, r *http.Request) {
		var payload = types.VoteInfo{
			Weekend: utils.GetDoubleVote(),
		}

		b, err := json.Marshal(payload)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(b)
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/bots/{id}/seo",
		OpId:        "get_bot_seo",
		Summary:     "Get Bot SEO Info",
		Description: "Gets the minimal SEO information about a bot for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEOBot{},
		Tags:        []string{"Bots"},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	})
	r.Get("/bots/{id}/seo", rateLimitWrap(15, 1*time.Minute, "gbot", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "id")

		if name == "" {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		var botId string
		var short string
		err := pool.QueryRow(ctx, "SELECT bot_id, short FROM bots WHERE (bot_id = $1 OR vanity = $1 OR name = $1)", name).Scan(&botId, &short)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		bot, err := utils.GetDiscordUser(metro, redisCache, ctx, botId)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		bytes, err := json.Marshal(types.SEOBot{
			User:  bot,
			Short: short,
		})

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	}))

	docs.Route(&docs.Doc{
		Method:  "GET",
		Path:    "/bots/{id}",
		OpId:    "get_bot",
		Summary: "Get Bot",
		Description: `
Gets a bot by id or name

**Some things to note:**

-` + backTick + backTick + `external_source` + backTick + backTick + ` shows the source of where a bot came from (Metro Reviews etc etc.). If this is set to ` + backTick + backTick + `metro` + backTick + backTick + `, then ` + backTick + backTick + `list_source` + backTick + backTick + ` will be set to the metro list ID where it came from` + `
	`,
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
		Tags: []string{"Bots"},
	})
	getBotsFn := rateLimitWrap(10, 1*time.Minute, "gbot", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "id")

		if name == "" {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Check cache, this is how we can avoid hefty ratelimits
		cache := redisCache.Get(ctx, "bc-"+name).Val()
		if cache != "" {
			w.Header().Add("X-Popplio-Cached", "true")
			w.Write([]byte(cache))
			return
		}

		var bot types.Bot

		var err error

		row, err := pool.Query(ctx, "SELECT "+botsColsStr+" FROM bots WHERE (bot_id = $1 OR vanity = $1 OR name = $1)", name)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		err = pgxscan.ScanOne(&bot, row)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		err = utils.ParseBot(ctx, pool, &bot, metro, redisCache)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		var uniqueClicks int64
		err = pool.QueryRow(ctx, "SELECT cardinality(unique_clicks) AS unique_clicks FROM bots WHERE bot_id = $1", bot.BotID).Scan(&uniqueClicks)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		bot.UniqueClicks = uniqueClicks

		/* Removing or modifying fields directly in API is very dangerous as scrapers will
		 * just ignore owner checks anyways or cross-reference via another list. Also we
		 * want to respect the permissions of the owner if they're the one giving permission,
		 * blocking IPs is a better idea to this
		 */

		bytes, err := json.Marshal(bot)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		redisCache.Set(ctx, "bc-"+name, string(bytes), time.Minute*3)

		w.Write(bytes)
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{id}",
		OpId:        "get_user",
		Summary:     "Get User",
		Description: "Gets a user by id or username",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.User{},
		Tags: []string{"User"},
	})
	r.Get("/users/{id}", rateLimitWrap(10, 3*time.Minute, "guser", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "id")

		if name == "" {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Check cache, this is how we can avoid hefty ratelimits
		cache := redisCache.Get(ctx, "uc-"+name).Val()
		if cache != "" {
			w.Header().Add("X-Popplio-Cached", "true")
			w.Write([]byte(cache))
			return
		}

		var user types.User

		var err error

		row, err := pool.Query(ctx, "SELECT "+usersColsStr+" FROM users WHERE user_id = $1 OR username = $1", name)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		err = pgxscan.ScanOne(&user, row)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		err = utils.ParseUser(ctx, pool, &user, metro, redisCache)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		/* Removing or modifying fields directly in API is very dangerous as scrapers will
		 * just ignore owner checks anyways or cross-reference via another list. Also we
		 * want to respect the permissions of the owner if they're the one giving permission,
		 * blocking IPs is a better idea to this
		 */

		bytes, err := json.Marshal(user)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		redisCache.Set(ctx, "uc-"+name, string(bytes), time.Minute*3)

		w.Write(bytes)
	}))

	r.Get("/bots/{id}", getBotsFn)
	r.Get("/bot/{id}", getBotsFn)

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/bots/{id}/reviews",
		OpId:        "get_bot_reviews",
		Summary:     "Get Bot Reviews",
		Description: "Gets the reviews of a bot by its ID, name or vanity",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ReviewList{},
		Tags: []string{"Bots"},
	})
	r.Get("/bots/{id}/reviews", rateLimitWrap(10, 1*time.Minute, "greview", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(ctx, "SELECT "+reviewColsStr+" FROM reviews WHERE (bot_id = $1 OR vanity = $1 OR name = $1)", chi.URLParam(r, "id"))

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusNotFound, w, r)
			return
		}

		var reviews []types.Review = []types.Review{}

		err = pgxscan.ScanAll(&reviews, rows)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var allReviews types.ReviewList = types.ReviewList{
			Reviews: reviews,
		}

		bytes, err := json.Marshal(allReviews)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	}))

	// TODO: Document this once its stable
	r.HandleFunc("/login/{act}", oauthFn)
	r.HandleFunc("/cosmog", performAct)
	r.HandleFunc("/cosmog/tasks/{tid}.arceus", getTask)
	r.HandleFunc("/cosmog/tasks/{tid}", taskFn)

	docs.Route(&docs.Doc{
		Method:      "POST",
		Path:        "/webhook-test",
		OpId:        "webhook_test",
		Summary:     "Test Webhook",
		Description: "Sends a test webhook to allow testing your vote system. **All fields are mandatory for this endpoint**",
		Req:         types.WebhookPost{},
		Resp:        types.ApiError{},
		Tags:        []string{"Bots"},
	})
	r.Post("/webhook-test", rateLimitWrap(7, 3*time.Minute, "webtest", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload types.WebhookPost

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		err = json.Unmarshal(bodyBytes, &payload)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		if utils.IsNone(payload.URL) && utils.IsNone(payload.URL2) {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		payload.Test = true // Always true

		var err1 error

		if !utils.IsNone(payload.URL) {
			err1 = sendWebhook(payload)
		}

		var err2 error

		if !utils.IsNone(payload.URL2) {
			payload.URL = payload.URL2 // Test second enpdoint if it's not empty
			err2 = sendWebhook(payload)
		}

		var errD = types.ApiError{}

		if err1 != nil {
			log.Error(err1)

			errD.Message = err1.Error()
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
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write(bytes)
	}))

	// Internal APIs

	r.Patch("/_protozoa/profile/{id}", rateLimitWrap(7, 1*time.Minute, "profile_update", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		// Fetch auth from postgresdb
		if r.Header.Get("Authorization") == "" {
			apiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			authId := authCheck(r.Header.Get("Authorization"), false)

			if authId == nil || *authId != id {
				log.Error(err)
				apiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		// Fetch profile update from body
		var profile types.ProfileUpdate

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		err = json.Unmarshal(bodyBytes, &profile)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		if profile.About != "" {
			// Update about
			_, err = pool.Exec(ctx, "UPDATE users SET about = $1 WHERE user_id = $2", profile.About, id)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}
		}

		redisCache.Del(ctx, "uc-"+id)
	}))

	r.Get("/_protozoa/notifications/info", rateLimitWrap(10, 1*time.Minute, "notif_info", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"public_key": os.Getenv("VAPID_PUBLIC_KEY"),
		}

		bytes, err := json.Marshal(data)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	}))

	r.HandleFunc("/_protozoa/notifications/{id}", rateLimitWrap(40, 1*time.Minute, "get_notifs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "DELETE" {
			apiDefaultReturn(http.StatusMethodNotAllowed, w, r)
			return
		}

		var id = chi.URLParam(r, "id")

		if id == "" {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Fetch auth from postgresdb
		if r.Header.Get("Authorization") == "" {
			apiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			authId := authCheck(r.Header.Get("Authorization"), false)

			if authId == nil || *authId != id {
				log.Error(err)
				apiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		if r.Method == "GET" {
			var subscription []types.NotifGet

			var subscriptionDb []struct {
				Endpoint  string    `db:"endpoint"`
				NotifID   string    `db:"notif_id"`
				CreatedAt time.Time `db:"created_at"`
				UA        string    `db:"ua"`
			}

			rows, err := pool.Query(ctx, "SELECT endpoint, notif_id, created_at, ua FROM poppypaw WHERE id = $1", id)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			err = pgxscan.ScanAll(&subscriptionDb, rows)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			if len(subscriptionDb) == 0 {
				apiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			for _, sub := range subscriptionDb {
				uaD := ua.Parse(sub.UA)
				fmt.Println("Parsing UA:", sub.UA, uaD)

				binfo := types.NotifBrowserInfo{
					OS:         uaD.OS,
					Browser:    uaD.Name,
					BrowserVer: uaD.Version,
					Mobile:     uaD.Mobile,
				}

				subscription = append(subscription, types.NotifGet{
					Endpoint:    sub.Endpoint,
					NotifID:     sub.NotifID,
					CreatedAt:   sub.CreatedAt,
					BrowserInfo: binfo,
				})
			}

			bytes, err := json.Marshal(subscription)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		} else {
			// Delete the notif
			if r.URL.Query().Get("notif_id") == "" {
				apiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			_, err := pool.Exec(ctx, "DELETE FROM poppypaw WHERE id = $1 AND notif_id = $2", id, r.URL.Query().Get("notif_id"))

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.WriteHeader(http.StatusOK)
		}
	}))

	r.Post("/_protozoa/notifications/{id}/sub", rateLimitWrap(10, 1*time.Minute, "notif_info", func(w http.ResponseWriter, r *http.Request) {
		var subscription struct {
			Auth     string `json:"auth"`
			P256dh   string `json:"p256dh"`
			Endpoint string `json:"endpoint"`
		}

		var id = chi.URLParam(r, "id")

		if id == "" {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		defer r.Body.Close()

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		err = json.Unmarshal(bodyBytes, &subscription)

		if err != nil {
			log.Error(err)
			apiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		if subscription.Auth == "" || subscription.P256dh == "" {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Fetch auth from postgresdb
		if r.Header.Get("Authorization") == "" {
			apiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			authId := authCheck(r.Header.Get("Authorization"), false)

			if authId == nil || *authId != id {
				log.Error(err)
				apiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		// Store new subscription

		notifId := utils.RandString(512)

		ua := r.UserAgent()

		if ua == "" {
			ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"
		}

		pool.Exec(ctx, "DELETE FROM poppypaw WHERE id = $1 AND endpoint = $2", id, subscription.Endpoint)

		pool.Exec(
			ctx,
			"INSERT INTO poppypaw (id, notif_id, auth, p256dh,  endpoint, ua) VALUES ($1, $2, $3, $4, $5, $6)",
			id,
			notifId,
			subscription.Auth,
			subscription.P256dh,
			subscription.Endpoint,
			ua,
		)

		// Fan out test notification
		notifChannel <- types.Notification{
			NotifID: notifId,
			Message: []byte(testNotif),
		}

		w.Write([]byte(success))
	}))

	r.HandleFunc("/_protozoa/reminders/{id}", rateLimitWrap(40, 1*time.Minute, "greminder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" && r.Method != "GET" && r.Method != "DELETE" {
			apiDefaultReturn(http.StatusMethodNotAllowed, w, r)
			return
		}

		var id = chi.URLParam(r, "id")

		if id == "" {
			apiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Fetch auth from postgresdb
		if r.Header.Get("Authorization") == "" {
			apiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			authId := authCheck(r.Header.Get("Authorization"), false)

			if authId == nil || *authId != id {
				log.Error(err)
				apiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		if r.Method == "GET" {
			// Fetch reminder from postgresdb
			rows, err := pool.Query(ctx, "SELECT "+silverpeltColsStr+" FROM silverpelt WHERE user_id = $1", id)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var reminders []types.Reminder

			pgxscan.ScanAll(&reminders, rows)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			if len(reminders) == 0 {
				apiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			for i, reminder := range reminders {
				// Try resolving the bot from discord API
				var resolvedBot types.ResolvedReminderBot
				bot, err := utils.GetDiscordUser(metro, redisCache, ctx, reminder.BotID)

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

				reminders[i].ResolvedBot = resolvedBot
			}

			bytes, err := json.Marshal(reminders)

			if err != nil {
				log.Error(err)
				apiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		} else {
			// Add subscription to collection
			var botId pgtype.Text

			err = pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE (vanity = $1 OR bot_id = $1 OR name = $1)", r.URL.Query().Get("bot_id")).Scan(&botId)

			if err != nil || botId.Status != pgtype.Present || botId.String == "" {
				log.Error("Error adding reminder: ", err)
				apiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			// Delete old
			pool.Exec(ctx, "DELETE FROM silverpelt WHERE user_id = $1 AND bot_id = $2", id, botId.String)

			// Insert new
			if r.Method == "PUT" {
				_, err := pool.Exec(ctx, "INSERT INTO silverpelt (user_id, bot_id) VALUES ($1, $2)", id, botId.String)

				if err != nil {
					log.Error("Error adding reminder: ", err)
					apiDefaultReturn(http.StatusNotFound, w, r)
					return
				}
			}

			w.Write([]byte(success))
		}
	}))

	// Load openapi here to avoid large marshalling in every request
	openapi, err = json.Marshal(docs.GetSchema())

	if err != nil {
		panic(err)
	}

	adp := DummyAdapter{}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		apiDefaultReturn(http.StatusNotFound, w, r)
	})

	createBucketMods()

	integrase.Prepare(adp, chiWrap{Router: r})

	http.ListenAndServe(":8081", r)
}
