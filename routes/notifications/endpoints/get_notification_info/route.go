package get_notification_info

import (
	"net/http"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Notifications Info",
		Description: "Gets info needed to subscribe to push notifications (VAPID public key)",
		Resp:        types.NotificationInfo{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	data := types.NotificationInfo{
		PublicKey: state.Config.Notifications.VapidPublicKey,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
