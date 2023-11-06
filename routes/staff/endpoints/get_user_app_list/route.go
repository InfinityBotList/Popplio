package get_user_app_list

import (
	"net/http"
	"popplio/db"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
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
		Summary:     "Staff: Get User Application List",
		Description: "Gets all applications of a user returning a list of apps.",
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
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusFailedDependency,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	// Check if the user has the permission to view apps
	var admin bool

	err = state.Pool.QueryRow(d.Context, "SELECT admin FROM users WHERE user_id = $1", d.Auth.ID).Scan(&admin)

	if err != nil {
		state.Logger.Error("Failed to fetch user from database", zap.Error(err), zap.String("userId", d.Auth.ID))
		return uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Message: "An error occurred while fetching the user from the database.",
			},
		}
	}

	if !admin {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You do not have permission to view apps.",
			},
		}
	}

	userId := chi.URLParam(r, "user_id")
	row, err := state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps WHERE user_id = $1", userId)

	if err != nil {
		state.Logger.Error("Failed to fetch application list [db fetch]", zap.String("userId", userId), zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	app, err := pgx.CollectRows(row, pgx.RowToStructByName[types.AppResponse])

	if err != nil {
		state.Logger.Error("Failed to fetch application list [collection]", zap.String("userId", userId), zap.Error(err))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	return uapi.HttpResponse{
		Json: types.AppListResponse{
			Apps: app,
		},
	}
}
