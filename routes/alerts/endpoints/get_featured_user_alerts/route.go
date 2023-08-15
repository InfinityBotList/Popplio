package get_featured_user_alerts

import (
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strconv"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
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
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	if ackedResCount > 20 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "acked_count must be less than or equal to 20",
			},
		}
	}

	unackedResCount, err := strconv.Atoi(r.URL.Query().Get("unacked_count"))

	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	if unackedResCount > 20 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "unacked_count must be less than or equal to 20",
			},
		}
	}

	ackedRows, err := state.Pool.Query(d.Context, "SELECT "+alertColsStr+" FROM alerts WHERE user_id = $1 AND acked = true ORDER BY created_at DESC, priority ASC LIMIT $2", d.Auth.ID, ackedResCount)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	ackedAlerts, err := pgx.CollectRows(ackedRows, pgx.RowToStructByName[types.Alert])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	unackedRows, err := state.Pool.Query(d.Context, "SELECT "+alertColsStr+" FROM alerts WHERE user_id = $1 AND acked = false ORDER BY created_at DESC, priority ASC LIMIT $2", d.Auth.ID, unackedResCount)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	unackedAlerts, err := pgx.CollectRows(unackedRows, pgx.RowToStructByName[types.Alert])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if len(unackedAlerts) == 0 {
		unackedAlerts = []types.Alert{}
	}

	var unackedCount uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM alerts WHERE user_id = $1 AND acked = false", d.Auth.ID).Scan(&unackedCount)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: types.FeaturedUserAlerts{
			UnackedCount: unackedCount,
			Unacked:      unackedAlerts,
			Acked:        ackedAlerts,
		},
	}
}
