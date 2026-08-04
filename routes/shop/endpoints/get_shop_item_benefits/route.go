// Package get_shop_item_benefits implements GET /shop/item-benefits — "Get
// Shop Items".
//
// Gets the publicly viewable shop items on the list
package get_shop_item_benefits

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
	// Shop item benefits
	shopItemBenefitsColsArr = db.GetCols(types.ShopItemBenefit{})
	shopItemBenefitsCols    = strings.Join(shopItemBenefitsColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Shop Items",
		Description: "Gets the publicly viewable shop items on the list",
		Resp:        types.ItemList[types.ShopItemBenefit]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+shopItemBenefitsCols+" FROM shop_item_benefits ORDER BY created_at DESC")

	if err != nil {
		return resp.Err("Failed to fetch shop item benefits list [db fetch]", err)
	}

	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ShopItemBenefit])

	if err != nil {
		return resp.Err("Failed to fetch shop item benefits list [db fetch]", err)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopItemBenefit]{
			Items: items,
		},
	}
}
