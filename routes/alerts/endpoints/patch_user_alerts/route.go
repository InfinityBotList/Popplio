package patch_user_alerts

import (
	"net/http"
	"popplio/state"
	"popplio/types"

	"github.com/go-playground/validator/v10"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

var compiledMessages = uapi.CompileValidationErrors(types.AlertPatch{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch User Alerts",
		Description: "Updates a set of user alerts with a given 'patch' to apply to the alert. Returns 204 on success",
		Req:         types.AlertPatch{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.AlertPatch

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		state.Logger.Error("Error while starting transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer tx.Rollback(d.Context)

	for _, patch := range payload.Patches {
		switch patch.Patch {
		case "ack":
			_, err = tx.Exec(d.Context, "UPDATE alerts SET acked = true WHERE user_id = $1 AND itag = $2", d.Auth.ID, patch.ITag)
		case "unack":
			_, err = tx.Exec(d.Context, "UPDATE alerts SET acked = false WHERE user_id = $1 AND itag = $2", d.Auth.ID, patch.ITag)
		case "delete":
			_, err = tx.Exec(d.Context, "DELETE FROM alerts WHERE user_id = $1 AND itag = $2", d.Auth.ID, patch.ITag)
		}

		if err != nil {
			state.Logger.Error("Error while patching user alerts", zap.Any("patch", patch), zap.String("userID", d.Auth.ID), zap.Error(err))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		state.Logger.Error("Error while committing transaction", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
