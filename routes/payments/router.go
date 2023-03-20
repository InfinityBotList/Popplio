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

	"github.com/go-chi/chi/v5"
)

const (
	tagName = "Payments"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to payments on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	api.Route{
		Pattern: "/payments/paypal",
		OpId:    "get_paypal",
		Method:  api.GET,
		Docs:    get_paypal.Docs,
		Handler: get_paypal.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/paypal",
		OpId:    "create_paypal_order",
		Method:  api.POST,
		Docs:    create_paypal_order.Docs,
		Handler: create_paypal_order.Route,
		Auth: []api.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "id",
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/paypal/capture",
		OpId:    "capture_paypal_order",
		Method:  api.POST,
		Docs:    capture_paypal_order.Docs,
		Handler: capture_paypal_order.Route,
		Auth: []api.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "id",
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/payments/stripe",
		OpId:    "get_stripe",
		Method:  api.GET,
		Docs:    get_stripe.Docs,
		Handler: get_stripe.Route,
	}.Route(r)

	api.Route{
		Pattern: "/users/{id}/stripe",
		OpId:    "create_stripe_checkout",
		Method:  api.POST,
		Docs:    create_stripe_checkout.Docs,
		Handler: create_stripe_checkout.Route,
		Auth: []api.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "id",
			},
		},
	}.Route(r)

	api.Route{
		Pattern: "/payments/stripe/webhook",
		OpId:    "handle_stripe_webhook",
		Method:  api.POST,
		Docs:    handle_stripe_webhook.Docs,
		Handler: handle_stripe_webhook.Route,
	}.Route(r)

	api.Route{
		Pattern: "/payments/premium/plans",
		OpId:    "get_premium_plans",
		Method:  api.GET,
		Docs:    get_premium_plans.Docs,
		Handler: get_premium_plans.Route,
	}.Route(r)
}
