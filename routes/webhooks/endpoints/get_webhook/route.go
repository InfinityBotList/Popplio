package get_webhook

import (
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"strings"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

var (
	webhookColsArr = db.GetCols(types.Webhook{})
	webhookCols    = strings.Join(webhookColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhook",
		Description: "Gets non-sensitive webhook data of a specific entity. **Requires authentication**",
		Resp:        types.Webhook{},
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
				Description: "The target ID to return webhook data for",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The entity type to return webhook data for.",
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
		state.Logger.Error(err)
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error getting user perms: " + err.Error()},
		}
	}

	if !perms.Has(targetType, teams.PermissionGetWebhooks) {
		return uapi.HttpResponse{
			Status: http.StatusForbidden,
			Json:   types.ApiError{Message: "You do not have permission to fetch webhook data for this entity"},
		}
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+webhookCols+" FROM webhooks WHERE target_id = $1 AND target_type = $2", targetId, targetType)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	webhook, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Webhook])

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: webhook,
	}
}
