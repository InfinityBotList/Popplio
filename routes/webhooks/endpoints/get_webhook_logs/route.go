package get_webhook_logs

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"
	"strconv"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const perPage = 50

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

	switch targetType {
	case "bot":
		var count int

		err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", targetId).Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return uapi.DefaultResponse(http.StatusNotFound)
		}

		perms, err := utils.GetUserBotPerms(d.Context, d.Auth.ID, targetId)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if !perms.Has(teams.TeamPermissionGetBotWebhookLogs) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to get bot webhook logs"},
			}
		}
	case "team":
		var count int

		err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM teams WHERE id = $1", targetId).Scan(&count)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if count == 0 {
			return uapi.DefaultResponse(http.StatusNotFound)
		}

		// Ensure manager is a member of the team
		var managerCount int

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND user_id = $2", targetId, d.Auth.ID).Scan(&managerCount)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if managerCount == 0 {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You are not a member of this team"},
			}
		}

		var managerPerms []types.TeamPermission
		err = state.Pool.QueryRow(d.Context, "SELECT perms FROM team_members WHERE team_id = $1 AND user_id = $2", targetId, d.Auth.ID).Scan(&managerPerms)

		if err != nil {
			state.Logger.Error(err)
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		mp := teams.NewPermissionManager(managerPerms)

		if !mp.Has(teams.TeamPermissionGetTeamWebhookLogs) {
			return uapi.HttpResponse{
				Status: http.StatusForbidden,
				Json:   types.ApiError{Message: "You do not have permission to get team webhook logs"},
			}
		}

	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "This entity type is not supported yet"},
		}
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
