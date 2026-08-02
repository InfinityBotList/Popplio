// Package get_public_coupons implements GET /shop/public-coupons — "Get Shop
// Coupons".
//
// Gets the publicly viewable shop coupons on the list
package get_public_coupons

import (
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

var (
	// Shop coupons
	shopCouponsColsArr = db.GetCols(types.ShopCoupon{})
	shopCouponsCols    = strings.Join(shopCouponsColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Shop Coupons",
		Description: "Gets the publicly viewable shop coupons on the list",
		Resp:        types.ItemList[types.ShopCoupon]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+shopCouponsCols+" FROM shop_coupons WHERE public = true ORDER BY created_at DESC")

	if err != nil {
		return resp.Err("Failed to fetch shop coupons list [db fetch]", err)
	}

	defer rows.Close()

	coupons, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ShopCoupon])

	if err != nil {
		return resp.Err("Failed to fetch shop coupons list [db fetch]", err)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopCoupon]{
			Items: coupons,
		},
	}
}
