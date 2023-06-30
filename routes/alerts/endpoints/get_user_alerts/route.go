package get_user_alerts

import (
	"net/http"
	"popplio/config"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strconv"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

var (
	alertCols    = utils.GetCols(types.Alert{})
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
	if config.CurrentEnv == config.CurrentEnvProd {
		return uapi.HttpResponse{
			Status: http.StatusUnsupportedMediaType,
			Json: types.ApiError{
				Error:   true,
				Message: "Alerts is currently being rewritten and as such is disabled.",
				Context: map[string]string{
					"try": "https://reedwhisker.infinitybots.gg",
				},
			},
		}
	}

	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rows, err := state.Pool.Query(d.Context, "SELECT "+alertColsStr+" FROM alerts WHERE user_id = $1 ORDER BY created_at DESC, priority ASC LIMIT $2 OFFSET $3", d.Auth.ID, limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var alerts []types.Alert

	err = pgxscan.ScanAll(&alerts, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if len(alerts) == 0 {
		alerts = []types.Alert{}
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM alerts WHERE user_id = $1", d.Auth.ID).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var unackedCount uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM alerts WHERE user_id = $1 AND acked = false", d.Auth.ID).Scan(&unackedCount)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
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
