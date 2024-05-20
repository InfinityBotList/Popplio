package get_webhook_list

import (
	"errors"
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	kittycat "github.com/infinitybotlist/kittycat/go"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	webhookColsArr = db.GetCols(types.Webhook{})
	webhookCols    = strings.Join(webhookColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhooks",
		Description: "Gets a list of webhooks of a specific entity (excluding the secret due to security concerns). **Requires authentication**",
		Resp:        []types.Webhook{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID to return webhooks for",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The entity type to return webhooks for.",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := r.URL.Query().Get("target_type")
	targetId := chi.URLParam(r, "target_id")

	perms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, targetType, targetId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !kittycat.HasPerm(perms, kittycat.Permission{Namespace: targetType, Perm: teams.PermissionGetWebhooks}) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to fetch webhooks for this entity"},
		}
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+webhookCols+" FROM webhooks WHERE target_id = $1 AND target_type = $2", targetId, targetType)

	if err != nil {
		state.Logger.Error("Error while querying webhooks [db fetch]", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	webhook, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Webhook])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{
			Json: []types.Webhook{},
		}
	}

	if err != nil {
		state.Logger.Error("Error while querying webhooks [collect]", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: webhook,
	}
}
