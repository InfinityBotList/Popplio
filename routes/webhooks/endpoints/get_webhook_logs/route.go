package get_webhook_logs

import (
	"net/http"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/utils"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

const perPage = 50

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhook Logs",
		Description: "Gets logs of a specific entity. The entity type is determined by the auth type used. Paginated to 50 at a time. **Requires authentication**",
		Resp:        types.PagedResult[types.WebhookLogEntry]{},
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
				Json:   types.ApiError{Message: "You do not have permission to get bot webhook logs", Error: true},
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
				Json:   types.ApiError{Message: "You are not a member of this team", Error: true},
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
				Json:   types.ApiError{Message: "You do not have permission to get team webhook logs", Error: true},
			}
		}

	default:
		return uapi.HttpResponse{
			Status: http.StatusNotImplemented,
			Json:   types.ApiError{Message: "This entity type is not supported yet", Error: true},
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusNotImplemented,
		Json:   types.ApiError{Message: "This endpoint is not implemented yet", Error: true},
	}
}
