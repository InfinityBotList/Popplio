// Package get_apps_list implements GET /users/{user_id}/apps — "Get
// Application List".
//
// Gets all applications of the user returning a list of apps.
package get_apps_list

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5"
)

var (
	appColsArr = db.GetCols(types.AppResponse{})
	appCols    = strings.Join(appColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Application List",
		Description: "Gets all applications of the user returning a list of apps.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user to use.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AppListResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	row, err := state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps WHERE user_id = $1", d.Auth.ID)

	if err != nil {
		return resp.Err("Failed to fetch application list [db fetch]", err, zap.String("userId", d.Auth.ID))
	}

	app, err := pgx.CollectRows(row, pgx.RowToStructByName[types.AppResponse])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{
			Json: types.AppListResponse{
				Apps: []types.AppResponse{},
			},
		}
	}

	if err != nil {
		state.Logger.Error("Failed to fetch application list [collection]", zap.String("userId", d.Auth.ID), zap.Error(err))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	return uapi.HttpResponse{
		Json: types.AppListResponse{
			Apps: app,
		},
	}
}
