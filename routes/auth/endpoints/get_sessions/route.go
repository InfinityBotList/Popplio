package get_sessions

import (
	"errors"
	"net/http"
	"popplio/db"
	"popplio/teams"
	"strings"

	"popplio/state"
	"popplio/types"

	"popplio/routes/auth/assets"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

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
		teams.PermissionViewSession,
	)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "Entity permission checks failed: " + err.Error()},
		}
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+sessionCols+" FROM api_sessions WHERE target_id = $1 AND target_type = $2", targetId, targetType)

	if err != nil {
		state.Logger.Error("Error while getting user tokens", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	defer rows.Close()

	tokens, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.Session])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{
			Status: http.StatusNotFound,
			Json:   types.ApiError{Message: "No sessions found"},
		}
	}

	if err != nil {
		state.Logger.Error("Error while getting user sessions", zap.Error(err))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: types.SessionList{Sessions: tokens},
	}
}
