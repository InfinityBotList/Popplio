// Package create_paypal_order implements POST /users/{id}/paypal — "Create
// Paypal Order".
//
// Creates a paypal order returning the URL. Use this to initiate a new
// paypal order in your client.
package create_paypal_order

import (
	"fmt"
	"net/http"
	"popplio/api/resp"
	"popplio/routes/payments/assets"
	"popplio/state"
	"time"

	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/ratelimit"
	"go.uber.org/zap"

	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-playground/validator/v10"
	"github.com/plutov/paypal/v4"
)

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
	if state.Paypal == nil {
		return resp.Status(http.StatusServiceUnavailable, "Paypal is currently not available as a payment option. Please contact support!")
	}

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 2,
		Bucket:      "payments",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err, zap.String("bucket", "payments"))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
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
		state.Logger.Error("Error while finding perk", zap.Error(err), zap.Any("payload", payload), zap.String("user_id", d.Auth.ID))
		return resp.BadRequest("Error: " + err.Error())
	}

	priceStr := fmt.Sprintf("%.2f", perk.Price)

	customId, err := jsonimpl.Marshal(payload)

	if err != nil {
		return resp.Err("Error while marshalling payload", err, zap.Any("payload", payload), zap.String("user_id", d.Auth.ID))
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
	}, nil, &paypal.ApplicationContext{
		ReturnURL: state.Config.Sites.API.Parse() + "/payments/paypal/capture/" + refId,
		CancelURL: state.Config.Sites.Frontend.Parse() + "/payments/cancelled",
	})

	if err != nil {
		return resp.Err("Error while creating paypal order", err, zap.Any("payload", payload), zap.String("user_id", d.Auth.ID))
	}

	var approvalLink string

	for _, link := range order.Links {
		if link.Rel == "approve" {
			approvalLink = link.Href
		}
	}

	if approvalLink == "" {
		return resp.ErrBody("Internal Error: Could not find approval link", "Internal Error: Could not find approval link", nil)
	}

	// Save the refId to redis, associated with the order ID
	err = state.Redis.Set(d.Context, "paypal:"+refId, order.ID, 8*time.Hour).Err()

	if err != nil {
		return resp.Err("Error while saving refId to redis", err, zap.Any("payload", payload), zap.String("user_id", d.Auth.ID))
	}

	return uapi.HttpResponse{
		Json: assets.RedirectUser{
			URL: approvalLink,
		},
	}
}
