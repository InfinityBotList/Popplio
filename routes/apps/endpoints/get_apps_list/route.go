package get_apps_list

import (
	"net/http"
	"popplio/db"
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
		state.Logger.Error("Failed to fetch application list [db fetch]", zap.String("userId", d.Auth.ID), zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	app, err := pgx.CollectRows(row, pgx.RowToStructByName[types.AppResponse])

	if err != nil {
		state.Logger.Error("Failed to fetch application list [collection]", zap.String("userId", d.Auth.ID), zap.Error(err))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	for i := range app {
		app[i].User, err = dovewing.GetUser(d.Context, app[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to fetch application list [user fetch]", zap.String("userId", app[i].UserID), zap.Error(err))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return uapi.HttpResponse{
		Json: types.AppListResponse{
			Apps: app,
		},
	}
}
