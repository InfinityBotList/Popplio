package capture_paypal_order

import (
	"net/http"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/plutov/paypal/v4"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Capture Paypal Order",
		Description: "Captures a paypal order, giving any needed perks. This is an internal endpoint.",
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "ref_id",
				Description: "The reference ID of the order returned during paypals redirect back to us",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if state.Paypal == nil {
		return uapi.HttpResponse{
			Status: http.StatusServiceUnavailable,
			Json: types.ApiError{
				Message: "Paypal is currently not available as a payment option. Please contact support!",
			},
		}
	}

	refId := chi.URLParam(r, "ref_id")

	if refId == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Missing ref_id",
			},
		}
	}

	// Get order ID from redis
	orderIdRedis := state.Redis.Get(d.Context, "paypal:"+refId)

	if orderIdRedis.Err() != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid ref_id. Please contact support if you believe this is an error.",
			},
		}
	}

	orderId := orderIdRedis.Val()

	captured, err := state.Paypal.CaptureOrder(d.Context, orderId, paypal.CaptureOrderRequest{})

	if err != nil {
		state.Logger.Error("Failed to capture paypal order", zap.Error(err), zap.String("order_id", orderId))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "Failed to capture paypal order: " + err.Error(),
			},
		}
	}

	if captured.Status == "VOIDED" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Order is voided. Please contact support if you believe this is an error.",
			},
		}
	}

	if len(captured.PurchaseUnits) == 0 {
		// Refund the order
		_, err = state.Paypal.RefundCapture(d.Context, orderId, paypal.RefundCaptureRequest{})

		if err != nil {
			state.Logger.Error("Failed to refund order [len(captured.PurchaseUnits) == 0]", zap.Error(err), zap.String("order_id", orderId))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json: types.ApiError{
					Message: "Failed to refund order.",
				},
			}
		}

		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "No purchase units found. Please contact support if you believe this is an error.",
			},
		}
	}

	if len(captured.PurchaseUnits[0].Items) == 0 {
		// Refund the order
		_, err = state.Paypal.RefundCapture(d.Context, orderId, paypal.RefundCaptureRequest{})

		if err != nil {
			state.Logger.Error("Failed to refund order [captured.PurchaseUnits[0].Items == 0]", zap.Error(err), zap.String("order_id", orderId))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json: types.ApiError{
					Message: "Failed to refund order.",
				},
			}
		}

		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "No purchase items found. Please contact support if you believe this is an error.",
			},
		}
	}

	var productJson = captured.PurchaseUnits[0].Items[0].SKU

	var product assets.PerkData

	err = jsonimpl.Unmarshal([]byte(productJson), &product)

	if err != nil {
		// Refund the order
		_, err = state.Paypal.RefundCapture(d.Context, orderId, paypal.RefundCaptureRequest{})

		if err != nil {
			state.Logger.Error("Failed to refund order [jsonimpl.Unmarshal]", zap.Error(err), zap.String("order_id", orderId))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json: types.ApiError{
					Message: "Failed to refund order.",
				},
			}
		}

		state.Logger.Error("Failed to unmarshal product json", zap.Error(err), zap.String("json", productJson))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = assets.GivePerks(d.Context, product)

	if err != nil {
		// Refund the order
		_, err = state.Paypal.RefundCapture(d.Context, orderId, paypal.RefundCaptureRequest{})

		if err != nil {
			state.Logger.Error("Failed to refund order [GivePerks: err != nil]", zap.Error(err), zap.String("order_id", orderId))
			return uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json: types.ApiError{
					Message: "Failed to refund order.",
				},
			}
		}

		state.Logger.Error("Failed to give perks", zap.Error(err), zap.String("json", productJson))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Failed to give perks: " + err.Error(),
			},
		}
	}

	state.Redis.Del(d.Context, "paypal:"+refId)

	return uapi.HttpResponse{
		Redirect: state.Config.Sites.Frontend.Parse() + "/payments/success",
	}
}
