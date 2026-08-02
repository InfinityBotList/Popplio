// Package get_sessions implements GET /{target_type}/{target_id}/sessions —
// "Get Sessions".
//
// Gets all session tokens of an entity
package get_sessions

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/validators"
	"strings"

	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

var (
	sessionCols = strings.Join(db.GetCols(types.Session{}), ", ")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Sessions",
		Description: "Gets all session tokens of an entity",
		Resp:        types.SessionList{},
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
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Missing target_id or target_type")
	}

	targetType = strings.TrimSuffix(targetType, "s")

	rows, err := state.Pool.Query(d.Context, "SELECT "+sessionCols+" FROM api_sessions WHERE target_id = $1 AND target_type = $2", targetId, targetType)

	if err != nil {
		return resp.Err("Error while getting user tokens", err)
	}

	defer rows.Close()

	tokens, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.Session])

	if errors.Is(err, pgx.ErrNoRows) {
		return resp.NotFound("No sessions found")
	}

	if err != nil {
		return resp.Err("Error while getting user sessions", err)
	}

	return uapi.HttpResponse{
		Json: types.SessionList{Sessions: tokens},
	}
}
