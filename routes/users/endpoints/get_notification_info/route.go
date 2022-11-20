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
		OpId:        "get_user_notifications",
		Summary:     "Get User Notifications",
		Description: "Gets a users notifications",
		Resp:        types.NotificationInfo{},
		Tags:        []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	data := types.NotificationInfo{
		PublicKey: os.Getenv("VAPID_PUBLIC_KEY"),
	}

	d.Resp <- types.HttpResponse{
		Json: data,
	}
}
