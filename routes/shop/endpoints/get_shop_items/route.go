// Package get_shop_items implements GET /shop/items — "Get Shop Items".
//
// Gets the publicly viewable shop items on the list
package get_shop_items

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
	// Shop items
	shopItemsColsArr = db.GetCols(types.ShopItem{})
	shopItemsCols    = strings.Join(shopItemsColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Shop Items",
		Description: "Gets the publicly viewable shop items on the list",
		Resp:        types.ItemList[types.ShopItem]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+shopItemsCols+" FROM shop_items ORDER BY created_at DESC")

	if err != nil {
		return resp.Err("Failed to fetch shop items list [db fetch]", err)
	}

	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ShopItem])

	if err != nil {
		return resp.Err("Failed to fetch shop items list [db fetch]", err)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopItem]{
			Items: items,
		},
	}
}
