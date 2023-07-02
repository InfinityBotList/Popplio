package payments

import (
	"popplio/api"
	"popplio/routes/payments/endpoints/capture_paypal_order"
	"popplio/routes/payments/endpoints/create_paypal_order"
	"popplio/routes/payments/endpoints/create_stripe_checkout"
	"popplio/routes/payments/endpoints/get_paypal"
	"popplio/routes/payments/endpoints/get_premium_plans"
	"popplio/routes/payments/endpoints/get_stripe"
	"popplio/routes/payments/endpoints/handle_stripe_webhook"
	"popplio/routes/payments/endpoints/redeem_payment_offer"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const (
	tagName = "Payments"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to payments on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/payments/paypal",
		OpId:    "get_paypal",
		Method:  uapi.GET,
		Docs:    get_paypal.Docs,
		Handler: get_paypal.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/paypal",
		OpId:    "create_paypal_order",
		Method:  uapi.POST,
		Docs:    create_paypal_order.Docs,
		Handler: create_paypal_order.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "id",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/payments/paypal/capture/{ref_id}",
		OpId:    "capture_paypal_order",
		Method:  uapi.GET,
		Docs:    capture_paypal_order.Docs,
		Handler: capture_paypal_order.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/payments/stripe",
		OpId:    "get_stripe",
		Method:  uapi.GET,
		Docs:    get_stripe.Docs,
		Handler: get_stripe.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/stripe",
		OpId:    "create_stripe_checkout",
		Method:  uapi.POST,
		Docs:    create_stripe_checkout.Docs,
		Handler: create_stripe_checkout.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "id",
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/payments/stripe/webhook",
		OpId:    "handle_stripe_webhook",
		Method:  uapi.POST,
		Docs:    handle_stripe_webhook.Docs,
		Handler: handle_stripe_webhook.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/payments/premium/plans",
		OpId:    "get_premium_plans",
		Method:  uapi.GET,
		Docs:    get_premium_plans.Docs,
		Handler: get_premium_plans.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/redeem-payment-offer",
		OpId:    "redeem_payment_offer",
		Method:  uapi.POST,
		Docs:    redeem_payment_offer.Docs,
		Handler: redeem_payment_offer.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "id",
			},
		},
	}.Route(r)
}
