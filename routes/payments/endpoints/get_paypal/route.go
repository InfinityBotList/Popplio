package get_paypal

import (
	"net/http"
	"popplio/state"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

type PaypalMeta struct {
	PaypalClientID string `json:"paypal_client_id"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Paypal",
		Description: "Gets the required info needed for paypal payments.",
		Resp:        PaypalMeta{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: PaypalMeta{
			PaypalClientID: state.Config.Meta.PaypalClientID,
		},
	}
}
