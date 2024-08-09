package revoke_session

import (
	"errors"
	"net/http"

	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Revoke Session",
		Description: "Revokes a session of an entity based on session ID",
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The entity type to use",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID to use",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "session_id",
				Description: "The session ID to revoke",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))
	sessionId := chi.URLParam(r, "session_id")

	if targetId == "" || targetType == "" || sessionId == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Missing target_id or target_type"},
		}
	}

	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM api_sessions WHERE target_type = $1 AND target_id = $2 AND id = $3", d.Auth.TargetType, d.Auth.ID, sessionId).Scan(&count)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "No sessions found"},
		}
	}

	if err != nil {
		state.Logger.Error("Error while getting user session", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if count == 0 {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "No sessions found"},
		}
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM api_sessions WHERE id = $1 AND target_id = $2 AND target_type = $3", sessionId, targetId, targetType)

	if err != nil {
		state.Logger.Error("Error while revoking user session", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
