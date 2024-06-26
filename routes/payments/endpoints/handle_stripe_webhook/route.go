package handle_stripe_webhook

import (
	"io"
	"net/http"
	"popplio/notifications"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/webhook"
	"golang.org/x/exp/slices"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Handle Stripe Webhook",
		Description: "Handles stripe payment webhooks. Not intended for public use and firewalled to only stripe IPs.",
		Resp:        types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if state.StripeWebhSecret == "" {
		return uapi.HttpResponse{
			Status: http.StatusFailedDependency,
			Json: types.ApiError{
				Message: "Stripe webhooks are not configured yet! Please try again in a few moments?",
			},
		}
	}

	// Get request IP
	if !slices.Contains(state.StripeWebhIPList, r.RemoteAddr) {
		state.Logger.Error("IP is not allowed to access this endpoint", zap.String("ip", r.RemoteAddr))
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "IP is not allowed to access this endpoint",
			},
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		state.Logger.Error("Failed to read request body", zap.Error(err))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid request body",
			},
		}
	}

	// Pass the request body and Stripe-Signature header to ConstructEvent, along with the webhook signing key
	// You can find your endpoint's secret in your webhook settings
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), state.StripeWebhSecret)

	if err != nil {
		state.Logger.Error("Failed to construct event", zap.Error(err))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "Invalid request body",
			},
		}
	}

	var s stripe.CheckoutSession
	var failed bool

	switch event.Type {
	case "checkout.session.completed":
		err := jsonimpl.Unmarshal(event.Data.Raw, &s)
		if err != nil {
			state.Logger.Error("Failed to unmarshal event data", zap.Error(err))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if s.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
			state.Logger.Error("Payment status is not paid")
			return uapi.HttpResponse{
				Status: http.StatusOK,
				Data:   "Payment status is not paid yet!",
			}
		}

	case "checkout.session.async_payment_succeeded":
		err := jsonimpl.Unmarshal(event.Data.Raw, &s)
		if err != nil {
			state.Logger.Error("Failed to unmarshal event data", zap.Error(err))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

	case "checkout.session.async_payment_failed":
		var s stripe.CheckoutSession
		err := jsonimpl.Unmarshal(event.Data.Raw, &s)
		if err != nil {
			state.Logger.Error("Failed to unmarshal event data", zap.Error(err))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		failed = true

	default:
		state.Logger.Error("Unknown event type", zap.String("event", string(event.Type)))
		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Unknown event type: " + string(event.Type),
		}
	}

	// Fulfill the purchase...
	var payload assets.PerkData

	err = jsonimpl.Unmarshal([]byte(s.ClientReferenceID), &payload)

	if err != nil {
		state.Logger.Error("Failed to unmarshal client reference id", zap.Error(err))
		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Failed to unmarshal client reference id: " + err.Error(),
		}
	}

	if failed {
		state.Logger.Error("Payment failed for user " + payload.UserID)

		// Send in bg as this can take a while and we don't want to block the request
		go notifications.PushNotification(payload.UserID, types.Alert{
			Title:    "Payment Failed",
			Message:  "Your payment for \"" + payload.ProductName + "\" for " + payload.For + " has failed. Please contact our support team to learn more!",
			Type:     types.AlertTypeError,
			Priority: types.AlertPriorityHigh,
		})

		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Payment failed for user " + payload.UserID + " for product " + payload.ProductID + "( " + payload.ProductName + " )" + " for " + payload.For,
		}
	}

	go func() {
		state.Logger.Info("Giving perks", zap.Any("payload", payload))

		err = assets.GivePerks(d.Context, payload)

		if err != nil {
			// Warn user about it as refunding is costly
			state.Logger.Error("Failed to give perks", zap.Error(err), zap.Any("payload", payload))
			notifications.PushNotification(payload.UserID, types.Alert{
				Title:    "Perk Delivery Failed",
				Message:  "Your payment for \"" + payload.ProductName + "\" for " + payload.For + " has succeeded but couldn't be handled correctly. Please contact our support team IMMEDIATELY: " + err.Error(),
				Type:     types.AlertTypeError,
				Priority: types.AlertPriorityHigh,
			})
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
