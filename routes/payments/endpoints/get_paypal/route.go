package get_paypal

import (
	"net/http"
	"popplio/state"
	"popplio/types"

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
	if state.Paypal == nil {
		return uapi.HttpResponse{
			Status: http.StatusServiceUnavailable,
			Json: types.ApiError{
				Message: "Paypal is currently not available as a payment option. Please contact support!",
			},
		}
	}
	return uapi.HttpResponse{
		Json: PaypalMeta{
			PaypalClientID: state.Config.Meta.PaypalClientID.Parse(),
		},
	}
}
