package get_notification_info

import (
	"net/http"
	"os"

	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/docs"
	"github.com/infinitybotlist/popplio/types"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/notifications/info",
		OpId:        "get_notification_info",
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
