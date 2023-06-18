package get_user_alert_by_itag

import (
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

var (
	alertCols    = utils.GetCols(types.Alert{})
	alertColsStr = strings.Join(alertCols, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Alert By Itag",
		Description: "Gets a single user alert based on its `itag`. This returns an alertlist to aid with consistency.",
		Resp:        types.AlertList{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "itag",
				Description: "The itag of the alert",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	itag := chi.URLParam(r, "itag")

	rows, err := state.Pool.Query(d.Context, "SELECT "+alertColsStr+" FROM alerts WHERE user_id = $1 AND itag = $2", d.Auth.ID, itag)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var alert types.Alert

	err = pgxscan.ScanOne(&alert, rows)

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

	return uapi.HttpResponse{
		Json: types.AlertList{
			Alerts: []types.Alert{
				alert,
			},
			UnackedCount: unackedCount,
		},
	}
}
