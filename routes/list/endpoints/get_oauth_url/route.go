package get_oauth_url

import (
	"net/http"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Oauth2 URL",
		Description: "Gets the oauth2 url to redirect to. Primarily for externally managed clients (like iblcli)",
		Resp:        types.OauthMeta{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.OauthMeta{
			URL: "https://discord.com/api/oauth2/authorize?client_id=" + state.Config.DiscordAuth.ClientID + "&scope=identify%20guilds&response_type=code&redirect_uri=%REDIRECT_URL%",
		},
	}
}
