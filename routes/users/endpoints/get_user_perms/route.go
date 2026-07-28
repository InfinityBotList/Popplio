package get_user_perms

import (
	"errors"
	"net/http"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

var (
	userPermColsArr = db.GetCols(types.UserPerm{})
	userPermCols    = strings.Join(userPermColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Perms",
		Description: "Gets a users permissions by ID",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserPerm{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	row, err := state.Pool.Query(d.Context, "SELECT "+userPermCols+" FROM users WHERE user_id = $1", id)

	if err != nil {
		state.Logger.Error("Failed to get user perms", zap.Error(err), zap.String("user_id", id))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	up, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.UserPerm])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Failed to get user perms", zap.Error(err), zap.String("user_id", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	user, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to get user perms", zap.Error(err), zap.String("user_id", id))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	up.User = user

	// Fetch staff status
	var positions int

	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(positions) FROM staff_members WHERE user_id = $1", user.ID).Scan(&positions)

	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		state.Logger.Error("Error while getting staff status", zap.Error(err), zap.String("userID", user.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	up.Staff = positions > 0

	return uapi.HttpResponse{
		Json: up,
	}
}
