package get_shop_items

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
		state.Logger.Error("Failed to fetch shop items list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ShopItem])

	if err != nil {
		state.Logger.Error("Failed to fetch shop items list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopItem]{
			Items: items,
		},
	}
}
