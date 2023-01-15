package get_app

import (
	"net/http"
	"popplio/api"
	"popplio/apps"
	"popplio/docs"
	"popplio/state"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

var (
	appColsArr = utils.GetCols(apps.AppResponse{})
	appCols    = strings.Join(appColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Method:      "GET",
		Summary:     "Get Application",
		Description: "Gets an application. **Does not require authentication.**",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The ID of the application.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: apps.AppResponse{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	appId := chi.URLParam(r, "id")

	if appId == "" {
		return api.DefaultResponse(http.StatusBadRequest)
	}

	// First check count so we can avoid expensive DB calls
	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM apps WHERE app_id = $1", appId).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var app apps.AppResponse

	row, err := state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps WHERE app_id = $1", appId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanOne(&app, row)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	return api.HttpResponse{
		Json: app,
	}
}
