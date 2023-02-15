package manage_app

import (
	"net/http"
	"popplio/api"
	"popplio/apps"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type ManageApp struct {
	Approved bool   `json:"approved" validate:"required"`
	Reason   string `json:"reason" validate:"required,min=5,max=1000" msg:"Reason must be between 5 and 1000 characters long"`
}

var (
	compiledMessages = api.CompileValidationErrors(ManageApp{})
	appColsArr       = utils.GetCols(apps.AppResponse{})
	appCols          = strings.Join(appColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Manage Application",
		Description: "Approves or denies an application. **Is staff-only and requires the ``iblhdev`` or the ``hadmin`` permission(s)**. Returns a 204 on success.",
		Req:         ManageApp{},
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user to create the application for.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "app_id",
				Description: "The App ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	// Check if the user has the permission to approve/deny the app
	var iblhdev bool
	var hadmin bool

	err := state.Pool.QueryRow(d.Context, "SELECT iblhdev, hadmin FROM users WHERE user_id = $1", d.Auth.ID).Scan(&iblhdev, &hadmin)

	if err != nil {
		state.Logger.Error(err)
		return api.HttpResponse{
			Status: http.StatusInternalServerError,
			Json: types.ApiError{
				Error:   true,
				Message: "An error occurred while fetching the user from the database.",
			},
		}
	}

	if !iblhdev && !hadmin {
		return api.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Error:   true,
				Message: "You do not have permission to approve/deny apps.",
			},
		}
	}

	var payload ManageApp

	hresp, ok := api.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return api.ValidatorErrorResponse(compiledMessages, errors)
	}

	// Fetch app info such as the position from database
	appId := chi.URLParam(r, "id")

	// First check count so we can avoid expensive DB calls
	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM apps WHERE app_id = $1", appId).Scan(&count)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return api.DefaultResponse(http.StatusNotFound)
	}

	var app apps.AppResponse

	rows, err := state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps WHERE app_id = $1", appId)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = pgxscan.ScanOne(&app, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if app.State != "pending" {
		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "This app is not pending approval",
			},
		}
	}

	positionData, ok := apps.Apps[app.Position]

	if !ok {
		// Delete the app from the database
		_, err = state.Pool.Exec(d.Context, "DELETE FROM apps WHERE app_id = $1", appId)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}

		return api.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Error:   true,
				Message: "This position doesn't exist and so the app has been deleted.",
			},
		}
	}

	if payload.Approved {
		if positionData.ReviewLogic != nil {
			add, err := positionData.ReviewLogic(d, app, payload.Reason)

			if err != nil {
				state.Logger.Error(err)
				return api.HttpResponse{
					Json: types.ApiError{
						Error:   true,
						Message: "Error: " + err.Error(),
					},
					Status: http.StatusBadRequest,
				}
			}

			if !add {
				return api.DefaultResponse(http.StatusNoContent)
			}
		}

		_, err = state.Pool.Exec(d.Context, "UPDATE apps SET state = 'approved', review_feedback = $2 WHERE app_id = $1", appId, payload.Reason)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	} else {
		_, err = state.Pool.Exec(d.Context, "UPDATE apps SET state = 'denied', review_feedback = $2 WHERE app_id = $1", appId, payload.Reason)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return api.DefaultResponse(http.StatusNoContent)
}
