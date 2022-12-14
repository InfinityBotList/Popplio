package get_authorize_info

import (
	"net/http"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/types"
)

var clientInfo string

func init() {
	clientInfo = `{"client_id":"` + os.Getenv("CLIENT_ID") + `"}`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/authorize",
		OpId:        "get_authorize_info",
		Summary:     "Get Login Info",
		Description: "Gets the login info such as the client ID to use for the login.",
		Tags:        []string{api.CurrentTag},
		Resp:        types.AuthInfo{},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Data: clientInfo,
	}
}
