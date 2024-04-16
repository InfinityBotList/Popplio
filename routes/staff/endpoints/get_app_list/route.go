package get_app_list

import (
	"errors"
	"net/http"
	"popplio/db"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"
	"popplio/validators/kittycat/ext"
	"popplio/validators/kittycat/perms"
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
		return uapi.HttpResponse{
			Status: http.StatusFailedDependency,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	permList, err := ext.GetUserStaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusFailedDependency,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	resolvedPerms := permList.Resolve()

	// Check if the user has the permission to view apps
	if !perms.HasPerm(resolvedPerms, "apps.view") {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You do not have permission to view apps.",
			},
		}
	}

	userId := r.URL.Query().Get("user_id")

	var row pgx.Rows
	if userId != "" {
		row, err = state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps WHERE user_id = $1 ORDER BY created_at DESC", userId)
	} else {
		row, err = state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps ORDER BY created_at DESC")
	}

	if err != nil {
		state.Logger.Error("Failed to fetch application list [db fetch]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
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
		state.Logger.Error("Failed to fetch application list [collection]", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range app {
		user, err := dovewing.GetUser(d.Context, app[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to fetch application list [user fetch]", zap.String("userId", app[i].UserID), zap.Error(err))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		app[i].User = user
	}

	return uapi.HttpResponse{
		Json: types.AppListResponse{
			Apps: app,
		},
	}
}
