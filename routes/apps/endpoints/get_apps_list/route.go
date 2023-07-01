package get_apps_list

import (
	"net/http"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
)

var (
	appColsArr = utils.GetCols(types.AppResponse{})
	appCols    = strings.Join(appColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Application List",
		Description: "Gets all applications that the user can access returning a list of apps.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user to use.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "full",
				Description: "Whether to return the full application list or not. Requires admin permissions.",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AppListResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var full = r.URL.Query().Get("full")

	if full != "true" && full != "false" {
		full = "false"
	}

	var app []types.AppResponse

	// Check if the user is an admin
	var admin bool

	err := state.Pool.QueryRow(d.Context, "SELECT admin FROM users WHERE user_id = $1", d.Auth.ID).Scan(&admin)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	// Full needs admin permissions
	if full == "true" && (!admin || d.Auth.Banned) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "Only admins may use the 'full' query parameter.",
			},
		}
	}

	var count int64

	if full == "true" {
		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM apps").Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	} else {
		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM apps WHERE user_id = $1", d.Auth.ID).Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	if count == 0 {
		return uapi.HttpResponse{
			Json: types.AppListResponse{
				Apps: []types.AppResponse{},
			},
		}
	}

	var row pgx.Rows
	if full == "true" {
		row, err = state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps")
	} else {
		row, err = state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps WHERE user_id = $1", d.Auth.ID)
	}

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanAll(&app, row)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if len(app) == 0 {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	return uapi.HttpResponse{
		Json: types.AppListResponse{
			Apps: app,
		},
	}
}
