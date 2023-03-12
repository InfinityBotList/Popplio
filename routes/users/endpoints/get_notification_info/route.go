package get_notification_info

import (
	"net/http"

	"popplio/api"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/doclib"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Notifications Info",
		Description: "Gets a users notifications",
		Resp:        types.NotificationInfo{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	data := types.NotificationInfo{
		PublicKey: state.Config.Notifications.VapidPublicKey,
	}

	return api.HttpResponse{
		Json: data,
	}
}
