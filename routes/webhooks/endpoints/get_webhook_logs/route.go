package get_webhook_logs

import (
	"net/http"
	"popplio/routes/webhooks/assets"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strconv"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const perPage = 10

var (
	webhookLogColsArr = utils.GetCols(types.WebhookLogEntry{})
	webhookLogCols    = strings.Join(webhookLogColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhook Logs",
		Description: "Gets logs of a specific entity. The entity type is determined by the auth type used. Paginated to 50 at a time. **Requires authentication**",
		Resp:        types.PagedResult[[]types.WebhookLogEntry]{},
		RespName:    "PagedResultWebhookLogEntry",
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
				Description: "The target ID to return webhook logs for",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The entity type to return logs for. Must be `bot` or `team` (other entity types coming soon)",
				Required:    true,
				In:          "query",
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
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	resp, ok := assets.CheckWebhookPermissions(
		d.Context,
		targetId,
		targetType,
		d.Auth.ID,
		assets.OpWebhookLogs,
	)

	if !ok {
		return resp
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+webhookLogCols+" FROM webhook_logs WHERE target_id = $1 AND target_type = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4", targetId, targetType, limit, offset)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	var webhooks []types.WebhookLogEntry

	err = pgxscan.ScanAll(&webhooks, rows)

	if err != nil {
		state.Logger.Error(err)
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if len(webhooks) == 0 {
		webhooks = []types.WebhookLogEntry{}
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM webhook_logs WHERE target_id = $1 AND target_type = $2", targetId, targetType).Scan(&count)

	if err != nil {
		state.Logger.Error(err)
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
