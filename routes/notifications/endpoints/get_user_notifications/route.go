package get_user_notifications

import (
	"net/http"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	ua "github.com/mileusna/useragent"
)

var (
	notifGetCols    = db.GetCols(types.NotifGet{})
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")

	rows, err := state.Pool.Query(d.Context, "SELECT "+notifGetColsStr+" FROM user_notifications WHERE user_id = $1", id)

	if err != nil {
		state.Logger.Error("Failed to get user notifications", zap.Error(err), zap.String("user_id", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	notifications, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.NotifGet])

	if err != nil {
		state.Logger.Error("Failed to get user notifications", zap.Error(err), zap.String("user_id", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
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

	return uapi.HttpResponse{
		Json: sublist,
	}
}
