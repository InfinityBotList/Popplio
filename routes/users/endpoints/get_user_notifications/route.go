package get_user_notifications

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	ua "github.com/mileusna/useragent"
	"go.uber.org/zap"
)

func Docs(tagName string) {
	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/users/{id}/notifications",
		OpId:        "get_user_notifications",
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
		Resp:     types.NotifGetList{},
		Tags:     []string{tagName},
		AuthType: []string{"User"},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var id = chi.URLParam(r, "id")

	if id == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusBadRequest)
		return
	}

	// Fetch auth from postgresdb
	if r.Header.Get("Authorization") == "" {
		d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
		return
	} else {
		authId := utils.AuthCheck(r.Header.Get("Authorization"), false)

		if authId == nil || *authId != id {
			d.Resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
			return
		}
	}

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
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	err = pgxscan.ScanAll(&subscriptionDb, rows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusInternalServerError)
		return
	}

	if len(subscriptionDb) == 0 {
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
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

	d.Resp <- types.HttpResponse{
		Json: sublist,
	}
}
