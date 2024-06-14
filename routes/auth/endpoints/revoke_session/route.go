package revoke_session

import (
	"errors"
	"net/http"
	"strings"

	"popplio/state"
	"popplio/types"

	"popplio/routes/auth/assets"

	"popplio/teams"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Revoke User Session",
		Description: "Revokes a session of a user based on session ID",
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
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := chi.URLParam(r, "target_type")

	if targetId == "" || targetType == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Missing target_id or target_type"},
		}
	}

	targetType = strings.TrimSuffix(targetType, "s")

	// Perform entity specific checks
	err := assets.AuthEntityPermCheck(
		d.Context,
		d.Auth,
		targetType,
		targetId,
		teams.PermissionRevokeSession,
	)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "Entity permission checks failed: " + err.Error()},
		}
	}

	sessionId := r.URL.Query().Get("session_id")

	if sessionId == "" {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Missing session_id"},
		}
	}

	var count int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM api_sessions WHERE user_id = $1 AND id = $2", d.Auth.ID, sessionId).Scan(&count)

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

	_, err = state.Pool.Exec(d.Context, "DELETE FROM api_sessions WHERE id = $1", sessionId)

	if err != nil {
		state.Logger.Error("Error while revoking user session", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
