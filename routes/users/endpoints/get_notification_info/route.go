package get_notification_info

import (
	"net/http"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
		Path:        "/users/notifications/info",
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
