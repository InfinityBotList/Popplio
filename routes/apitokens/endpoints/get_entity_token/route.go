package get_entity_token

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Entity Token",
		Description: "Gets the API token of an entity. You must have 'View Existing Tokens' for the entity in the team.",
		Resp:        types.UserLogin{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := r.URL.Query().Get("target_type")

	if targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Both target_id and target_type must be specified"},
		}
	}

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

	if err != nil {
		state.Logger.Error("Error while getting entity perms", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetType", targetType), zap.String("targetID", targetId))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error: " + err.Error()},
		}
	}

	if !perms.Has(targetType, teams.PermissionViewAPITokens) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to view existing tokens of this entity"},
		}
	}

	var token string

	switch targetType {
	case "bot":
		err = state.Pool.QueryRow(d.Context, "SELECT api_token FROM bots WHERE bot_id = $1", targetId).Scan(&token)

		if err != nil {
			state.Logger.Error("Error while getting bot token", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetType", targetType), zap.String("targetID", targetId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	case "server":
		err = state.Pool.QueryRow(d.Context, "SELECT api_token FROM servers WHERE server_id = $1", targetId).Scan(&token)

		if err != nil {
			state.Logger.Error("Error while getting server token", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetType", targetType), zap.String("targetID", targetId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "Target type not implemented"},
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   types.UserLogin{UserID: targetId, Token: token},
	}
}
