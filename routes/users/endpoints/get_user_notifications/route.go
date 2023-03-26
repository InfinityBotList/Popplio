package get_user_notifications

import (
	"net/http"
	"strings"

	"popplio/api"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/doclib"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	ua "github.com/mileusna/useragent"
)

var (
	notifGetCols    = utils.GetCols(types.NotifGet{})
	notifGetColsStr = strings.Join(notifGetCols, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Notifications",
		Description: "Gets a users notifications",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.NotifGetList{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var id = chi.URLParam(r, "id")

	var notifications []types.NotifGet

	rows, err := state.Pool.Query(d.Context, "SELECT "+notifGetColsStr+" FROM poppypaw WHERE user_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&notifications, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if len(notifications) == 0 {
		notifications = []types.NotifGet{}
	}

	for i := range notifications {
		uaD := ua.Parse(notifications[i].UA)

		notifications[i].BrowserInfo = types.NotifBrowserInfo{
			OS:         uaD.OS,
			Browser:    uaD.Name,
			BrowserVer: uaD.Version,
			Mobile:     uaD.Mobile,
		}
	}

	sublist := types.NotifGetList{
		Notifications: notifications,
	}

	return api.HttpResponse{
		Json: sublist,
	}
}
