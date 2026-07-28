package get_public_coupons

import (
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
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
		state.Logger.Error("Failed to fetch shop coupons list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	coupons, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ShopCoupon])

	if err != nil {
		state.Logger.Error("Failed to fetch shop coupons list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopCoupon]{
			Items: coupons,
		},
	}
}
