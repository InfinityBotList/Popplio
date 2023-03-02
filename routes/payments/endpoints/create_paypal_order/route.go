package create_paypal_order

import (
	"fmt"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/payments"
	"popplio/ratelimit"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/plutov/paypal/v4"
)

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
		MaxRequests: 2,
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

	var paymentData struct {
		ID      string
		Name    string
		Benefit string
		Price   float32
	}

	switch payload.ProductID {
	case "premium":
		for _, plan := range payments.Plans {
			if plan.ID == payload.ProductName {
				paymentData.ID = plan.ID
				paymentData.Name = plan.Name
				paymentData.Benefit = plan.Benefit
				paymentData.Price = plan.Price
				break
			}

			// Ensure the bot associated with For exists
			var count int64

			err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", payload.For).Scan(&count)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}

			if count == 0 {
				return api.HttpResponse{
					Json: types.ApiError{
						Error:   true,
						Message: "Invalid bot ID",
					},
				}
			}

			var typeStr string

			err = state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", payload.For).Scan(&typeStr)

			if err != nil {
				state.Logger.Error(err)
				return api.DefaultResponse(http.StatusInternalServerError)
			}

			if typeStr != "approved" && typeStr != "certified" {
				return api.HttpResponse{
					Json: types.ApiError{
						Error:   true,
						Message: "Bot is not approved or certified. You cannot purchase premium for this bot.",
					},
				}
			}
		}
	default:
		return api.HttpResponse{
			Json: types.ApiError{
				Error:   true,
				Message: "Invalid product ID",
			},
		}
	}

	paypalCli, err := state.CreatePaypalClient()

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	_, err = paypalCli.GetAccessToken(d.Context)

	if err != nil {
		state.Logger.Error("At accesstoken", err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	priceStr := fmt.Sprintf("%.2f", paymentData.Price)

	order, err := paypalCli.CreateOrder(d.Context, "CAPTURE", []paypal.PurchaseUnitRequest{
		{
			Description: paymentData.Name,
			CustomID:    paymentData.ID + "-" + d.Auth.ID + "-" + payload.For,
			Items: []paypal.Item{
				{
					Name:        paymentData.Name,
					Description: paymentData.Benefit,
					UnitAmount: &paypal.Money{
						Currency: "USD",
						Value:    priceStr,
					},
					Quantity: "1",
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
