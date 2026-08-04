// Package get_user_alerts implements GET /users/{id}/alerts — "Get User
// Alerts".
//
// Gets a users alerts.
package get_user_alerts

import (
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	alertCols    = db.GetCols(types.Alert{})
	alertColsStr = strings.Join(alertCols, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Alerts",
		Description: "Gets a users alerts.\n\nAll alerts are also sent via push notifications if the user has subscribed to them.",
		Resp:        types.PagedResult[types.AlertList]{},
		RespName:    "PagedResultAlertList",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

const perPage = 20

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Page must be an integer")
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rows, err := state.Pool.Query(d.Context, "SELECT "+alertColsStr+" FROM alerts WHERE user_id = $1 ORDER BY created_at DESC, priority ASC LIMIT $2 OFFSET $3", d.Auth.ID, limit, offset)

	if err != nil {
		return resp.Err("Error getting alerts [db]", err, zap.String("userID", d.Auth.ID), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	alerts, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Alert])

	if err != nil {
		return resp.Err("Error getting alerts [collect]", err, zap.String("userID", d.Auth.ID), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM alerts WHERE user_id = $1", d.Auth.ID).Scan(&count)

	if err != nil {
		return resp.Err("Error getting total alert count", err, zap.String("userID", d.Auth.ID), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	var unackedCount uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM alerts WHERE user_id = $1 AND acked = false", d.Auth.ID).Scan(&unackedCount)

	if err != nil {
		return resp.Err("Error getting total unacked alert count", err, zap.String("userID", d.Auth.ID), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	data := types.PagedResult[types.AlertList]{
		Count: count,
		Results: types.AlertList{
			UnackedCount: unackedCount,
			Alerts:       alerts,
		},
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
