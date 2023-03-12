package get_paypal

import (
	"net/http"
	"popplio/api"
	"popplio/state"

	docs "github.com/infinitybotlist/doclib"
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Json: PaypalMeta{
			PaypalClientID: state.Config.Meta.PaypalClientID,
		},
	}
}
