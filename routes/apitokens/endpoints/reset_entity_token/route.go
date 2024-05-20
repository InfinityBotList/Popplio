package reset_entity_token

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Reset Entity Token",
		Description: "Reset the API token of an entity. You must have 'Reset Tokens' for the entity in the team. Returns the new token on success",
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

	if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: targetType, Perm: teams.PermissionResetAPITokens}) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to reset api tokens of this entity"},
		}
	}

	token := crypto.RandString(128)

	switch targetType {
	case "bot":
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET api_token = $1 WHERE bot_id = $2", token, targetId)

		if err != nil {
			state.Logger.Error("Error while resetting bot token", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetID", targetId))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	case "server":
		_, err = state.Pool.Exec(d.Context, "UPDATE servers SET api_token = $1 WHERE server_id = $2", token, targetId)

		if err != nil {
			state.Logger.Error("Error while resetting server token", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("targetID", targetId))
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
