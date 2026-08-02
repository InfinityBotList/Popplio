// Package revoke_session implements DELETE
// /{target_type}/{target_id}/sessions/{session_id} — "Revoke Session".
//
// Revokes a session of an entity based on session ID
package revoke_session

import (
	"errors"
	"net/http"
	"popplio/api/resp"

	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

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
		return resp.BadRequest("Missing target_id or target_type")
	}

	var count int64

	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM api_sessions WHERE target_type = $1 AND target_id = $2 AND id = $3", targetType, targetId, sessionId).Scan(&count)

	if errors.Is(err, pgx.ErrNoRows) {
		return resp.NotFound("No sessions found")
	}

	if err != nil {
		return resp.Err("Error while getting user session", err)
	}

	if count == 0 {
		return resp.NotFound("No sessions found")
	}

	_, err = state.Pool.Exec(d.Context, "DELETE FROM api_sessions WHERE id = $1 AND target_id = $2 AND target_type = $3", sessionId, targetId, targetType)

	if err != nil {
		return resp.Err("Error while revoking user session", err)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
