package get_webhooks

import (
	"errors"
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"strings"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
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
		Description: "Gets a list of webhooks of a specific entity (excluding the secret due to security concerns). **Requires the Get Webhooks permission**",
		Resp:        []types.Webhook{},
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The entity type to return webhooks for.",
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
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

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
