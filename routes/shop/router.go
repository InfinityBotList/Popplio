package shop

import (
	"popplio/routes/shop/endpoints/get_public_coupons"
	"popplio/routes/shop/endpoints/get_shop_item_benefits"
	"popplio/routes/shop/endpoints/get_shop_items"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/uapi"
)

const tagName = "Shop"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to the IBL shop."
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/shop/public-coupons",
		OpId:    "get_public_coupons",
		Method:  uapi.GET,
		Docs:    get_public_coupons.Docs,
		Handler: get_public_coupons.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/shop/items",
		OpId:    "get_shop_items",
		Method:  uapi.GET,
		Docs:    get_shop_items.Docs,
		Handler: get_shop_items.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/shop/item-benefits",
		OpId:    "get_shop_item_benefits",
		Method:  uapi.GET,
		Docs:    get_shop_item_benefits.Docs,
		Handler: get_shop_item_benefits.Route,
	}.Route(r)
}
