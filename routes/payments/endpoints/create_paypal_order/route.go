package create_paypal_order

import (
	"fmt"
	"net/http"
	"popplio/api"
	"popplio/ratelimit"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"
	"time"

	docs "github.com/infinitybotlist/doclib"

	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"github.com/plutov/paypal/v4"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var compiledMessages = api.CompileValidationErrors(assets.PerkData{})

type PaypalOrderID struct {
	OrderID string `json:"order_id"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Paypal Order",
		Description: "Creates a paypal order. Not intended for public use.",
		Req:         assets.PerkData{},
		Resp:        PaypalOrderID{},
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

	order, err := state.Paypal.CreateOrder(d.Context, "CAPTURE", []paypal.PurchaseUnitRequest{
		{
			Description: perk.Name,
			CustomID:    string(customId),
			Items: []paypal.Item{
				{
					Name:        perk.Name,
					Description: perk.Benefit,
					UnitAmount: &paypal.Money{
						Currency: "USD",
						Value:    priceStr,
					},
					Quantity: "1",
					SKU:      string(customId),
				},
			},
			Amount: &paypal.PurchaseUnitAmount{
				Currency: "USD",
				Value:    priceStr,
				Breakdown: &paypal.PurchaseUnitAmountBreakdown{
					ItemTotal: &paypal.Money{
						Currency: "USD",
						Value:    priceStr,
					},
				},
			},
		},
	}, &paypal.CreateOrderPayer{}, &paypal.ApplicationContext{})

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: PaypalOrderID{
			OrderID: order.ID,
		},
	}
}
