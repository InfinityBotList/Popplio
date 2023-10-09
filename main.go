package main

import (
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"popplio/api"
	poplapps "popplio/apps"
	"popplio/constants"
	"popplio/notifications"
	"popplio/routes/alerts"
	"popplio/routes/apitokens"
	"popplio/routes/apps"
	"popplio/routes/assets"
	"popplio/routes/blogs"
	"popplio/routes/bots"
	"popplio/routes/diagnostics"
	"popplio/routes/list"
	notifrouter "popplio/routes/notifications"
	"popplio/routes/packs"
	"popplio/routes/payments"
	"popplio/routes/platform"
	"popplio/routes/reminders"
	"popplio/routes/reviews"
	"popplio/routes/servers"
	"popplio/routes/staff"
	"popplio/routes/teams"
	"popplio/routes/tickets"
	"popplio/routes/users"
	"popplio/routes/vanity"
	"popplio/routes/votes"
	"popplio/routes/webhooks"
	"popplio/stafftemplates"
	"popplio/state"
	"popplio/types"
	poplhooks "popplio/webhooks"

	"github.com/cloudflare/tableflip"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/infinitybotlist/eureka/zapchi"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "embed"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

//go:embed docs/docs.html
var docsHTML string

var openapi []byte

// Simple middleware to handle CORS
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit body to 10mb
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

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

		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "X-Client, Content-Type, Authorization")
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

	docs.DocsSetupData = &docs.SetupData{
		URL:         state.Config.Sites.API.Parse(),
		ErrorStruct: types.ApiError{},
		Info: docs.Info{
			Title:          "Infinity Bot List API",
			TermsOfService: "https://infinitybotlist.com/terms",
			Version:        "7.0",
			Description:    "",
			Contact: docs.Contact{
				Name:  "Infinity Bot List",
				URL:   "https://infinitybotlist.com",
				Email: "support@infinitybots.gg",
			},
			License: docs.License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
	}

	docs.Setup()
	poplhooks.Setup()

	docs.AddSecuritySchema("User", "User-Auth", "Requires a user token. Should be prefixed with `User ` in `Authorization` header.")
	docs.AddSecuritySchema("Bot", "Bot-Auth", "Requires a bot token. Should be prefixed with `Bot ` in `Authorization` header.")

	api.Setup()

	r := chi.NewRouter()

	r.Use(
		middleware.Recoverer,
		middleware.RealIP,
		middleware.CleanPath,
		corsMiddleware,
		zapchi.Logger(state.Logger, "api"),
		middleware.Timeout(30*time.Second),
	)

	routers := []uapi.APIRouter{
		// Use same order as routes folder
		alerts.Router{},
		apitokens.Router{},
		apps.Router{},
		assets.Router{},
		blogs.Router{},
		bots.Router{},
		diagnostics.Router{},
		list.Router{},
		notifrouter.Router{},
		packs.Router{},
		payments.Router{},
		platform.Router{},
		reminders.Router{},
		reviews.Router{},
		servers.Router{},
		staff.Router{},
		teams.Router{},
		tickets.Router{},
		users.Router{},
		vanity.Router{},
		votes.Router{},
		webhooks.Router{},
	}

	for _, router := range routers {
		name, desc := router.Tag()
		if name != "" {
			docs.AddTag(name, desc)
			uapi.CurrentTag = name
		} else {
			panic("Router tag name cannot be empty")
		}

		router.Routes(r)
	}

	r.Get("/openapi", func(w http.ResponseWriter, r *http.Request) {
		w.Write(openapi)
	})

	docsTempl := template.Must(template.New("docs").Parse(docsHTML))

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/popplio", http.StatusFound)
	})

	r.Get("/docs/{srv}", func(w http.ResponseWriter, r *http.Request) {
		var docMap = map[string]string{
			"popplio": "/openapi",
			"arcadia": "https://prod--panel-api.infinitybots.gg/openapi",
		}

		srv := chi.URLParam(r, "srv")

		if srv == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid service name"))
			return
		}

		v, ok := docMap[srv]

		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid service"))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		docsTempl.Execute(w, map[string]string{
			"url": v,
		})
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
	stafftemplates.Setup()

	go notifications.VrLoop()

	// If GOOS is windows, do normal http server
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		upg, _ := tableflip.New(tableflip.Options{})
		defer upg.Stop()

		go func() {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGHUP)
			for range sig {
				state.Logger.Info("Received SIGHUP, upgrading server")
				upg.Upgrade()
			}
		}()

		// Listen must be called before Ready
		ln, err := upg.Listen("tcp", state.Config.Meta.Port.Parse())

		if err != nil {
			state.Logger.Fatal(err)
		}

		defer ln.Close()

		server := http.Server{
			ReadTimeout: 30 * time.Second,
			Handler:     r,
		}

		go func() {
			err := server.Serve(ln)
			if err != http.ErrServerClosed {
				state.Logger.Error(err)
			}
		}()

		if err := upg.Ready(); err != nil {
			state.Logger.Fatal(err)
		}

		<-upg.Exit()
	} else {
		// Tableflip not supported
		state.Logger.Warn("Tableflip not supported on this platform, this is not a production-capable server.")
		err = http.ListenAndServe(state.Config.Meta.Port.Parse(), r)

		if err != nil {
			state.Logger.Fatal(err)
		}
	}
}
