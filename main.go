package main

import (
	"context"
	"crypto/sha512"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"popplio/constants"
	"popplio/docs"
	"popplio/migrations"
	"popplio/routes/announcements"
	"popplio/routes/auth"
	"popplio/routes/bots"
	"popplio/routes/compat"
	"popplio/routes/duser"
	"popplio/routes/list"
	"popplio/routes/packs"
	"popplio/routes/special"
	"popplio/routes/transcripts"
	"popplio/routes/users"
	"popplio/state"
	"popplio/utils"

	integrase "github.com/MetroReviews/metro-integrase/lib"
	jsoniter "github.com/json-iterator/go"

	_ "embed"

	"popplio/zapchi"

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
	ctx context.Context

	docsJs  string
	openapi []byte

	// Default global ratelimit handler
	defaultGlobalBucket = moderatedBucket{BucketName: "global", Requests: 500, Time: 2 * time.Minute}
)

func bucketHandle(bucket moderatedBucket, id string, w http.ResponseWriter, r *http.Request) bool {
	rlKey := "rl:" + id + "-" + bucket.BucketName

	v := state.Redis.Get(r.Context(), rlKey).Val()

	if v == "" {
		v = "0"

		err := state.Redis.Set(ctx, rlKey, "0", bucket.Time).Err()

		if err != nil {
			state.Logger.Error(err)
			utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
			return false
		}
	}

	err := state.Redis.Incr(ctx, rlKey).Err()

	if err != nil {
		state.Logger.Error(err)
		utils.ApiDefaultReturn(http.StatusInternalServerError, w, r)
		return false
	}

	vInt, err := strconv.Atoi(v)

	if err != nil {
		state.Logger.Error(err)
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
	state.Logger.Info("Test\n\n")

	docs.AddSecuritySchema("User", "User-Auth", "Requires a user token. Usually must be prefixed with `User `. Note that both ``User-Auth`` and ``Authorization`` headers are supported")
	docs.AddSecuritySchema("Bot", "Bot-Auth", "Requires a bot token. Can be optionally prefixed. Note that both ``Bot-Auth`` and ``Authorization`` headers are supported")

	ctx = context.Background()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.CleanPath)
	r.Use(corsMiddleware)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(zapchi.Logger(state.Logger, "api"))

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(30 * time.Second))

	if os.Getenv("MIGRATION") == "true" || os.Getenv("MIGRATION") == "1" {
		state.Migration = true
		migrations.Migrate(ctx, state.Pool)
		os.Exit(0)
	}

	if !migrations.HasMigrated(ctx, state.Pool) {
		panic("Database has not been migrated, run popplio with the MIGRATION environment variable set to true to migrate")
	}

	routers := []Router{
		// Use same order as routes folder
		announcements.Router{},
		auth.Router{},
		bots.Router{},
		compat.Router{},
		duser.Router{},
		list.Router{},
		packs.Router{},
		special.Router{},
		transcripts.Router{},
		users.Router{},
	}

	for _, router := range routers {
		name, desc := router.Tag()

		if name != "" {
			docs.AddTag(name, desc)
		}

		router.Routes(r)
	}

	// Create base payloads before startup
	// Index
	var helloWorldB = Hello{
		Message: "Hello world from IBL API v6!",
		Docs:    docsSite,
		OurSite: mainSite,
		Status:  statusPage,
	}

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

	r.Get("/openapi", func(w http.ResponseWriter, r *http.Request) {
		w.Write(openapi)
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(docsHTML))
	})

	// Load openapi here to avoid large marshalling in every request
	docs.DocumentMicroservices()

	openapi, err = json.Marshal(docs.GetSchema())

	if err != nil {
		panic(err)
	}

	adp := DummyAdapter{}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(constants.NotFoundPage))
	})

	integrase.Prepare(adp, chiWrap{Router: r})

	http.ListenAndServe(":8081", r)
}
