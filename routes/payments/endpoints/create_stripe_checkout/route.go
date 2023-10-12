package create_stripe_checkout

import (
	"net/http"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"
	"strconv"
	"time"

	"github.com/infinitybotlist/eureka/uapi/ratelimit"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/checkout/session"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var compiledMessages = uapi.CompileValidationErrors(assets.PerkData{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Stripe Checkout",
		Description: "Creates a stripe checkout session returning the URL. Use this to initiate a new stripe order in your client.",
		Req:         assets.CreatePerkData{},
		Resp:        assets.RedirectUser{},
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 5,
		Bucket:      "payments",
	}.Limit(d.Context, r)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if limit.Exceeded {
		return uapi.HttpResponse{
			Json: types.ApiError{
				Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String(),
			},
			Headers: limit.Headers(),
			Status:  http.StatusTooManyRequests,
		}
	}

	var create assets.CreatePerkData

	hresp, ok := uapi.MarshalReqWithHeaders(r, &create, limit.Headers())

	if !ok {
		return hresp
	}

	payload := create.Parse(d.Auth.ID)

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	perk, err := assets.FindPerks(d.Context, payload)

	if err != nil {
		state.Logger.Error(err)
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Error: " + err.Error(),
			},
		}
	}

	customId, err := json.Marshal(payload)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				// Provide the exact Price ID (for example, pr_1234) of the product you want to sell
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(string(stripe.CurrencyUSD)),
					UnitAmount: stripe.Int64(int64(perk.Price * 100)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(perk.Name),
						Description: stripe.String("Gives " + perk.Benefit + " for " + payload.For + " with duration of " + strconv.Itoa(perk.TimePeriod) + " hours"),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(string(customId)),
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		//AutomaticTax:      &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)},
		SuccessURL: stripe.String(state.Config.Sites.Frontend.Parse() + "/payments/success"),
		CancelURL:  stripe.String(state.Config.Sites.Frontend.Parse() + "/payments/cancelled"),
	}

	order, err := session.New(params)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: assets.RedirectUser{
			URL: order.URL,
		},
	}
}
