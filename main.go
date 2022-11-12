package main

import (
	"context"
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"popplio/docs"
	"popplio/migrations"
	"popplio/routes/announcements"
	"popplio/routes/auth"
	"popplio/routes/bots"
	"popplio/routes/duser"
	"popplio/routes/list"
	"popplio/routes/packs"
	"popplio/routes/transcripts"
	"popplio/routes/users"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"popplio/webhooks"

	integrase "github.com/MetroReviews/metro-integrase/lib"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgtype"
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

	testNotif = "{\"message\":\"Test notification!\", \"title\":\"Test notification!\",\"icon\":\"https://i.imgur.com/GRo0Zug.png\",\"error\":false}"
	backTick  = "`"
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
	ctx       context.Context
	migration bool

	docsJs  string
	openapi []byte

	// Default global ratelimit handler
	defaultGlobalBucket = moderatedBucket{BucketName: "global", Requests: 500, Time: 2 * time.Minute}

	silverpeltCols = utils.GetCols(types.Reminder{})

	silverpeltColsStr = strings.Join(silverpeltCols, ",")
)

func bucketHandle(bucket moderatedBucket, id string, w http.ResponseWriter, r *http.Request) bool {
	rlKey := "rl:" + id + "-" + bucket.BucketName

	v := state.Redis.Get(r.Context(), rlKey).Val()

	if v == "" {
		v = "0"

		err := state.Redis.Set(ctx, rlKey, "0", bucket.Time).Err()

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return false
		}
	}

	err := state.Redis.Incr(ctx, rlKey).Err()

	if err != nil {
		log.Error(err)
		utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
		return false
	}

	vInt, err := strconv.Atoi(v)

	if err != nil {
		log.Error(err)
		utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
		return false
	}

	if vInt < 0 {
		state.Redis.Expire(ctx, rlKey, 1*time.Second)
		vInt = 0
	}

	if vInt > bucket.Requests {
		retryAfter := state.Redis.TTL(ctx, rlKey).Val()

		if bucket.Global {
			w.Header().Set("X-Global-Ratelimit", "true")
		}

		w.Header().Set("Retry-After", strconv.FormatFloat(retryAfter.Seconds(), 'g', -1, 64))

		w.WriteHeader(http.StatusTooManyRequests)

		// Set ratelimit to expire in more time if not global
		if !bucket.Global {
			state.Redis.Expire(ctx, rlKey, retryAfter+2*time.Second)
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

// Public ratelimit handler
func Ratelimit(reqs int, t time.Duration, bucket moderatedBucket, w http.ResponseWriter, r *http.Request) bool {
	// Get ratelimit from redis
	var id string

	auth := r.Header.Get("Authorization")

	if auth != "" {
		if strings.HasPrefix(auth, "User ") {
			idCheck := utils.AuthCheck(auth, false)

			if idCheck == nil {
				// Bot does not exist, return
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("{\"message\":\"Invalid API token\",\"error\":true}"))
				return false
			}

			id = *idCheck
		} else {
			idCheck := utils.AuthCheck(auth, true)

			if idCheck == nil {
				// Bot does not exist, return
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("{\"message\":\"Invalid API token\",\"error\":true}"))
				return false
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

	if ok := bucketHandle(bucket, id, w, r); !ok {
		return false
	}

	return true
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

type Router interface {
	Routes(r *chi.Mux)
	Tag() (string, string)
}

func main() {
	// Add the base tags
	docs.AddTag("System", "These API endpoints are core basic system APIs")

	docs.AddSecuritySchema("User", "User-Auth", "Requires a user token. Usually must be prefixed with `User `. Note that both ``User-Auth`` and ``Authorization`` headers are supported")
	docs.AddSecuritySchema("Bot", "Bot-Auth", "Requires a bot token. Can be optionally prefixed. Note that both ``Bot-Auth`` and ``Authorization`` headers are supported")

	ctx = context.Background()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.CleanPath)
	r.Use(corsMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(30 * time.Second))

	if os.Getenv("MIGRATION") == "true" || os.Getenv("MIGRATION") == "1" {
		migration = true
		migrations.Migrate(ctx, state.Pool)
		os.Exit(0)
	}

	if !migrations.HasMigrated(ctx, state.Pool) {
		panic("Database has not been migrated, run popplio with the MIGRATION environment variable set to true to migrate")
	}

	routers := []Router{
		bots.Router{},
		users.Router{},
		auth.Router{},
		duser.Router{},
		packs.Router{},
		announcements.Router{},
		list.Router{},
		transcripts.Router{},
	}

	for _, router := range routers {
		name, desc := router.Tag()

		docs.AddTag(name, desc)

		router.Routes(r)
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

	// For compatibility with old API
	r.HandleFunc("/votes/{bot_id}/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			utils.ApiDefaultReturn(http.StatusMethodNotAllowed, w, r)
			return
		}

		var botId = chi.URLParam(r, "bot_id")
		var userId = chi.URLParam(r, "user_id")

		if r.Header.Get("Authorization") == "" {
			utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			id := utils.AuthCheck(r.Header.Get("Authorization"), true)

			if id == nil || *id != botId {
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}

			// To try and push users into new API, vote ban and approved check on GET is enforced on the old API
			var voteBannedState bool

			err := state.Pool.QueryRow(ctx, "SELECT vote_banned FROM bots WHERE bot_id = $1", id).Scan(&voteBannedState)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		var botType pgtype.Text

		state.Pool.QueryRow(ctx, "SELECT type FROM bots WHERE bot_id = $1", botId).Scan(&botType)

		if botType.String != "approved" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(state.NotApproved))
			return
		}

		voteParsed, err := utils.GetVoteData(ctx, userId, botId)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		var compatData = types.UserVoteCompat{
			HasVoted: voteParsed.HasVoted,
		}

		bytes, err := json.Marshal(compatData)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	})

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
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(b)
	})

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
	r.Post("/webhook-test", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload types.WebhookPost

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		err = json.Unmarshal(bodyBytes, &payload)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		if utils.IsNone(payload.URL) && utils.IsNone(payload.URL2) {
			utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		payload.Test = true // Always true

		var err1 error

		if !utils.IsNone(payload.URL) {
			err1 = webhooks.Send(payload)
		}

		var err2 error

		if !utils.IsNone(payload.URL2) {
			payload.URL = payload.URL2 // Test second enpdoint if it's not empty
			err2 = webhooks.Send(payload)
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
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write(bytes)
	})

	// Internal APIs
	r.Get("/_protozoa/notifications/info", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"public_key": os.Getenv("VAPID_PUBLIC_KEY"),
		}

		bytes, err := json.Marshal(data)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		w.Write(bytes)
	})

	r.HandleFunc("/_protozoa/notifications/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "DELETE" {
			utils.ApiDefaultReturn(http.StatusMethodNotAllowed, w, r)
			return
		}

		var id = chi.URLParam(r, "id")

		if id == "" {
			utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Fetch auth from postgresdb
		if r.Header.Get("Authorization") == "" {
			utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

			if authId == nil || *authId != id {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
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

			rows, err := state.Pool.Query(ctx, "SELECT endpoint, notif_id, created_at, ua FROM poppypaw WHERE id = $1", id)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			err = pgxscan.ScanAll(&subscriptionDb, rows)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			if len(subscriptionDb) == 0 {
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
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
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		} else {
			// Delete the notif
			if r.URL.Query().Get("notif_id") == "" {
				utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
				return
			}

			_, err := state.Pool.Exec(ctx, "DELETE FROM poppypaw WHERE id = $1 AND notif_id = $2", id, r.URL.Query().Get("notif_id"))

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.WriteHeader(http.StatusOK)
		}
	})

	r.Post("/_protozoa/notifications/{id}/sub", func(w http.ResponseWriter, r *http.Request) {
		var subscription struct {
			Auth     string `json:"auth"`
			P256dh   string `json:"p256dh"`
			Endpoint string `json:"endpoint"`
		}

		var id = chi.URLParam(r, "id")

		if id == "" {
			utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		defer r.Body.Close()

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		err = json.Unmarshal(bodyBytes, &subscription)

		if err != nil {
			log.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return
		}

		if subscription.Auth == "" || subscription.P256dh == "" {
			utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Fetch auth from postgresdb
		if r.Header.Get("Authorization") == "" {
			utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

			if authId == nil || *authId != id {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		// Store new subscription

		notifId := utils.RandString(512)

		ua := r.UserAgent()

		if ua == "" {
			ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"
		}

		state.Pool.Exec(ctx, "DELETE FROM poppypaw WHERE id = $1 AND endpoint = $2", id, subscription.Endpoint)

		state.Pool.Exec(
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

		w.Write([]byte(state.Success))
	})

	r.HandleFunc("/_protozoa/reminders/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" && r.Method != "GET" && r.Method != "DELETE" {
			utils.ApiDefaultReturn(http.StatusMethodNotAllowed, w, r)
			return
		}

		var id = chi.URLParam(r, "id")

		if id == "" {
			utils.ApiDefaultReturn(http.StatusBadRequest, w, r)
			return
		}

		// Fetch auth from postgresdb
		if r.Header.Get("Authorization") == "" {
			utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
			return
		} else {
			authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

			if authId == nil || *authId != id {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusUnauthorized, w, r)
				return
			}
		}

		if r.Method == "GET" {
			// Fetch reminder from postgresdb
			rows, err := state.Pool.Query(ctx, "SELECT "+silverpeltColsStr+" FROM silverpelt WHERE user_id = $1", id)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			var reminders []types.Reminder

			pgxscan.ScanAll(&reminders, rows)

			if err != nil {
				log.Error(err)
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			if len(reminders) == 0 {
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			for i, reminder := range reminders {
				// Try resolving the bot from discord API
				var resolvedBot types.ResolvedReminderBot
				bot, err := utils.GetDiscordUser(reminder.BotID)

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
				utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
				return
			}

			w.Write(bytes)
		} else {
			// Add subscription to collection
			var botId pgtype.Text

			err = state.Pool.QueryRow(ctx, "SELECT bot_id FROM bots WHERE (vanity = $1 OR bot_id = $1 OR name = $1)", r.URL.Query().Get("bot_id")).Scan(&botId)

			if err != nil || botId.Status != pgtype.Present || botId.String == "" {
				log.Error("Error adding reminder: ", err)
				utils.ApiDefaultReturn(http.StatusNotFound, w, r)
				return
			}

			// Delete old
			state.Pool.Exec(ctx, "DELETE FROM silverpelt WHERE user_id = $1 AND bot_id = $2", id, botId.String)

			// Insert new
			if r.Method == "PUT" {
				_, err := state.Pool.Exec(ctx, "INSERT INTO silverpelt (user_id, bot_id) VALUES ($1, $2)", id, botId.String)

				if err != nil {
					log.Error("Error adding reminder: ", err)
					utils.ApiDefaultReturn(http.StatusNotFound, w, r)
					return
				}
			}

			w.Write([]byte(state.Success))
		}
	})

	// Load openapi here to avoid large marshalling in every request
	docs.DocumentMicroservices()

	openapi, err = json.Marshal(docs.GetSchema())

	if err != nil {
		panic(err)
	}

	adp := DummyAdapter{}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		utils.ApiDefaultReturn(http.StatusNotFound, w, r)
	})

	integrase.Prepare(adp, chiWrap{Router: r})

	http.ListenAndServe(":8081", r)
}
