package capture_paypal_order

import (
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

type PaypalCaptureOrderReq struct {
	ID string `json:"id" validate:"required" msg:"ID is required."`
}

type PaypalOrderReq struct {
	ProductName string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID   string `json:"id" validate:"required" msg:"Product ID is required."`
	For         string `json:"for" validate:"required" msg:"For is required."`
}

var compiledMessages = api.CompileValidationErrors(PaypalCaptureOrderReq{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Capture Paypal Order",
		Description: "Captures a paypal order, giving any needed perks.",
		Req:         PaypalCaptureOrderReq{},
		Resp:        types.ApiError{},
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

	var payload PaypalCaptureOrderReq

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

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	captured, err := state.Paypal.CaptureOrder(d.Context, payload.ID, paypal.CaptureOrderRequest{})

	if err != nil {
		state.Logger.Error("At capture", err)
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: err.Error(),
			},
		}
	}

	if captured.Status == "VOIDED" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "Order is voided. Please contact support if you believe this is an error.",
			},
		}
	}

	if len(captured.PurchaseUnits) == 0 {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "No purchase units found. Please contact support if you believe this is an error.",
			},
		}
	}

	if len(captured.PurchaseUnits[0].Items) == 0 {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "No purchase items found. Please contact support if you believe this is an error.",
			},
		}
	}

	var productJson = captured.PurchaseUnits[0].Items[0].SKU

	var product PaypalOrderReq

	err = json.Unmarshal([]byte(productJson), &product)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = assets.GivePerks(d.Context, d.Auth.ID, assets.PerkData{
		For:         product.For,
		ProductName: product.ProductName,
		ProductID:   product.ProductID,
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

	return api.DefaultResponse(http.StatusNoContent)
}
