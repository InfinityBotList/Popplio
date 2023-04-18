package get_stripe

import (
	"net/http"
	"popplio/state"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

type StripeMeta struct {
	StripePublicKey string `json:"stripe_public_key"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Stripe",
		Description: "Gets the required info needed for stripe payments.",
		Resp:        StripeMeta{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: StripeMeta{
			StripePublicKey: state.Config.Meta.StripePublicKey,
		},
	}
}
