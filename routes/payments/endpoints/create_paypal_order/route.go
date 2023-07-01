package create_paypal_order

import (
	"fmt"
	"net/http"
	"popplio/ratelimit"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"github.com/plutov/paypal/v4"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var compiledMessages = uapi.CompileValidationErrors(assets.PerkData{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Paypal Order",
		Description: "Creates a paypal order returning the URL. Use this to initiate a new paypal order in your client.",
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

	priceStr := fmt.Sprintf("%.2f", perk.Price)

	customId, err := json.Marshal(payload)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	refId := crypto.RandString(32) // Paypal is stupid and requires a refId

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
	}, &paypal.CreateOrderPayer{}, &paypal.ApplicationContext{
		ReturnURL: state.Config.Sites.API.Parse() + "/payments/paypal/capture/" + refId,
		CancelURL: state.Config.Sites.Frontend.Parse() + "/payments/cancelled",
	})

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var approvalLink string

	for _, link := range order.Links {
		if link.Rel == "approve" {
			approvalLink = link.Href
		}
	}

	if approvalLink == "" {
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Internal Error: Could not find approval link",
			},
		}
	}

	// Save the refId to redis, associated with the order ID
	err = state.Redis.Set(d.Context, "paypal:"+refId, order.ID, 8*time.Hour).Err()

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: assets.RedirectUser{
			URL: approvalLink,
		},
	}
}
