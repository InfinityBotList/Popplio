package ping

import (
	"net/http"

	"popplio/state"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
)

type Hello struct {
	Message     string `json:"message"`
	Docs        string `json:"docs"`
	FrontendURL string `json:"frontend_url"`
}

var helloWorld []byte
var helloWorldB Hello

func Setup() {
	var docsSite string = state.Config.Sites.API.Parse() + "/docs"

	// This is done here to avoid constant remarshalling
	helloWorldB = Hello{
		Message:     "Hello world from IBL API v6!",
		Docs:        docsSite,
		FrontendURL: state.Config.Sites.Frontend.Parse(),
	}

	var err error
	helloWorld, err = jsonimpl.Marshal(helloWorldB)

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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Bytes: helloWorld,
	}
}
