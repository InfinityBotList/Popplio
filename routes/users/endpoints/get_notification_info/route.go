package get_notification_info

import (
	"net/http"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/types"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/notifications/info",
		OpId:        "get_notifications",
		Summary:     "Get Notifications Info",
		Description: "Gets a users notifications",
		Resp:        types.NotificationInfo{},
		Tags:        []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	data := types.NotificationInfo{
		PublicKey: os.Getenv("VAPID_PUBLIC_KEY"),
	}

	return api.HttpResponse{
		Json: data,
	}
}
