// Package get_featured_user_alerts implements GET
// /users/{id}/alerts/@featured — "Get Featured User Alerts".
//
// Gets the featured user alerts of the user.
package get_featured_user_alerts

import (
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strconv"
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
		Summary:     "Get Featured User Alerts",
		Description: "Gets the featured user alerts of the user.",
		Resp:        types.FeaturedUserAlerts{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "acked_count",
				Description: "The number of alerts to return that have been acknowledged.",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "unacked_count",
				Description: "The number of alerts to return that have not been acknowledged.",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	ackedResCount, err := strconv.Atoi(r.URL.Query().Get("acked_count"))

	if err != nil {
		return resp.BadRequest("acked_count must be an integer")
	}

	if ackedResCount > 20 {
		return resp.BadRequest("acked_count must be less than or equal to 20")
	}

	unackedResCount, err := strconv.Atoi(r.URL.Query().Get("unacked_count"))

	if err != nil {
		return resp.BadRequest("unacked_count must be an integer")
	}

	if unackedResCount > 20 {
		return resp.BadRequest("unacked_count must be less than or equal to 20")
	}

	ackedRows, err := state.Pool.Query(d.Context, "SELECT "+alertColsStr+" FROM alerts WHERE user_id = $1 AND acked = true ORDER BY created_at DESC, priority ASC LIMIT $2", d.Auth.ID, ackedResCount)

	if err != nil {
		return resp.Err("Error getting acked user alerts [query]", err, zap.String("userID", d.Auth.ID), zap.Int("ackedResCount", ackedResCount), zap.Int("unackedResCount", unackedResCount))
	}

	ackedAlerts, err := pgx.CollectRows(ackedRows, pgx.RowToStructByName[types.Alert])

	if err != nil {
		return resp.Err("Error getting acked user alerts [collect]", err, zap.String("userID", d.Auth.ID), zap.Int("ackedResCount", ackedResCount), zap.Int("unackedResCount", unackedResCount))
	}

	unackedRows, err := state.Pool.Query(d.Context, "SELECT "+alertColsStr+" FROM alerts WHERE user_id = $1 AND acked = false ORDER BY created_at DESC, priority ASC LIMIT $2", d.Auth.ID, unackedResCount)

	if err != nil {
		return resp.Err("Error getting unacked user alerts [query]", err, zap.String("userID", d.Auth.ID), zap.Int("ackedResCount", ackedResCount), zap.Int("unackedResCount", unackedResCount))
	}

	unackedAlerts, err := pgx.CollectRows(unackedRows, pgx.RowToStructByName[types.Alert])

	if err != nil {
		return resp.Err("Error getting unacked user alerts [collect]", err, zap.String("userID", d.Auth.ID), zap.Int("ackedResCount", ackedResCount), zap.Int("unackedResCount", unackedResCount))
	}

	if len(unackedAlerts) == 0 {
		unackedAlerts = []types.Alert{}
	}

	var unackedCount uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM alerts WHERE user_id = $1 AND acked = false", d.Auth.ID).Scan(&unackedCount)

	if err != nil {
		return resp.Err("Error getting unacked user alerts count", err, zap.String("userID", d.Auth.ID), zap.Int("ackedResCount", ackedResCount), zap.Int("unackedResCount", unackedResCount))
	}

	return uapi.HttpResponse{
		Json: types.FeaturedUserAlerts{
			UnackedCount: unackedCount,
			Unacked:      unackedAlerts,
			Acked:        ackedAlerts,
		},
	}
}
