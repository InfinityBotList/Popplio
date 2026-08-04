// Package get_app_list implements GET /staff/apps — "Staff: Get Application
// List".
//
// Gets all applications returning a list of apps.
package get_app_list

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
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
		Summary:     "Staff: Get Application List",
		Description: "Gets all applications returning a list of apps.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user to get the applications for. If not specified, all applications will be returned.",
				In:          "query",
				Required:    false,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AppListResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	staffPerms, err := perms.StaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	// Check if the user has the permission to view apps
	if !staffPerms.Has(perms.StaffViewApps) {
		return resp.Forbidden("You do not have permission to view apps.")
	}

	userId := r.URL.Query().Get("user_id")

	var row pgx.Rows
	if userId != "" {
		row, err = state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps WHERE user_id = $1 ORDER BY created_at DESC", userId)
	} else {
		row, err = state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps ORDER BY created_at DESC")
	}

	if err != nil {
		return resp.Err("Failed to fetch application list [db fetch]", err)
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
		return resp.Err("Failed to fetch application list [collection]", err)
	}

	for i := range app {
		user, err := dovewing.GetUser(d.Context, app[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Failed to fetch application list [user fetch]", err, zap.String("userId", app[i].UserID))
		}

		app[i].User = user
	}

	return uapi.HttpResponse{
		Json: types.AppListResponse{
			Apps: app,
		},
	}
}
