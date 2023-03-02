package create_paypal_order

import (
	"fmt"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/ratelimit"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"github.com/plutov/paypal/v4"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type PaypalOrderReq struct {
	ProductName string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID   string `json:"id" validate:"required" msg:"Product ID is required."`
	For         string `json:"for" validate:"required" msg:"For is required."`
}

var compiledMessages = api.CompileValidationErrors(PaypalOrderReq{})

type PaypalOrderID struct {
	OrderID string `json:"order_id"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Paypal Order",
		Description: "Creates a paypal order. Not intended for public use.",
		Req:         PaypalOrderReq{},
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

	var payload PaypalOrderReq

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

	perk, err := assets.FindPerks(d.Context, assets.PerkData{
		ProductName: payload.ProductName,
		ProductID:   payload.ProductID,
		For:         payload.For,
	})

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
