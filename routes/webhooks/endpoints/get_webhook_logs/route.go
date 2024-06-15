package get_webhook_logs

import (
	"net/http"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const perPage = 10

var (
	webhookLogColsArr = db.GetCols(types.WebhookLogEntry{})
	webhookLogCols    = strings.Join(webhookLogColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhook Logs",
		Description: "Gets webhook logs of a specific entity. Paginated to 10 at a time. **Requires authentication**",
		Resp:        types.PagedResult[[]types.WebhookLogEntry]{},
		RespName:    "PagedResultWebhookLogEntry",
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the entity",
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
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := r.URL.Query().Get("target_type")
	targetId := chi.URLParam(r, "target_id")

	page := r.URL.Query().Get("page")

	if page == "" {
		page = "1"
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Invalid page number"},
		}
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	// Fetch the logs
	rows, err := state.Pool.Query(d.Context, "SELECT "+webhookLogCols+" FROM webhook_logs WHERE target_id = $1 AND target_type = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4", targetId, targetType, limit, offset)

	if err != nil {
		state.Logger.Error("Error while querying webhook logs [db fetch]", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	webhooks, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.WebhookLogEntry])

	if err != nil {
		state.Logger.Error("Error while querying webhook logs [collect]", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	for i, webhook := range webhooks {
		webhooks[i].User, err = dovewing.GetUser(d.Context, webhook.UserID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Error while querying webhook logs [dovewing]", zap.Error(err), zap.String("userID", d.Auth.ID))
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM webhook_logs WHERE target_id = $1 AND target_type = $2", targetId, targetType).Scan(&count)

	if err != nil {
		state.Logger.Error("Error while querying webhook logs [db count]", zap.Error(err), zap.String("userID", d.Auth.ID))
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	data := types.PagedResult[[]types.WebhookLogEntry]{
		Count:   count,
		Results: webhooks,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
