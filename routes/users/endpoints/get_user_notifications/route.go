package get_user_notifications

import (
	"net/http"
	"time"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	ua "github.com/mileusna/useragent"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
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

	var subscription []types.NotifGet

	var subscriptionDb []struct {
		Endpoint  string    `db:"endpoint"`
		NotifID   string    `db:"notif_id"`
		CreatedAt time.Time `db:"created_at"`
		UA        string    `db:"ua"`
	}

	rows, err := state.Pool.Query(d.Context, "SELECT endpoint, notif_id, created_at, ua FROM poppypaw WHERE user_id = $1", id)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&subscriptionDb, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if len(subscriptionDb) == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	for _, sub := range subscriptionDb {
		uaD := ua.Parse(sub.UA)
		state.Logger.With(
			zap.String("endpoint", sub.Endpoint),
			zap.String("notif_id", sub.NotifID),
			zap.Time("created_at", sub.CreatedAt),
			zap.String("ua", sub.UA),
			zap.Any("browser", uaD),
		).Info("Parsed UA")

		binfo := types.NotifBrowserInfo{
			OS:         uaD.OS,
			Browser:    uaD.Name,
			BrowserVer: uaD.Version,
			Mobile:     uaD.Mobile,
		}

		subscription = append(subscription, types.NotifGet{
			Endpoint:    sub.Endpoint,
			NotifID:     sub.NotifID,
			CreatedAt:   sub.CreatedAt,
			BrowserInfo: binfo,
		})
	}

	sublist := types.NotifGetList{
		Notifications: subscription,
	}

	return api.HttpResponse{
		Json: sublist,
	}
}
