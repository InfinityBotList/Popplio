package get_authorize_info

import (
	"net/http"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
)

var clientInfo string

func Setup() {
	clientInfo = `{"client_id":"` + state.Config.DiscordAuth.ClientID + `"}`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
		Summary:     "Get Login Info",
		Description: "Gets the login info such as the client ID to use for the login.",
		Resp:        types.AuthInfo{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Data: clientInfo,
	}
}
