package manage_app

import (
	"errors"
	"fmt"
	"net/http"
	"popplio/apps"
	"popplio/db"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"strings"

	"github.com/disgoorg/disgo/discord"
	kittycat "github.com/infinitybotlist/kittycat/go"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type ManageApp struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason" validate:"required,min=5,max=1000" msg:"Reason must be between 5 and 1000 characters long"`
}

var (
	compiledMessages = uapi.CompileValidationErrors(ManageApp{})
	appColsArr       = db.GetCols(types.AppResponse{})
	appCols          = strings.Join(appColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Staff: Manage Application",
		Description: "Approves or denies an application. Returns a 204 on success.",
		Req:         ManageApp{},
		Params: []docs.Parameter{
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusFailedDependency,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	permList, err := validators.GetUserStaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusFailedDependency,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	resolvedPerms := permList.Resolve()

	// Check if the user has the permission to view apps
	if !kittycat.HasPerm(resolvedPerms, kittycat.Permission{Namespace: "apps", Perm: "manage"}) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json: types.ApiError{
				Message: "You do not have permission to manage apps.",
			},
		}
	}

	var payload ManageApp

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	// Fetch app info such as the position from database
	appId := chi.URLParam(r, "app_id")

	row, err := state.Pool.Query(d.Context, "SELECT "+appCols+" FROM apps WHERE app_id = $1", appId)

	if err != nil {
		state.Logger.Error("Failed to fetch application info", zap.Error(err), zap.String("appId", appId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	app, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.AppResponse])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Failed to fetch application info", zap.Error(err), zap.String("appId", appId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if app.State != "pending" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This app is not pending approval",
			},
		}
	}

	position := apps.FindPosition(app.Position)

	if position == nil {
		// Delete the app from the database
		_, err = state.Pool.Exec(d.Context, "DELETE FROM apps WHERE app_id = $1", appId)

		if err != nil {
			state.Logger.Error("Failed to delete app", zap.Error(err), zap.String("appId", appId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json: types.ApiError{
				Message: "This position doesn't exist and so the app has been deleted.",
			},
		}
	}

	var embeds []discord.Embed

	if payload.Approved {
		if position.ReviewLogic != nil {
			err := position.ReviewLogic(d, app, payload.Reason, true)

			if err != nil {
				state.Logger.Error("Error running review logic", zap.Error(err), zap.String("appId", appId))
				return uapi.HttpResponse{
					Json: types.ApiError{
						Message: "Error: " + err.Error(),
					},
					Status: http.StatusBadRequest,
				}
			}
		}

		_, err = state.Pool.Exec(d.Context, "UPDATE apps SET state = 'approved', review_feedback = $2 WHERE app_id = $1", appId, payload.Reason)

		if err != nil {
			state.Logger.Error("Failed to update app", zap.Error(err), zap.String("appId", appId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		embeds = []discord.Embed{
			{
				Title:       "Application Approved",
				URL:         state.Config.Sites.Panel.Production() + "/panel/apps",
				Description: fmt.Sprintf("<@%s> has approved an application by <@%s> for the position of %s", d.Auth.ID, app.UserID, app.Position),
				Color:       0x00ff00,
				Fields: []discord.EmbedField{
					{
						Name:   "App ID",
						Value:  appId,
						Inline: validators.TruePtr,
					},
					{
						Name:   "User ID",
						Value:  app.UserID,
						Inline: validators.TruePtr,
					},
					{
						Name:   "Approved By",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: validators.TruePtr,
					},
					{
						Name:   "Position",
						Value:  app.Position,
						Inline: validators.TruePtr,
					},
					{
						Name:   "Feedback",
						Value:  payload.Reason,
						Inline: validators.TruePtr,
					},
				},
			},
		}
	} else {
		if position.ReviewLogic != nil {
			err := position.ReviewLogic(d, app, payload.Reason, false)

			if err != nil {
				state.Logger.Error("Error running review logic", zap.Error(err), zap.String("appId", appId))
				return uapi.HttpResponse{
					Json: types.ApiError{
						Message: "Error: " + err.Error(),
					},
					Status: http.StatusBadRequest,
				}
			}
		}

		_, err = state.Pool.Exec(d.Context, "UPDATE apps SET state = 'denied', review_feedback = $2 WHERE app_id = $1", appId, payload.Reason)

		if err != nil {
			state.Logger.Error("Failed to update app", zap.Error(err), zap.String("appId", appId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		embeds = []discord.Embed{
			{
				Title:       "Application Denied",
				URL:         state.Config.Sites.Panel.Production() + "/panel/apps",
				Description: fmt.Sprintf("<@%s> has denied an application by <@%s> for the position of %s", d.Auth.ID, app.UserID, app.Position),
				Color:       0xff0000,
				Fields: []discord.EmbedField{
					{
						Name:   "App ID",
						Value:  appId,
						Inline: validators.TruePtr,
					},
					{
						Name:   "User ID",
						Value:  app.UserID,
						Inline: validators.TruePtr,
					},
					{
						Name:   "Denied By",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: validators.TruePtr,
					},
					{
						Name:   "Position",
						Value:  app.Position,
						Inline: validators.TruePtr,
					},
					{
						Name:   "Reason",
						Value:  payload.Reason,
						Inline: validators.TruePtr,
					},
				},
			},
		}
	}

	// Send message to apps channel
	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.Apps, discord.MessageCreate{
		Embeds: embeds,
	})

	if err != nil {
		state.Logger.Error("Failed to send message to apps channel", zap.Error(err), zap.String("appId", appId))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
