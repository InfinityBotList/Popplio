package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/constants"
	"github.com/infinitybotlist/popplio/docs"
	"github.com/infinitybotlist/popplio/routes/announcements"
	"github.com/infinitybotlist/popplio/routes/bots"
	"github.com/infinitybotlist/popplio/routes/compat"
	"github.com/infinitybotlist/popplio/routes/diagnostics"
	"github.com/infinitybotlist/popplio/routes/duser"
	"github.com/infinitybotlist/popplio/routes/list"
	"github.com/infinitybotlist/popplio/routes/packs"
	"github.com/infinitybotlist/popplio/routes/special"
	"github.com/infinitybotlist/popplio/routes/transcripts"
	"github.com/infinitybotlist/popplio/routes/users"
	"github.com/infinitybotlist/popplio/state"

	integrase "github.com/MetroReviews/metro-integrase/lib"

	_ "embed"

	"github.com/infinitybotlist/eureka/zapchi"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

//go:embed docs/assets/ext.js
var extUnminified string

//go:embed docs/assets/docs.html
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

var (
	ctx context.Context

	docsJs  string
	openapi []byte
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit body to 10mb
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		if strings.HasSuffix(r.Header.Get("Origin"), "infinitybots.gg") || strings.HasPrefix(r.Header.Get("Origin"), "localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")

		if r.Method == "OPTIONS" {
			w.Write([]byte{})
			return
		}

		// Needed for docs
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

func main() {
	state.Logger.Info("Test\n\n")

	docs.AddSecuritySchema("User", "User-Auth", "Requires a user token. Usually must be prefixed with `User `. Note that both ``User-Auth`` and ``Authorization`` headers are supported")
	docs.AddSecuritySchema("Bot", "Bot-Auth", "Requires a bot token. Can be optionally prefixed. Note that both ``Bot-Auth`` and ``Authorization`` headers are supported")

	ctx = context.Background()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(
		middleware.Recoverer,
		middleware.RealIP,
		middleware.CleanPath,
		corsMiddleware,
		zapchi.Logger(state.Logger, "api"),
		middleware.Timeout(30*time.Second),
	)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use()

	routers := []api.APIRouter{
		// Use same order as routes folder
		announcements.Router{},
		bots.Router{},
		compat.Router{},
		duser.Router{},
		list.Router{},
		packs.Router{},
		special.Router{},
		transcripts.Router{},
		users.Router{},
		diagnostics.Router{},
	}

	for _, router := range routers {
		name, desc := router.Tag()
		if name != "" {
			docs.AddTag(name, desc)
			api.CurrentTag = name
		} else {
			panic("Router tag name cannot be empty")
		}

		router.Routes(r)
	}

	r.Get("/openapi", func(w http.ResponseWriter, r *http.Request) {
		w.Write(openapi)
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(docsHTML))
	})

	// Load openapi here to avoid large marshalling in every request
	var err error
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

	err = http.ListenAndServe(":8081", r)

	if err != nil {
		fmt.Println(err)
	}
}
