package diagnostics

import (
	"encoding/json"
	"net/http"
	"popplio/docs"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
)

const (
	tagName    = "Diagnostics"
	docsSite   = "https://spider.infinitybotlist.com/docs"
	mainSite   = "https://infinitybotlist.com"
	statusPage = "https://status.botlist.site"
	apiBot     = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
)

type Hello struct {
	Message   string `json:"message"`
	Docs      string `json:"docs"`
	OurSite   string `json:"our_site"`
	Status    string `json:"status"`
	APIBotURL string `json:"api_bot_url"`
}

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints allow diagnosing potential connection issues to our API."
}

func (b Router) Routes(r *chi.Mux) {
	// This is done here to avoid constant remarshalling
	var helloWorldB = Hello{
		Message:   "Hello world from IBL API v6!",
		Docs:      docsSite,
		OurSite:   mainSite,
		Status:    statusPage,
		APIBotURL: apiBot,
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
		Tags:        []string{tagName},
		Resp:        helloWorldB,
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := make(chan types.HttpResponse)
		go func() {
			resp <- types.HttpResponse{
				Bytes: helloWorld,
			}
		}()

		utils.Respond(ctx, w, resp)
	})
}
