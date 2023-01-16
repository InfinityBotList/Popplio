package ping

import (
	"net/http"

	"popplio/api"
	"popplio/docs"
	"popplio/state"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Hello struct {
	Message   string `json:"message"`
	Docs      string `json:"docs"`
	OurSite   string `json:"our_site"`
	Status    string `json:"status"`
	APIBotURL string `json:"api_bot_url"`
}

const (
	mainSite   = "https://infinitybotlist.com"
	statusPage = "https://status.botlist.site"
	apiBot     = "https://discord.com/api/oauth2/authorize?client_id=818419115068751892&permissions=140898593856&scope=bot%20applications.commands"
)

var helloWorld []byte
var helloWorldB Hello

func Setup() {
	var docsSite string = state.Config.Sites.API + "/docs"

	// This is done here to avoid constant remarshalling
	helloWorldB = Hello{
		Message:   "Hello world from IBL API v6!",
		Docs:      docsSite,
		OurSite:   mainSite,
		Status:    statusPage,
		APIBotURL: apiBot,
	}

	var err error
	helloWorld, err = json.Marshal(helloWorldB)

	if err != nil {
		panic(err)
	}
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Ping Server",
		Description: "This is a simple ping endpoint to check if the API is online. It will return a simple JSON object with a message, docs link, our site link and status page link.",
		Resp:        helloWorldB,
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Bytes: helloWorld,
	}
}
