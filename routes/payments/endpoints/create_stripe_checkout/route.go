package create_stripe_checkout

import (
	"fmt"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/ratelimit"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var compiledMessages = api.CompileValidationErrors(assets.PerkData{})

type StripeCheckout struct {
	URL string `json:"url"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Stripe Checkout",
		Description: "Creates a stripe checkout session returning the URL. Not intended for public use.",
		Req:         assets.PerkData{},
		Resp:        StripeCheckout{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 5,
		Bucket:      "payments",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	if !d.IsClient {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "This endpoint is not available for public use",
			},
		}
	}

	var payload assets.PerkData

	hresp, ok := api.MarshalReqWithHeaders(r, &payload, limit.Headers())

	if !ok {
		return hresp
	}

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	perk, err := assets.FindPerks(d.Context, payload)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "Error: " + err.Error(),
			},
		}
	}

	priceStr := fmt.Sprintf("%.2f", perk.Price)

	customId, err := json.Marshal(payload)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				// Provide the exact Price ID (for example, pr_1234) of the product you want to sell
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(string(stripe.CurrencyUSD)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(perk.Name),
						Description: stripe.String("Gives " + perk.Benefit + " for " + payload.For + " with duration of " + strconv.Itoa(perk.TimePeriod) + " hours"),
					},
				},
				Price:    stripe.String(priceStr),
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(string(customId)),
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		AutomaticTax:      &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)},
		SuccessURL:        stripe.String(state.Config.Sites.Frontend + "/stripe/success"),
		CancelURL:         stripe.String(state.Config.Sites.Frontend + "/stripe/cancel"),
	}

	order, err := session.New(params)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: StripeCheckout{
			URL: order.URL,
		},
	}
}
