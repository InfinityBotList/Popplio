package handle_stripe_webhook

import (
	"fmt"
	"io"
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/stripe/stripe-go/v74/webhook"
	"golang.org/x/exp/slices"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Handle Stripe Webhook",
		Description: "Handles stripe payment webhooks. Not intended for public use and firewalled to only stripe IPs.",
		Resp:        types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	if state.StripeWebhSecret == "" {
		return api.HttpResponse{
			Status: http.StatusFailedDependency,
			Json: types.ApiError{
				Error:   true,
				Message: "Stripe webhooks are not configured yet! Please try again in a few moments?",
			},
		}
	}

	// Get request IP
	if !slices.Contains(state.StripeWebhIPList, r.RemoteAddr) {
		state.Logger.Error("IP " + r.RemoteAddr + " is not allowed to access this endpoint")
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Error:   true,
				Message: "You are not allowed to access this endpoint",
			},
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "Invalid request body",
			},
		}
	}

	// Pass the request body and Stripe-Signature header to ConstructEvent, along with the webhook signing key
	// You can find your endpoint's secret in your webhook settings
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), state.StripeWebhSecret)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "Invalid request body",
			},
		}
	}

	fmt.Println(event)

	return api.DefaultResponse(http.StatusNoContent)
}
