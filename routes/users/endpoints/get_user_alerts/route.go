package get_user_alerts

import (
	"net/http"
	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	docs "github.com/infinitybotlist/doclib"
)

var (
	alertCols    = utils.GetCols(types.Alert{})
	alertColsStr = strings.Join(alertCols, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Alerts",
		Description: "Gets a users alerts.\n\nAll alerts are also sent via push notifications if the user has subscribed to them.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AlertList{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+alertColsStr+" FROM alerts WHERE user_id = $1", d.Auth.ID)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var alerts []types.Alert

	err = pgxscan.ScanAll(&alerts, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if len(alerts) == 0 {
		alerts = []types.Alert{}
	}

	return api.HttpResponse{
		Json: types.AlertList{
			Alerts: alerts,
		},
	}
}
