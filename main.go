package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"popplio/api"
	poplapps "popplio/apps"
	"popplio/constants"
	"popplio/docs"
	"popplio/notifications"
	"popplio/partners"
	"popplio/routes/announcements"
	"popplio/routes/apps"
	"popplio/routes/blogs"
	"popplio/routes/bots"
	"popplio/routes/diagnostics"
	"popplio/routes/duser"
	"popplio/routes/list"
	"popplio/routes/packs"
	"popplio/routes/reviews"
	"popplio/routes/special"
	"popplio/routes/staff"
	"popplio/routes/teams"
	"popplio/routes/tickets"
	"popplio/routes/users"
	"popplio/routes/votes"
	"popplio/state"

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
var extJsUnminified string

//go:embed docs/assets/ext.css
var extCssUnminified string

//go:embed docs/assets/docs.html
var docsHTML string

var (
	docsJs  string
	openapi []byte
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit body to 10mb
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		if r.Header.Get("Origin") == "" || strings.HasSuffix(r.Header.Get("Origin"), "spider.infinitybots.gg") {
			w.Header().Set("Docs-Site", "true")
			// Needed for docs
			if r.Header.Get("User-Auth") != "" {
				if strings.HasPrefix(r.Header.Get("User-Auth"), "User ") {
					r.Header.Set("Authorization", r.Header.Get("User-Auth"))
				} else {
					r.Header.Set("Authorization", "User "+r.Header.Get("User-Auth"))
				}
			} else if r.Header.Get("Bot-Auth") != "" {
				if strings.HasPrefix(r.Header.Get("Bot-Auth"), "Bot ") {
					r.Header.Set("Authorization", r.Header.Get("Bot-Auth"))
				}
				r.Header.Set("Authorization", "Bot "+r.Header.Get("Bot-Auth"))
			}
		}

		if strings.HasSuffix(r.Header.Get("Origin"), "infinitybots.gg") || strings.HasPrefix(r.Header.Get("Origin"), "localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Client")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")

		if r.Method == "OPTIONS" {
			w.Write([]byte{})
			return
		}

		w.Header().Set("Content-Type", "application/json")

		next.ServeHTTP(w, r)
	})
}

func main() {
	state.Setup()
	docs.Setup()

	m := minify.New()
	m.AddFunc("application/javascript", js.Minify)
	m.AddFunc("text/css", css.Minify)

	strWriter := &strings.Builder{}
	strReader := strings.NewReader(extCssUnminified)

	if err := m.Minify("text/css", strWriter, strReader); err != nil {
		panic(err)
	}

	extJsUnminified = strings.Replace(extJsUnminified, "[CSS]", "\""+strWriter.String()+"\"", 1)

	strWriter = &strings.Builder{}
	strReader = strings.NewReader(extJsUnminified)
	if err := m.Minify("application/javascript", strWriter, strReader); err != nil {
		panic(err)
	}

	docsJs = strWriter.String()

	docsHTML = strings.Replace(docsHTML, "[JS]", docsJs, 1)

	docs.AddSecuritySchema("User", "User-Auth", "Requires a user token. Should be prefixed with `User ` *in `Authorization` header*. **This docs page has an exemption to allow differenciation of schemas.**")
	docs.AddSecuritySchema("Bot", "Bot-Auth", "Requires a bot token. Should be prefixed with `Bot ` *in `Authorization` header*. **This docs page has an exemption to allow differenciation of schemas.**")

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
		blogs.Router{},
		bots.Router{},
		duser.Router{},
		list.Router{},
		packs.Router{},
		special.Router{},
		teams.Router{},
		tickets.Router{},
		users.Router{},
		votes.Router{},
		diagnostics.Router{},
		apps.Router{},
		reviews.Router{},
		staff.Router{},
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

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(constants.NotFoundPage))
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(constants.MethodNotAllowed))
	})

	poplapps.Setup()
	partners.Setup()

	go notifications.VrLoop()

	err = http.ListenAndServe(state.Config.Meta.Port, r)

	if err != nil {
		fmt.Println(err)
	}
}
