package get_stripe

import (
	"net/http"
	"popplio/api"
	"popplio/state"

	docs "github.com/infinitybotlist/doclib"
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

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	return api.HttpResponse{
		Json: StripeMeta{
			StripePublicKey: state.Config.Meta.StripePublicKey,
		},
	}
}
